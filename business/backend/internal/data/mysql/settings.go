package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"

	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

type SettingsRepository struct{ db *sql.DB }

var _ biz.SettingsRepository = (*SettingsRepository)(nil)

func NewSettingsRepository(db *sql.DB) *SettingsRepository { return &SettingsRepository{db: db} }

func (r *SettingsRepository) List(ctx context.Context, scope model.SettingScope) ([]model.SettingDocument, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT namespace,value_json,version,updated_by,updated_at FROM platform_setting
        WHERE realm=? AND merchant_id=? ORDER BY namespace`, scope.Realm, scope.MerchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []model.SettingDocument
	for rows.Next() {
		var item model.SettingDocument
		if err := rows.Scan(&item.Namespace, &item.Value, &item.Version, &item.UpdatedBy, &item.UpdatedAt); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}

func (r *SettingsRepository) Get(ctx context.Context, scope model.SettingScope, namespace string) (model.SettingDocument, error) {
	var item model.SettingDocument
	err := r.db.QueryRowContext(ctx, `SELECT namespace,value_json,version,updated_by,updated_at FROM platform_setting
        WHERE realm=? AND merchant_id=? AND namespace=?`, scope.Realm, scope.MerchantID, namespace).
		Scan(&item.Namespace, &item.Value, &item.Version, &item.UpdatedBy, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.SettingDocument{Namespace: namespace, Value: []byte(`{}`), Version: 0}, nil
	}
	return item, err
}

func (r *SettingsRepository) Put(ctx context.Context, scope model.SettingScope, namespace string, expectedVersion int64, canonical []byte) (model.SettingDocument, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return model.SettingDocument{}, err
	}
	defer tx.Rollback()

	var current model.SettingDocument
	err = tx.QueryRowContext(ctx, `SELECT namespace,value_json,version,updated_by,updated_at FROM platform_setting
        WHERE realm=? AND merchant_id=? AND namespace=? FOR UPDATE`, scope.Realm, scope.MerchantID, namespace).
		Scan(&current.Namespace, &current.Value, &current.Version, &current.UpdatedBy, &current.UpdatedAt)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.SettingDocument{}, err
	}
	if exists && bytes.Equal(model.CompactJSON(current.Value), canonical) && (expectedVersion == current.Version || expectedVersion == current.Version-1) {
		return current, tx.Commit()
	}
	if (!exists && expectedVersion != 0) || (exists && current.Version != expectedVersion) {
		return model.SettingDocument{}, model.ErrSettingsConflict
	}

	nextVersion := int64(1)
	if exists {
		nextVersion = current.Version + 1
		result, err := tx.ExecContext(ctx, `UPDATE platform_setting SET value_json=?,version=?,updated_by=?,updated_at=NOW()
            WHERE realm=? AND merchant_id=? AND namespace=? AND version=?`, canonical, nextVersion, scope.Subject, scope.Realm, scope.MerchantID, namespace, expectedVersion)
		if err != nil {
			return model.SettingDocument{}, err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return model.SettingDocument{}, model.ErrSettingsConflict
		}
	} else if _, err := tx.ExecContext(ctx, `INSERT INTO platform_setting(realm,merchant_id,namespace,value_json,version,updated_by)
        VALUES(?,?,?,?,1,?)`, scope.Realm, scope.MerchantID, namespace, canonical, scope.Subject); err != nil {
		return model.SettingDocument{}, err
	}
	if err := insertAuditEvent(ctx, tx, auditEntry{
		Realm:        scope.Realm,
		MerchantID:   scope.MerchantID,
		ActorSubject: scope.Subject,
		Action:       "settings.update",
		ResourceType: "platform.settings",
		ResourceID:   namespace,
		Details:      map[string]any{"namespace": namespace, "version": nextVersion},
	}); err != nil {
		return model.SettingDocument{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.SettingDocument{}, err
	}
	return r.Get(ctx, scope, namespace)
}
