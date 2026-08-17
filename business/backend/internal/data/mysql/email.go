package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/email"
	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
	"github.com/liveshop-platform/module-platform/internal/data/secretbox"
)

const emailTransactionTimeout = 5 * time.Second

type EmailRepository struct {
	db  *sql.DB
	box *secretbox.Box
}

type storedEmail struct {
	config  emailmodel.Config
	secrets map[string]string
	sealed  []byte
}

var _ email.Repository = (*EmailRepository)(nil)

func NewEmailRepository(db *sql.DB, box *secretbox.Box) *EmailRepository {
	return &EmailRepository{db: db, box: box}
}

func (r *EmailRepository) GetConfig(ctx context.Context, scope emailmodel.Scope) (emailmodel.Config, error) {
	item, err := scanPublicEmail(r.db.QueryRowContext(ctx, `SELECT id,driver,enabled,public_config,credential_key_id,credential_masks,version,created_at,updated_at
		FROM email_config`))
	if errors.Is(err, sql.ErrNoRows) {
		return emailmodel.Config{PublicConfig: map[string]string{}, SecretMasks: map[string]string{}}, nil
	}
	return item, err
}

func (r *EmailRepository) UpsertConfig(ctx context.Context, scope emailmodel.Scope, input emailmodel.UpsertConfig, requestHash string) (emailmodel.Config, error) {
	var output emailmodel.Config
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayEmailCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			output = replay
			return nil
		}
		current, err := r.lockConfig(ctx, tx)
		if err != nil {
			return err
		}
		if current == nil && input.ExpectedVersion != 0 {
			return emailmodel.ErrConflict
		}
		if current != nil && current.config.Version != input.ExpectedVersion {
			return emailmodel.ErrConflict
		}
		secrets := map[string]string{}
		if current != nil && current.config.Driver == input.Driver {
			secrets = current.secrets
		}
		secrets = emailmodel.ApplySecrets(secrets, input.Secrets)
		sealed, masks, keyID, err := r.sealConfig(secrets)
		if err != nil {
			return err
		}
		next := storedEmail{config: emailmodel.Config{
			Driver: input.Driver, Enabled: current == nil || current.config.Enabled,
			PublicConfig: input.PublicConfig, SecretMasks: masks, CredentialKeyID: keyID, Version: 1,
		}, secrets: secrets, sealed: sealed}
		if current != nil {
			next.config.ID = current.config.ID
			next.config.Version = current.config.Version + 1
			next.config.CreatedAt = current.config.CreatedAt
			if err := updateEmailAndSnapshot(ctx, tx, next); err != nil {
				return err
			}
		} else if err := insertEmailAndSnapshot(ctx, tx, &next); err != nil {
			return err
		}
		if err := insertEmailCommand(ctx, tx, input.CommandKey, requestHash, "UPSERT_CONFIG", next.config.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "email.config.upsert", ResourceType: "platform.email.config", ResourceID: emailmodel.ResourceID, Details: map[string]any{"version": next.config.Version, "driver": input.Driver}}); err != nil {
			return err
		}
		output = next.config
		return bumpEmailCatalogue(ctx, tx)
	})
	return output, err
}

func (r *EmailRepository) SetEnabled(ctx context.Context, scope emailmodel.Scope, input emailmodel.SetEnabled, requestHash string) (emailmodel.Config, error) {
	var output emailmodel.Config
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayEmailCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			output = replay
			return nil
		}
		current, err := r.lockConfig(ctx, tx)
		if err != nil {
			return err
		}
		if current == nil {
			return emailmodel.ErrNotFound
		}
		if current.config.Version != input.ExpectedVersion {
			return emailmodel.ErrConflict
		}
		current.config.Enabled = input.Enabled
		current.config.Version++
		if err := updateEmailAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if err := insertEmailCommand(ctx, tx, input.CommandKey, requestHash, "ENABLE_CONFIG", current.config.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "email.config.enabled", ResourceType: "platform.email.config", ResourceID: emailmodel.ResourceID, Details: map[string]any{"version": current.config.Version, "enabled": input.Enabled}}); err != nil {
			return err
		}
		output = current.config
		return bumpEmailCatalogue(ctx, tx)
	})
	return output, err
}

