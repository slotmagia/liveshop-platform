package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
	"github.com/liveshop-platform/module-platform/internal/data/secretbox"
)

type LocalizationRepository struct {
	db  *sql.DB
	box *secretbox.Box
}

var _ localization.Repository = (*LocalizationRepository)(nil)

func NewLocalizationRepository(db *sql.DB, box *secretbox.Box) *LocalizationRepository {
	return &LocalizationRepository{db: db, box: box}
}

func (r *LocalizationRepository) GetConfig(ctx context.Context) (model.Config, error) {
	var item model.Config
	var provider string
	var apiKeySet bool
	err := r.db.QueryRowContext(ctx, `SELECT provider, api_key_set, version, created_at, updated_at FROM i18n_config WHERE id=1`).
		Scan(&provider, &apiKeySet, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Config{Provider: model.ProviderNoop}, nil
	}
	if err != nil {
		return model.Config{}, err
	}
	item.Provider = model.Provider(provider)
	item.APIKeySet = apiKeySet
	return item, nil
}

func (r *LocalizationRepository) LoadAPIKey(ctx context.Context) (string, error) {
	var sealed []byte
	var keyID string
	err := r.db.QueryRowContext(ctx, `SELECT credential_ciphertext, credential_key_id FROM i18n_config WHERE id=1`).Scan(&sealed, &keyID)
	if errors.Is(err, sql.ErrNoRows) || len(sealed) == 0 {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if r.box == nil || keyID != r.box.KeyID() {
		return "", model.ErrUnavailable
	}
	plain, err := r.box.Open(sealed, []byte("i18n-config:mt"))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (r *LocalizationRepository) UpsertConfig(ctx context.Context, input model.UpsertConfig, requestHash string) (model.Config, error) {
	var out model.Config
	err := withTx(ctx, r.db, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayI18nConfig(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			out = replay
			return nil
		}
		current, sealed, keyID, err := lockI18nConfig(ctx, tx, r.box)
		if err != nil {
			return err
		}
		if current.Version != input.ExpectedVersion {
			return model.ErrConflict
		}
		current.Provider = input.Provider
		if input.APIKeyClear {
			sealed, keyID = nil, ""
			current.APIKeySet = false
		} else if input.APIKey != "" {
			if r.box == nil {
				return model.ErrUnavailable
			}
			sealed, err = r.box.Seal([]byte(input.APIKey), []byte("i18n-config:mt"))
			if err != nil {
				return err
			}
			keyID = r.box.KeyID()
			current.APIKeySet = true
		}
		current.Version++
		if _, err := tx.ExecContext(ctx, `UPDATE i18n_config SET provider=?, credential_ciphertext=?, credential_key_id=?, api_key_set=?, version=?, updated_at=CURRENT_TIMESTAMP(3) WHERE id=1 AND version=?`,
			current.Provider, nullableBytes(sealed), keyID, current.APIKeySet, current.Version, input.ExpectedVersion); err != nil {
			return err
		}
		if err := insertI18nCommand(ctx, tx, input.CommandKey, requestHash, "CONFIG", "CONFIG", "singleton", current.Version); err != nil {
			return err
		}
		out = current
		return tx.QueryRowContext(ctx, `SELECT created_at, updated_at FROM i18n_config WHERE id=1`).Scan(&out.CreatedAt, &out.UpdatedAt)
	})
	return out, err
}

func (r *LocalizationRepository) ListWorklist(ctx context.Context, entityType, locale string) ([]model.WorklistRow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.entity_id, s.merchant_id, s.shop_id, s.source_text, COALESCE(t.value,''), COALESCE(t.status,''), COALESCE(t.text_source,''), COALESCE(t.version,0),
		CASE WHEN t.entity_id IS NULL THEN 0 ELSE s.source_version > t.source_version_at_write END
		FROM i18n_source s
		LEFT JOIN i18n_text t ON t.entity_type=s.entity_type AND t.entity_id=s.entity_id AND t.merchant_id=s.merchant_id AND t.shop_id=s.shop_id AND t.locale=?
		WHERE s.entity_type=?
		ORDER BY s.merchant_id, s.shop_id, s.entity_id`, locale, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.WorklistRow{}
	for rows.Next() {
		var row model.WorklistRow
		var stale int
		if err := rows.Scan(&row.EntityID, &row.MerchantID, &row.ShopID, &row.Source, &row.Value, &row.Status, &row.TextSource, &row.Version, &stale); err != nil {
			return nil, err
		}
		row.Stale = stale == 1
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *LocalizationRepository) Publish(ctx context.Context, input model.PublishInput, requestHash string) (model.PublishResult, error) {
	var out model.PublishResult
	err := withTx(ctx, r.db, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayI18nVersion(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			out = model.PublishResult{OK: true, Version: replay}
			return nil
		}
		var sourceVersion int64
		if err := tx.QueryRowContext(ctx, `SELECT source_version FROM i18n_source WHERE entity_type=? AND entity_id=? AND merchant_id=? AND shop_id=? FOR UPDATE`,
			input.EntityType, input.EntityID, input.MerchantID, input.ShopID).Scan(&sourceVersion); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		var currentVersion int64
		err := tx.QueryRowContext(ctx, `SELECT version FROM i18n_text WHERE entity_type=? AND entity_id=? AND locale=? AND merchant_id=? AND shop_id=? FOR UPDATE`,
			input.EntityType, input.EntityID, input.Locale, input.MerchantID, input.ShopID).Scan(&currentVersion)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if currentVersion != input.ExpectedVersion {
			return model.ErrConflict
		}
		next := currentVersion + 1
		if _, err := tx.ExecContext(ctx, `INSERT INTO i18n_text(entity_type,entity_id,locale,merchant_id,shop_id,value,status,text_source,source_version_at_write,version)
			VALUES(?,?,?,?,?,?,?,?,?,?)
			ON DUPLICATE KEY UPDATE value=VALUES(value), status=VALUES(status), text_source=VALUES(text_source), source_version_at_write=VALUES(source_version_at_write), version=VALUES(version)`,
			input.EntityType, input.EntityID, input.Locale, input.MerchantID, input.ShopID, input.Value, model.StatusPublished, model.SourceHuman, sourceVersion, next); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO i18n_text_history(entity_type,entity_id,locale,merchant_id,shop_id,value,status,text_source,source_version_at_write,version)
			VALUES(?,?,?,?,?,?,?,?,?,?)`, input.EntityType, input.EntityID, input.Locale, input.MerchantID, input.ShopID, input.Value, model.StatusPublished, model.SourceHuman, sourceVersion, next); err != nil {
			return err
		}
		if err := insertI18nCommand(ctx, tx, input.CommandKey, requestHash, "PUBLISH", "TEXT", input.EntityType+":"+input.EntityID, next); err != nil {
			return err
		}
		out = model.PublishResult{OK: true, Version: next}
		return nil
	})
	return out, err
}

