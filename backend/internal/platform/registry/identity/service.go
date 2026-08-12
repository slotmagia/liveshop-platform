// Package identity owns local back-office credentials and refresh sessions.
// Production may replace the login endpoint with an external IdP, but the
// access-identity and realm contract remains the same.
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/lvtuopen-ai/kernel-go/accessidentity"
	"github.com/lvtuopen-ai/kernel-go/apperror"
)

var (
	ErrInvalidCredentials = apperror.New("platform.identity.invalid_credentials", "invalid username or password")
	ErrInvalidRefresh     = apperror.New("platform.identity.invalid_refresh", "refresh session is invalid or expired")
	ErrRefreshReuse       = apperror.New("platform.identity.refresh_reuse", "refresh token reuse detected; session revoked")
	ErrDisabled           = apperror.New("platform.identity.disabled", "account is disabled")
	ErrLocked             = apperror.New("platform.identity.locked", "account is temporarily locked after repeated failed logins")
)

type LoginInput struct {
	Realm      string `json:"realm"`
	AppID      int64  `json:"appId"`
	MerchantID int64  `json:"merchantId"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

type Principal struct {
	Realm      string `json:"realm"`
	AppID      int64  `json:"appId"`
	MerchantID int64  `json:"merchantId"`
	Subject    string `json:"subject"`
	Username   string `json:"username"`
}

type Result struct {
	AccessToken  string    `json:"accessToken"`
	ExpiresIn    int64     `json:"expiresIn"`
	RefreshToken string    `json:"-"`
	Principal    Principal `json:"principal"`
}

type Service struct {
	db         *sql.DB
	issuer     *accessidentity.Issuer
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func New(db *sql.DB, issuer *accessidentity.Issuer) (*Service, error) {
	if db == nil || issuer == nil {
		return nil, errors.New("identity database and issuer are required")
	}
	return &Service{db: db, issuer: issuer, accessTTL: 15 * time.Minute, refreshTTL: 7 * 24 * time.Hour, now: time.Now}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (Result, error) {
	input.Realm = strings.ToUpper(strings.TrimSpace(input.Realm))
	input.Username = strings.ToLower(strings.TrimSpace(input.Username))
	if !validLogin(input) {
		return Result{}, ErrInvalidCredentials
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()
	var principal Principal
	var passwordHash, status string
	var failedLoginCount int
	var lockedUntil sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT realm,app_id,merchant_id,subject,username,password_hash,status,failed_login_count,locked_until
		FROM platform_identity_account
		WHERE realm=$1 AND username=$2 AND ($1='PLATFORM' OR (app_id=$3 AND merchant_id=$4)) FOR UPDATE`, input.Realm, input.Username, input.AppID, input.MerchantID).
		Scan(&principal.Realm, &principal.AppID, &principal.MerchantID, &principal.Subject, &principal.Username, &passwordHash, &status, &failedLoginCount, &lockedUntil)
	if errors.Is(err, sql.ErrNoRows) {
		// Keep unknown-account timing close to a real bcrypt verification.
		_ = bcrypt.CompareHashAndPassword([]byte("$2b$12$GRl.9VYBu/dC7wJXY/JUJe/GAq9Ru3VdCFNWDm/kJp3tNzd/Cj4DK"), []byte(input.Password))
		return Result{}, ErrInvalidCredentials
	}
	if err != nil {
		return Result{}, err
	}
	now := s.now()
	if lockedUntil.Valid && lockedUntil.Time.After(now) {
		return Result{}, ErrLocked
	}
	if status != "ACTIVE" {
		return Result{}, ErrDisabled
	}
	if lockedUntil.Valid {
		failedLoginCount = 0
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)) != nil {
		failedLoginCount++
		var nextLock any
		locked := failedLoginCount >= 5
		if locked {
			nextLock = now.Add(15 * time.Minute)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE platform_identity_account SET failed_login_count=$1,locked_until=$2,updated_at=NOW()
			WHERE realm=$3 AND app_id=$4 AND merchant_id=$5 AND subject=$6`, failedLoginCount, nextLock, principal.Realm, principal.AppID, principal.MerchantID, principal.Subject); err != nil {
			return Result{}, err
		}
		if err := appendAuditResult(ctx, tx, principal, "identity.login_denied", "identity.account", principal.Realm+":"+principal.Subject, "DENIED", map[string]any{"failedLoginCount": failedLoginCount, "locked": locked}); err != nil {
			return Result{}, err
		}
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		if locked {
			return Result{}, ErrLocked
		}
		return Result{}, ErrInvalidCredentials
	}
	accessToken, err := s.issue(principal)
	if err != nil {
		return Result{}, err
	}
	refreshToken := randomToken(32)
	sessionID := randomToken(18)
	expires := now.Add(s.refreshTTL)
	if _, err = tx.ExecContext(ctx, `UPDATE platform_identity_account SET failed_login_count=0,locked_until=NULL,updated_at=CASE WHEN failed_login_count<>0 OR locked_until IS NOT NULL THEN NOW() ELSE updated_at END
		WHERE realm=$1 AND app_id=$2 AND merchant_id=$3 AND subject=$4`, principal.Realm, principal.AppID, principal.MerchantID, principal.Subject); err != nil {
		return Result{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO platform_identity_session(session_id,realm,app_id,merchant_id,subject,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, sessionID, principal.Realm, principal.AppID, principal.MerchantID, principal.Subject, expires); err != nil {
		return Result{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO platform_identity_refresh_token(token_hash,session_id,status,issued_at,expires_at) VALUES($1,$2,'ACTIVE',$3,$4)`, tokenHash(refreshToken), sessionID, now, expires); err != nil {
		return Result{}, err
	}
	if err = appendAudit(ctx, tx, principal, "identity.login", "identity.session", sessionID, map[string]any{"username": principal.Username}); err != nil {
		return Result{}, err
	}
	if err = tx.Commit(); err != nil {
		return Result{}, err
	}
	return Result{AccessToken: accessToken, ExpiresIn: int64(s.accessTTL.Seconds()), RefreshToken: refreshToken, Principal: principal}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Result, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return Result{}, ErrInvalidRefresh
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()
	var principal Principal
	var sessionID, tokenStatus, accountStatus string
	var tokenExpires, sessionExpires time.Time
	err = tx.QueryRowContext(ctx, `SELECT s.session_id,t.status,t.expires_at,s.expires_at,a.realm,a.app_id,a.merchant_id,a.subject,a.username,a.status
        FROM platform_identity_refresh_token t
        JOIN platform_identity_session s ON s.session_id=t.session_id
        JOIN platform_identity_account a ON a.realm=s.realm AND a.app_id=s.app_id AND a.merchant_id=s.merchant_id AND a.subject=s.subject
        WHERE t.token_hash=$1 FOR UPDATE OF t,s`, tokenHash(refreshToken)).
		Scan(&sessionID, &tokenStatus, &tokenExpires, &sessionExpires, &principal.Realm, &principal.AppID, &principal.MerchantID, &principal.Subject, &principal.Username, &accountStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrInvalidRefresh
	}
	if err != nil {
		return Result{}, err
	}
	now := s.now()
	if tokenStatus == "USED" {
		if _, err = tx.ExecContext(ctx, `UPDATE platform_identity_session SET revoked_at=$1,revoke_reason='REFRESH_REUSE' WHERE session_id=$2 AND revoked_at IS NULL`, now, sessionID); err != nil {
			return Result{}, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE platform_identity_refresh_token SET status='REVOKED' WHERE session_id=$1 AND status='ACTIVE'`, sessionID); err != nil {
			return Result{}, err
		}
		if err = appendAudit(ctx, tx, principal, "identity.refresh_reuse", "identity.session", sessionID, nil); err != nil {
			return Result{}, err
		}
		if err = tx.Commit(); err != nil {
			return Result{}, err
		}
		return Result{}, ErrRefreshReuse
	}
	var revokedAt sql.NullTime
	if err = tx.QueryRowContext(ctx, `SELECT revoked_at FROM platform_identity_session WHERE session_id=$1 FOR UPDATE`, sessionID).Scan(&revokedAt); err != nil {
		return Result{}, err
	}
	if tokenStatus != "ACTIVE" || revokedAt.Valid || !tokenExpires.After(now) || !sessionExpires.After(now) {
		return Result{}, ErrInvalidRefresh
	}
	if accountStatus != "ACTIVE" {
		return Result{}, ErrDisabled
	}
	accessToken, err := s.issue(principal)
	if err != nil {
		return Result{}, err
	}
	next := randomToken(32)
	if _, err = tx.ExecContext(ctx, `UPDATE platform_identity_refresh_token SET status='USED',used_at=$1 WHERE token_hash=$2 AND status='ACTIVE'`, now, tokenHash(refreshToken)); err != nil {
		return Result{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO platform_identity_refresh_token(token_hash,session_id,status,issued_at,expires_at) VALUES($1,$2,'ACTIVE',$3,$4)`, tokenHash(next), sessionID, now, sessionExpires); err != nil {
		return Result{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE platform_identity_session SET last_refreshed_at=$1 WHERE session_id=$2`, now, sessionID); err != nil {
		return Result{}, err
	}
	if err = tx.Commit(); err != nil {
		return Result{}, err
	}
	return Result{AccessToken: accessToken, ExpiresIn: int64(s.accessTTL.Seconds()), RefreshToken: next, Principal: principal}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var sessionID string
	err = tx.QueryRowContext(ctx, `SELECT session_id FROM platform_identity_refresh_token WHERE token_hash=$1 FOR UPDATE`, tokenHash(refreshToken)).Scan(&sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE platform_identity_session SET revoked_at=COALESCE(revoked_at,NOW()),revoke_reason=COALESCE(revoke_reason,'LOGOUT') WHERE session_id=$1`, sessionID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE platform_identity_refresh_token SET status='REVOKED' WHERE session_id=$1 AND status='ACTIVE'`, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) issue(principal Principal) (string, error) {
	return s.issuer.Sign(accessidentity.Claims{Subject: principal.Subject, Realm: principal.Realm, AppID: principal.AppID, MerchantID: principal.MerchantID}, s.accessTTL)
}

func validLogin(input LoginInput) bool {
	if input.Username == "" || input.Password == "" {
		return false
	}
	if input.Realm == accessidentity.RealmPlatform {
		return true
	}
	return input.Realm == accessidentity.RealmMerchant && input.AppID > 0 && input.MerchantID > 0
}

func randomToken(size int) string {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(value)
}

func tokenHash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func appendAudit(ctx context.Context, tx *sql.Tx, principal Principal, action, resourceType, resourceID string, details map[string]any) error {
	return appendAuditResult(ctx, tx, principal, action, resourceType, resourceID, "SUCCEEDED", details)
}

func appendAuditResult(ctx context.Context, tx *sql.Tx, principal Principal, action, resourceType, resourceID, result string, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO platform_audit_event(event_id,realm,app_id,merchant_id,actor_subject,action,resource_type,resource_id,result,details)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, randomToken(18), principal.Realm, principal.AppID, principal.MerchantID, principal.Subject, action, resourceType, resourceID, result, payload)
	return err
}