func (r *EmailRepository) LoadSecrets(ctx context.Context, scope emailmodel.Scope) (emailmodel.ConfigSecrets, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,driver,enabled,public_config,credential_ciphertext,credential_key_id,credential_masks,version,created_at,updated_at
		FROM email_config`)
	stored, err := r.scanStoredEmail(row)
	if errors.Is(err, sql.ErrNoRows) {
		return emailmodel.ConfigSecrets{Config: emailmodel.Config{PublicConfig: map[string]string{}, SecretMasks: map[string]string{}}, Values: map[string]string{}}, nil
	}
	if err != nil {
		return emailmodel.ConfigSecrets{}, err
	}
	values := map[string]string{}
	for key, value := range stored.config.PublicConfig {
		values[key] = value
	}
	for key, value := range stored.secrets {
		values[key] = value
	}
	return emailmodel.ConfigSecrets{Config: stored.config, Values: values}, nil
}

func (r *EmailRepository) withCatalogue(ctx context.Context, scope emailmodel.Scope, operation func(context.Context, *sql.Tx) error) error {
	return transaction(ctx, r.db, emailTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO email_catalogue(id,revision) VALUES(1,0) ON DUPLICATE KEY UPDATE id=id`); err != nil {
			return err
		}
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM email_catalogue WHERE id=1 FOR UPDATE`).Scan(&revision); err != nil {
			return err
		}
		return operation(ctx, tx)
	})
}

func (r *EmailRepository) sealConfig(secrets map[string]string) ([]byte, map[string]string, string, error) {
	masks := emailmodel.MaskSecrets(secrets)
	if len(secrets) == 0 {
		return nil, masks, "", nil
	}
	plain, err := json.Marshal(secrets)
	if err != nil {
		return nil, nil, "", err
	}
	sealed, err := r.box.Seal(plain, []byte(fmt.Sprintf("email-config:%d", 0)))
	return sealed, masks, r.box.KeyID(), err
}

func (r *EmailRepository) openConfig(keyID string, sealed []byte) (map[string]string, error) {
	if len(sealed) == 0 {
		return map[string]string{}, nil
	}
	if keyID != r.box.KeyID() {
		return nil, errors.New("email config credential key is unavailable")
	}
	plain, err := r.box.Open(sealed, []byte(fmt.Sprintf("email-config:%d", 0)))
	if err != nil {
		return nil, err
	}
	var secrets map[string]string
	if err := json.Unmarshal(plain, &secrets); err != nil {
		return nil, err
	}
	if secrets == nil {
		secrets = map[string]string{}
	}
	return secrets, nil
}

func (r *EmailRepository) lockConfig(ctx context.Context, tx *sql.Tx) (*storedEmail, error) {
	stored, err := r.scanStoredEmail(tx.QueryRowContext(ctx, `SELECT id,driver,enabled,public_config,credential_ciphertext,credential_key_id,credential_masks,version,created_at,updated_at
		FROM email_config FOR UPDATE`))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &stored, nil
}

func (r *EmailRepository) scanStoredEmail(row scanner) (storedEmail, error) {
	var item storedEmail
	var publicRaw, masksRaw []byte
	var keyID string
	err := row.Scan(&item.config.ID, &item.config.Driver, &item.config.Enabled, &publicRaw, &item.sealed, &keyID, &masksRaw, &item.config.Version, &item.config.CreatedAt, &item.config.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(publicRaw, &item.config.PublicConfig); err != nil {
		return item, err
	}
	if item.config.PublicConfig == nil {
		item.config.PublicConfig = map[string]string{}
	}
	if err := json.Unmarshal(masksRaw, &item.config.SecretMasks); err != nil {
		return item, err
	}
	if item.config.SecretMasks == nil {
		item.config.SecretMasks = map[string]string{}
	}
	item.config.CredentialKeyID = keyID
	item.secrets, err = r.openConfig(keyID, item.sealed)
	return item, err
}

func scanPublicEmail(row scanner) (emailmodel.Config, error) {
	var item emailmodel.Config
	var publicRaw, masksRaw []byte
	err := row.Scan(&item.ID, &item.Driver, &item.Enabled, &publicRaw, &item.CredentialKeyID, &masksRaw, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(publicRaw, &item.PublicConfig); err != nil {
		return item, err
	}
	if item.PublicConfig == nil {
		item.PublicConfig = map[string]string{}
	}
	if err := json.Unmarshal(masksRaw, &item.SecretMasks); err != nil {
		return item, err
	}
	if item.SecretMasks == nil {
		item.SecretMasks = map[string]string{}
	}
	return item, nil
}

func insertEmailAndSnapshot(ctx context.Context, tx *sql.Tx, item *storedEmail) error {
	publicRaw, masksRaw, err := emailJSON(*item)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO email_config(driver,enabled,public_config,credential_ciphertext,credential_key_id,credential_masks,version)
		VALUES(?,?,?,?,?,?,?)`, item.config.Driver, item.config.Enabled, publicRaw, nullableBytes(item.sealed), item.config.CredentialKeyID, masksRaw, item.config.Version)
	if err != nil {
		return err
	}
	item.config.ID, _ = result.LastInsertId()
	if err := insertEmailSnapshot(ctx, tx, *item, publicRaw, masksRaw); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT created_at,updated_at FROM email_config WHERE id=?`, item.config.ID).Scan(&item.config.CreatedAt, &item.config.UpdatedAt)
}

func updateEmailAndSnapshot(ctx context.Context, tx *sql.Tx, item storedEmail) error {
	publicRaw, masksRaw, err := emailJSON(item)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE email_config SET driver=?,enabled=?,public_config=?,credential_ciphertext=?,credential_key_id=?,credential_masks=?,version=?,updated_at=CURRENT_TIMESTAMP(3)
		WHERE version=?`, item.config.Driver, item.config.Enabled, publicRaw, nullableBytes(item.sealed), item.config.CredentialKeyID, masksRaw, item.config.Version, item.config.Version-1)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return emailmodel.ErrConflict
	}
	if err := insertEmailSnapshot(ctx, tx, item, publicRaw, masksRaw); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT updated_at FROM email_config`).Scan(&item.config.UpdatedAt)
}

func insertEmailSnapshot(ctx context.Context, tx *sql.Tx, item storedEmail, publicRaw, masksRaw []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO email_config_version(version,driver,enabled,public_config,credential_ciphertext,credential_key_id,credential_masks)
		VALUES(?,?,?,?,?,?,?)`, item.config.Version, item.config.Driver, item.config.Enabled, publicRaw, nullableBytes(item.sealed), item.config.CredentialKeyID, masksRaw)
	return err
}

func insertEmailCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash, action string, version int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO email_command(command_key,request_hash,action,resource_kind,resource_id,result_version) VALUES(?,?,?,?,?,?)`, commandKey, requestHash, action, "CONFIG", emailmodel.ResourceID, version)
	return err
}

func bumpEmailCatalogue(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `UPDATE email_catalogue SET revision=revision+1,updated_at=CURRENT_TIMESTAMP(3) WHERE id=1`)
	return err
}

func replayEmailCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash string) (emailmodel.Config, bool, error) {
	var storedHash, resourceKind string
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash,resource_kind,result_version FROM email_command WHERE command_key=? FOR UPDATE`, commandKey).Scan(&storedHash, &resourceKind, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return emailmodel.Config{}, false, nil
	}
	if err != nil {
		return emailmodel.Config{}, false, err
	}
	if storedHash != requestHash || resourceKind != "CONFIG" {
		return emailmodel.Config{}, false, emailmodel.ErrConflict
	}
	item, err := loadEmailSnapshot(ctx, tx, version)
	return item, true, err
}

func loadEmailSnapshot(ctx context.Context, tx *sql.Tx, version int64) (emailmodel.Config, error) {
	item, err := scanPublicEmail(tx.QueryRowContext(ctx, `SELECT COALESCE(c.id,0),v.driver,v.enabled,v.public_config,v.credential_key_id,v.credential_masks,v.version,COALESCE(c.created_at,v.created_at),v.created_at
		FROM email_config_version v LEFT JOIN email_config c ON c.singleton=1
		WHERE v.version=?`, version))
	if errors.Is(err, sql.ErrNoRows) {
		return item, emailmodel.ErrConflict
	}
	return item, err
}

func emailJSON(item storedEmail) ([]byte, []byte, error) {
	publicRaw, err := json.Marshal(item.config.PublicConfig)
	if err != nil {
		return nil, nil, err
	}
	masksRaw, err := json.Marshal(item.config.SecretMasks)
	return publicRaw, masksRaw, err
}
