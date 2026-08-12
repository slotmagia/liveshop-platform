package identity

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
	"github.com/lvtuopen-ai/kernel-go/apperror"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidAccount  = apperror.New("platform.identity.invalid_account", "identity account input is invalid")
	ErrAccountConflict = apperror.New("platform.identity.account_conflict", "identity account was concurrently changed or conflicts with another account")
	ErrAccountNotFound = apperror.New("platform.identity.account_not_found", "identity account was not found")
)

type AccountScope struct {
	AppID      int64
	MerchantID int64
}

type Account struct {
	Realm      string    `json:"realm"`
	AppID      int64     `json:"appId"`
	MerchantID int64     `json:"merchantId"`
	Subject    string    `json:"subject"`
	Username   string    `json:"username"`
	Status     string    `json:"status"`
	Version    int64     `json:"version"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type PutAccountInput struct {
	Realm           string `json:"realm"`
	Subject         string `json:"subject"`
	Username        string `json:"username"`
	Status          string `json:"status"`
	Password        string `json:"password,omitempty"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

func (s *Service) Accounts(ctx context.Context, scope AccountScope) ([]Account, error) {
	if scope.AppID <= 0 || scope.MerchantID <= 0 {
		return nil, ErrInvalidAccount
	}
	rows, err := s.db.QueryContext(ctx, `SELECT realm,app_id,merchant_id,subject,username,status,version,updated_at
		FROM platform_identity_account WHERE app_id=$1 AND merchant_id=$2 ORDER BY realm,username`, scope.AppID, scope.MerchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []Account
	for rows.Next() {
		var item Account
		if err := rows.Scan(&item.Realm, &item.AppID, &item.MerchantID, &item.Subject, &item.Username, &item.Status, &item.Version, &item.UpdatedAt); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}

func (s *Service) PutAccount(ctx context.Context, actor Principal, scope AccountScope, input PutAccountInput) (Account, error) {
	input.Realm = strings.ToUpper(strings.TrimSpace(input.Realm))
	input.Subject = strings.TrimSpace(input.Subject)
	input.Username = strings.ToLower(strings.TrimSpace(input.Username))
	if scope.AppID <= 0 || scope.MerchantID <= 0 || (input.Realm != accessidentity.RealmPlatform && input.Realm != accessidentity.RealmMerchant) || input.Subject == "" || input.Username == "" || (input.Status != "ACTIVE" && input.Status != "DISABLED") || input.ExpectedVersion < 0 || len(input.Subject) > 128 || len(input.Username) > 128 {
		return Account{}, ErrInvalidAccount
	}
	var passwordHash []byte
	var err error
	if input.Password != "" {
		if len(input.Password) < 12 || len(input.Password) > 128 {
			return Account{}, ErrInvalidAccount
		}
		passwordHash, err = bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return Account{}, err
		}
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Account{}, err
	}
	defer tx.Rollback()
	current, exists, err := accountForUpdate(ctx, tx, scope, input.Realm, input.Subject)
	if err != nil {
		return Account{}, err
	}
	if !exists && input.ExpectedVersion != 0 {
		return Account{}, ErrAccountConflict
	}
	if exists && current.Version != input.ExpectedVersion {
		if input.Password == "" && current.Username == input.Username && current.Status == input.Status && input.ExpectedVersion == current.Version-1 {
			return current, nil
		}
		return Account{}, ErrAccountConflict
	}
	if !exists && len(passwordHash) == 0 {
		return Account{}, ErrInvalidAccount
	}
	output := Account{Realm: input.Realm, AppID: scope.AppID, MerchantID: scope.MerchantID, Subject: input.Subject, Username: input.Username, Status: input.Status, Version: 1, UpdatedAt: s.now()}
	if exists {
		if input.Password == "" && current.Username == input.Username && current.Status == input.Status {
			return current, nil
		}
		output.Version = current.Version + 1
		var result sql.Result
		if len(passwordHash) > 0 {
			result, err = tx.ExecContext(ctx, `UPDATE platform_identity_account SET username=$1,status=$2,password_hash=$3,version=$4,failed_login_count=0,locked_until=NULL,updated_at=$5
				WHERE realm=$6 AND app_id=$7 AND merchant_id=$8 AND subject=$9 AND version=$10`, output.Username, output.Status, passwordHash, output.Version, output.UpdatedAt, output.Realm, output.AppID, output.MerchantID, output.Subject, current.Version)
		} else {
			result, err = tx.ExecContext(ctx, `UPDATE platform_identity_account SET username=$1,status=$2,version=$3,updated_at=$4
				WHERE realm=$5 AND app_id=$6 AND merchant_id=$7 AND subject=$8 AND version=$9`, output.Username, output.Status, output.Version, output.UpdatedAt, output.Realm, output.AppID, output.MerchantID, output.Subject, current.Version)
		}
		if err != nil {
			return Account{}, accountWriteError(err)
		}
		rows, _ := result.RowsAffected()
		if rows != 1 {
			return Account{}, ErrAccountConflict
		}
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO platform_identity_account(realm,app_id,merchant_id,subject,username,password_hash,status,version,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,1,$8)`, output.Realm, output.AppID, output.MerchantID, output.Subject, output.Username, passwordHash, output.Status, output.UpdatedAt)
		if err != nil {
			return Account{}, accountWriteError(err)
		}
	}
	if exists && (input.Password != "" || output.Status == "DISABLED") {
		if _, err := tx.ExecContext(ctx, `UPDATE platform_identity_session SET revoked_at=COALESCE(revoked_at,NOW()),revoke_reason=COALESCE(revoke_reason,'ACCOUNT_CHANGED')
			WHERE realm=$1 AND app_id=$2 AND merchant_id=$3 AND subject=$4 AND revoked_at IS NULL`, output.Realm, output.AppID, output.MerchantID, output.Subject); err != nil {
			return Account{}, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE platform_identity_refresh_token SET status='REVOKED' WHERE session_id IN (
			SELECT session_id FROM platform_identity_session WHERE realm=$1 AND app_id=$2 AND merchant_id=$3 AND subject=$4) AND status='ACTIVE'`, output.Realm, output.AppID, output.MerchantID, output.Subject); err != nil {
			return Account{}, err
		}
	}
	details := map[string]any{"realm": output.Realm, "username": output.Username, "status": output.Status, "version": output.Version, "passwordChanged": input.Password != ""}
	if err := appendAudit(ctx, tx, actor, "identity.account.put", "identity.account", output.Realm+":"+output.Subject, details); err != nil {
		return Account{}, err
	}
	if err := tx.Commit(); err != nil {
		return Account{}, accountWriteError(err)
	}
	return output, nil
}

func accountForUpdate(ctx context.Context, tx *sql.Tx, scope AccountScope, realm, subject string) (Account, bool, error) {
	var item Account
	err := tx.QueryRowContext(ctx, `SELECT realm,app_id,merchant_id,subject,username,status,version,updated_at FROM platform_identity_account
		WHERE realm=$1 AND app_id=$2 AND merchant_id=$3 AND subject=$4 FOR UPDATE`, realm, scope.AppID, scope.MerchantID, subject).
		Scan(&item.Realm, &item.AppID, &item.MerchantID, &item.Subject, &item.Username, &item.Status, &item.Version, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, false, nil
	}
	return item, err == nil, err
}

func accountWriteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && (postgresError.Code == "23505" || postgresError.Code == "40001") {
		return ErrAccountConflict
	}
	return err
}
