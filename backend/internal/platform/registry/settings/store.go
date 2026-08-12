// Package settings owns non-secret, versioned platform and tenant settings.
package settings

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/lvtuopen-ai/kernel-go/apperror"
)

var (
	ErrInvalid  = apperror.New("platform.settings.invalid", "settings namespace or value is invalid")
	ErrConflict = apperror.New("platform.settings.conflict", "settings version conflict")
)

var namespacePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
var secretKeyPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|private.?key|credential|api.?key)`)

type Scope struct {
	Realm      string
	AppID      int64
	MerchantID int64
	Subject    string
}

func (s Scope) Valid() bool {
	return (s.Realm == "PLATFORM" || s.Realm == "MERCHANT") && s.AppID > 0 && s.MerchantID > 0 && s.Subject != ""
}

type Document struct {
	Namespace string          `json:"namespace"`
	Value     json.RawMessage `json:"value"`
	Version   int64           `json:"version"`
	UpdatedBy string          `json:"updatedBy,omitempty"`
	UpdatedAt time.Time       `json:"updatedAt,omitempty"`
}

type Store struct{ db *sql.DB }

func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("settings database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) List(ctx context.Context, scope Scope) ([]Document, error) {
	if !scope.Valid() {
		return nil, ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT namespace,value_json,version,updated_by,updated_at FROM platform_setting
        WHERE realm=$1 AND app_id=$2 AND merchant_id=$3 ORDER BY namespace`, scope.Realm, scope.AppID, scope.MerchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []Document
	for rows.Next() {
		var item Document
		if err := rows.Scan(&item.Namespace, &item.Value, &item.Version, &item.UpdatedBy, &item.UpdatedAt); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}

func (s *Store) Get(ctx context.Context, scope Scope, namespace string) (Document, error) {
	namespace = strings.ToLower(strings.TrimSpace(namespace))
	if !scope.Valid() || !namespacePattern.MatchString(namespace) {
		return Document{}, ErrInvalid
	}
	var item Document
	err := s.db.QueryRowContext(ctx, `SELECT namespace,value_json,version,updated_by,updated_at FROM platform_setting
        WHERE realm=$1 AND app_id=$2 AND merchant_id=$3 AND namespace=$4`, scope.Realm, scope.AppID, scope.MerchantID, namespace).
		Scan(&item.Namespace, &item.Value, &item.Version, &item.UpdatedBy, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Document{Namespace: namespace, Value: json.RawMessage(`{}`), Version: 0}, nil
	}
	return item, err
}

func (s *Store) Put(ctx context.Context, scope Scope, namespace string, expectedVersion int64, value json.RawMessage) (Document, error) {
	namespace = strings.ToLower(strings.TrimSpace(namespace))
	canonical, err := validateValue(namespace, value)
	if !scope.Valid() || !namespacePattern.MatchString(namespace) || expectedVersion < 0 || err != nil {
		return Document{}, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback()
	var current Document
	err = tx.QueryRowContext(ctx, `SELECT namespace,value_json,version,updated_by,updated_at FROM platform_setting
        WHERE realm=$1 AND app_id=$2 AND merchant_id=$3 AND namespace=$4 FOR UPDATE`, scope.Realm, scope.AppID, scope.MerchantID, namespace).
		Scan(&current.Namespace, &current.Value, &current.Version, &current.UpdatedBy, &current.UpdatedAt)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Document{}, err
	}
	if exists && bytes.Equal(compact(current.Value), canonical) && (expectedVersion == current.Version || expectedVersion == current.Version-1) {
		return current, tx.Commit()
	}
	if (!exists && expectedVersion != 0) || (exists && current.Version != expectedVersion) {
		return Document{}, ErrConflict
	}
	nextVersion := int64(1)
	if exists {
		nextVersion = current.Version + 1
		result, err := tx.ExecContext(ctx, `UPDATE platform_setting SET value_json=$1,version=$2,updated_by=$3,updated_at=NOW()
            WHERE realm=$4 AND app_id=$5 AND merchant_id=$6 AND namespace=$7 AND version=$8`, canonical, nextVersion, scope.Subject, scope.Realm, scope.AppID, scope.MerchantID, namespace, expectedVersion)
		if err != nil {
			return Document{}, err
		}
		rows, _ := result.RowsAffected()
		if rows != 1 {
			return Document{}, ErrConflict
		}
	} else if _, err := tx.ExecContext(ctx, `INSERT INTO platform_setting(realm,app_id,merchant_id,namespace,value_json,version,updated_by)
        VALUES($1,$2,$3,$4,$5,1,$6)`, scope.Realm, scope.AppID, scope.MerchantID, namespace, canonical, scope.Subject); err != nil {
		return Document{}, err
	}
	details, _ := json.Marshal(map[string]any{"namespace": namespace, "version": nextVersion})
	if _, err := tx.ExecContext(ctx, `INSERT INTO platform_audit_event(event_id,realm,app_id,merchant_id,actor_subject,action,resource_type,resource_id,result,details)
        VALUES($1,$2,$3,$4,$5,'settings.update','platform.settings',$6,'SUCCEEDED',$7)`, eventID(), scope.Realm, scope.AppID, scope.MerchantID, scope.Subject, namespace, details); err != nil {
		return Document{}, err
	}
	if err := tx.Commit(); err != nil {
		return Document{}, err
	}
	return s.Get(ctx, scope, namespace)
}

func validateValue(namespace string, value json.RawMessage) ([]byte, error) {
	if namespace == "secrets" || len(value) == 0 {
		return nil, ErrInvalid
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, ErrInvalid
	}
	if _, ok := decoded.(map[string]any); !ok || containsSecret(decoded) {
		return nil, ErrInvalid
	}
	return json.Marshal(decoded)
}

func containsSecret(value any) bool {
	switch current := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if secretKeyPattern.MatchString(key) || containsSecret(current[key]) {
				return true
			}
		}
	case []any:
		for _, item := range current {
			if containsSecret(item) {
				return true
			}
		}
	}
	return false
}

func compact(value []byte) []byte {
	var output bytes.Buffer
	if json.Compact(&output, value) != nil {
		return value
	}
	return output.Bytes()
}

func eventID() string {
	value := make([]byte, 18)
	if _, err := rand.Read(value); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(value)
}
