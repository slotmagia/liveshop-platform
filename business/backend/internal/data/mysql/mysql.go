// Package mysql implements the biz repository ports on MySQL 8. It owns
// SQL, transactions and row mapping, and never sees a transport type.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

const maxTransactionAttempts = 3

// transaction runs one unit of work under REPEATABLE READ, retrying MySQL
// deadlock and lock-wait classes with a bounded backoff. Row locks (FOR UPDATE)
// and optimistic version columns supply the isolation SERIALIZABLE used to.
func transaction(ctx context.Context, db *sql.DB, timeout time.Duration, operation func(context.Context, *sql.Tx) error) error {
	var last error
	for attempt := 0; attempt < maxTransactionAttempts; attempt++ {
		last = transactionOnce(ctx, db, timeout, operation)
		if !retryableTransactionError(last) {
			return last
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return last
}

func transactionOnce(parent context.Context, db *sql.DB, timeout time.Duration, operation func(context.Context, *sql.Tx) error) error {
	ctx := parent
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(parent, timeout)
		defer cancel()
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := operation(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func retryableTransactionError(err error) bool {
	var mysqlError *mysqldriver.MySQLError
	return errors.As(err, &mysqlError) && (mysqlError.Number == 1213 || mysqlError.Number == 1205)
}

// Verify is the single reachability gate for this adapter set. The composition
// root calls it once before constructing any repository, so a misconfigured
// process fails fast instead of falling back to an in-memory store.
func Verify(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("mysql: database is required")
	}
	return db.PingContext(ctx)
}

type auditEntry struct {
	Realm        string
	MerchantID   int64
	ActorSubject string
	Action       string
	ResourceType string
	ResourceID   string
	Result       string
	Details      any
}

func insertAuditEvent(ctx context.Context, tx *sql.Tx, entry auditEntry) error {
	if entry.Result == "" {
		entry.Result = "SUCCEEDED"
	}
	if entry.Details == nil {
		entry.Details = map[string]any{}
	}
	payload, err := json.Marshal(entry.Details)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO platform_audit_event(event_id,realm,merchant_id,actor_subject,action,resource_type,resource_id,result,details)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		model.NewEventID(), entry.Realm, entry.MerchantID, entry.ActorSubject,
		entry.Action, entry.ResourceType, entry.ResourceID, entry.Result, payload)
	return err
}