func (r *LocalizationRepository) Fill(ctx context.Context, input model.FillInput, provider model.Provider, apiKey string, translator localization.Translator, requestHash string) (model.FillResult, error) {
	var out model.FillResult
	err := withTx(ctx, r.db, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayI18nFill(ctx, tx, input.CommandKey, requestHash, provider); err != nil {
			return err
		} else if found {
			out = replay
			return nil
		}
		rows, err := tx.QueryContext(ctx, `SELECT s.entity_id, s.merchant_id, s.shop_id, s.source_text, s.source_version, COALESCE(t.status,''), COALESCE(t.source_version_at_write,0), COALESCE(t.version,0)
			FROM i18n_source s
			LEFT JOIN i18n_text t ON t.entity_type=s.entity_type AND t.entity_id=s.entity_id AND t.merchant_id=s.merchant_id AND t.shop_id=s.shop_id AND t.locale=?
			WHERE s.entity_type=? FOR UPDATE`, input.Locale, input.EntityType)
		if err != nil {
			return err
		}
		type fillRow struct {
			entityID, source, status                        string
			merchantID, shopID, sourceVersion, written, ver int64
		}
		pending := []fillRow{}
		for rows.Next() {
			var row fillRow
			if err := rows.Scan(&row.entityID, &row.merchantID, &row.shopID, &row.source, &row.sourceVersion, &row.status, &row.written, &row.ver); err != nil {
				rows.Close()
				return err
			}
			pending = append(pending, row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		filled, skipped := 0, 0
		for _, row := range pending {
			if row.status == model.StatusPublished && row.sourceVersion <= row.written {
				skipped++
				continue
			}
			translated, err := translator.Translate(ctx, provider, apiKey, row.source, input.Locale)
			if err != nil {
				skipped++
				continue
			}
			next := row.ver + 1
			if _, err := tx.ExecContext(ctx, `INSERT INTO i18n_text(entity_type,entity_id,locale,merchant_id,shop_id,value,status,text_source,source_version_at_write,version)
				VALUES(?,?,?,?,?,?,?,?,?,?)
				ON DUPLICATE KEY UPDATE value=VALUES(value), status=VALUES(status), text_source=VALUES(text_source), source_version_at_write=VALUES(source_version_at_write), version=VALUES(version)`,
				input.EntityType, row.entityID, input.Locale, row.merchantID, row.shopID, translated, model.StatusMachine, model.SourceMachine, row.sourceVersion, next); err != nil {
				return err
			}
			filled++
		}
		if err := insertI18nCommand(ctx, tx, input.CommandKey, requestHash, "FILL", "TEXT", input.EntityType+":"+input.Locale, int64(filled)); err != nil {
			return err
		}
		out = model.FillResult{Provider: provider, Filled: filled, Skipped: skipped}
		return nil
	})
	return out, err
}

func (r *LocalizationRepository) UpsertSource(ctx context.Context, snapshot model.SourceSnapshot) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO i18n_source(entity_type,entity_id,merchant_id,shop_id,source_text,source_version)
		VALUES(?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE source_text=VALUES(source_text), source_version=VALUES(source_version)`,
		snapshot.EntityType, snapshot.EntityID, snapshot.MerchantID, snapshot.ShopID, snapshot.Source, snapshot.SourceVersion)
	return err
}

func (r *LocalizationRepository) ListPublished(ctx context.Context, entityType, locale string, merchantID, shopID int64) ([]model.PublishedText, error) {
	query := `SELECT entity_id, value, version FROM i18n_text WHERE entity_type=? AND locale=? AND status=?`
	args := []any{entityType, locale, model.StatusPublished}
	if merchantID != 0 || shopID != 0 {
		query += ` AND merchant_id=? AND shop_id=?`
		args = append(args, merchantID, shopID)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PublishedText{}
	for rows.Next() {
		var item model.PublishedText
		if err := rows.Scan(&item.EntityID, &item.Value, &item.Version); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func lockI18nConfig(ctx context.Context, tx *sql.Tx, box *secretbox.Box) (model.Config, []byte, string, error) {
	var item model.Config
	var provider, keyID string
	var sealed []byte
	var apiKeySet bool
	err := tx.QueryRowContext(ctx, `SELECT provider, credential_ciphertext, credential_key_id, api_key_set, version, created_at, updated_at FROM i18n_config WHERE id=1 FOR UPDATE`).
		Scan(&provider, &sealed, &keyID, &apiKeySet, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO i18n_config(id, provider, version) VALUES(1, 'noop', 0)`); err != nil {
			return model.Config{}, nil, "", err
		}
		return model.Config{Provider: model.ProviderNoop}, nil, "", nil
	}
	if err != nil {
		return model.Config{}, nil, "", err
	}
	item.Provider = model.Provider(provider)
	item.APIKeySet = apiKeySet
	_ = box
	return item, sealed, keyID, nil
}

func insertI18nCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash, action, kind, resourceID string, version int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO i18n_command(command_key,request_hash,action,resource_kind,resource_id,result_version) VALUES(?,?,?,?,?,?)`,
		commandKey, requestHash, action, kind, resourceID, version)
	return err
}

func replayI18nConfig(ctx context.Context, tx *sql.Tx, commandKey, requestHash string) (model.Config, bool, error) {
	var storedHash, kind string
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash, resource_kind, result_version FROM i18n_command WHERE command_key=? FOR UPDATE`, commandKey).Scan(&storedHash, &kind, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Config{}, false, nil
	}
	if err != nil {
		return model.Config{}, false, err
	}
	if storedHash != requestHash || kind != "CONFIG" {
		return model.Config{}, false, model.ErrConflict
	}
	item, _, _, err := lockI18nConfig(ctx, tx, nil)
	return item, true, err
}

func replayI18nVersion(ctx context.Context, tx *sql.Tx, commandKey, requestHash string) (int64, bool, error) {
	var storedHash, kind string
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash, resource_kind, result_version FROM i18n_command WHERE command_key=? FOR UPDATE`, commandKey).Scan(&storedHash, &kind, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if storedHash != requestHash {
		return 0, false, model.ErrConflict
	}
	return version, true, nil
}

func replayI18nFill(ctx context.Context, tx *sql.Tx, commandKey, requestHash string, provider model.Provider) (model.FillResult, bool, error) {
	var storedHash, action string
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash, action, result_version FROM i18n_command WHERE command_key=? FOR UPDATE`, commandKey).Scan(&storedHash, &action, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return model.FillResult{}, false, nil
	}
	if err != nil {
		return model.FillResult{}, false, err
	}
	if storedHash != requestHash || action != "FILL" {
		return model.FillResult{}, false, model.ErrConflict
	}
	return model.FillResult{Provider: provider, Filled: int(version)}, true, nil
}

func withTx(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("i18n commit: %w", err)
	}
	return nil
}
