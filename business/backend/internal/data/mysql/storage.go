package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/storage"
	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
	"github.com/liveshop-platform/module-platform/internal/data/secretbox"
)

const storageTransactionTimeout = 5 * time.Second

type StorageRepository struct {
	db  *sql.DB
	box *secretbox.Box
}

var _ storage.Repository = (*StorageRepository)(nil)

func NewStorageRepository(db *sql.DB, box *secretbox.Box) *StorageRepository {
	return &StorageRepository{db: db, box: box}
}

func (r *StorageRepository) ListChannels(ctx context.Context, scope storagemodel.Scope, filter storagemodel.ChannelFilter) ([]storagemodel.Channel, error) {
	query := `SELECT id,code,name,driver,enabled,is_default,lifecycle,public_config,credential_key_id,credential_masks,version,created_at,updated_at
		FROM storage_channel WHERE 1=1`
	args := []any{}
	if filter.Keyword != "" {
		query += ` AND (code LIKE ? OR name LIKE ?)`
		keyword := "%" + filter.Keyword + "%"
		args = append(args, keyword, keyword)
	}
	if filter.Driver != "" {
		query += ` AND driver=?`
		args = append(args, filter.Driver)
	}
	if filter.Lifecycle != "" {
		query += ` AND lifecycle=?`
		args = append(args, filter.Lifecycle)
	}
	query += ` ORDER BY lifecycle='ACTIVE' DESC,is_default DESC,id ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]storagemodel.Channel, 0)
	for rows.Next() {
		item, err := scanPublicStorageChannel(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *StorageRepository) UpsertChannel(ctx context.Context, scope storagemodel.Scope, input storagemodel.UpsertChannel, requestHash string) (storagemodel.Channel, error) {
	var output storagemodel.Channel
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayStorageCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			output = replay
			return nil
		}
		current, err := lockStorageChannel(ctx, tx, r, input.Code)
		if err != nil {
			return err
		}
		if current == nil && input.ExpectedVersion != 0 {
			return storagemodel.ErrConflict
		}
		if current != nil {
			if current.channel.Lifecycle == storagemodel.LifecycleRetired {
				return storagemodel.ErrRetired
			}
			if current.channel.Version != input.ExpectedVersion {
				return storagemodel.ErrConflict
			}
		}
		secrets := map[string]string{}
		if current != nil {
			secrets = current.secrets
		}
		secrets = storagemodel.ApplySecrets(secrets, input.Secrets)
		sealed, masks, keyID, err := r.sealChannel(input.Code, secrets)
		if err != nil {
			return err
		}
		next := storedStorageChannel{channel: storagemodel.Channel{
			Code: input.Code, Name: input.Name, Driver: input.Driver,
			Enabled: current == nil || current.channel.Enabled, IsDefault: current != nil && current.channel.IsDefault,
			Lifecycle: storagemodel.LifecycleActive, PublicConfig: input.PublicConfig, SecretMasks: masks, CredentialKeyID: keyID, Version: 1,
		}, secrets: secrets, sealed: sealed}
		if current != nil {
			next.channel.ID = current.channel.ID
			next.channel.Version = current.channel.Version + 1
			next.channel.CreatedAt = current.channel.CreatedAt
			if err := updateStorageChannelAndSnapshot(ctx, tx, next); err != nil {
				return err
			}
		} else if err := insertStorageChannelAndSnapshot(ctx, tx, &next); err != nil {
			return err
		}
		if err := insertStorageCommand(ctx, tx, input.CommandKey, requestHash, "UPSERT_CHANNEL", input.Code, next.channel.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "storage.channel.upsert", ResourceType: "platform.storage.channel", ResourceID: input.Code, Details: map[string]any{"version": next.channel.Version, "driver": input.Driver}}); err != nil {
			return err
		}
		output = next.channel
		return bumpStorageCatalogue(ctx, tx)
	})
	return output, err
}

func (r *StorageRepository) SetEnabled(ctx context.Context, scope storagemodel.Scope, input storagemodel.SetEnabled, requestHash string) (storagemodel.Channel, error) {
	var output storagemodel.Channel
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayStorageCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			output = replay
			return nil
		}
		current, err := requireActiveStorageChannel(ctx, tx, r, input.Code, input.ExpectedVersion)
		if err != nil {
			return err
		}
		current.channel.Enabled = input.Enabled
		if !input.Enabled {
			current.channel.IsDefault = false
		}
		current.channel.Version++
		if err := updateStorageChannelAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if err := insertStorageCommand(ctx, tx, input.CommandKey, requestHash, "ENABLE_CHANNEL", input.Code, current.channel.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "storage.channel.enabled", ResourceType: "platform.storage.channel", ResourceID: input.Code, Details: map[string]any{"version": current.channel.Version, "enabled": input.Enabled}}); err != nil {
			return err
		}
		output = current.channel
		return bumpStorageCatalogue(ctx, tx)
	})
	return output, err
}

func (r *StorageRepository) SetDefault(ctx context.Context, scope storagemodel.Scope, input storagemodel.SetDefault, requestHash string) (storagemodel.Channel, error) {
	var output storagemodel.Channel
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayStorageCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			output = replay
			return nil
		}
		current, err := requireActiveStorageChannel(ctx, tx, r, input.Code, input.ExpectedVersion)
		if err != nil {
			return err
		}
		if !current.channel.Enabled {
			return storagemodel.ErrDisabled
		}
		if err := clearOtherStorageDefaults(ctx, tx, r, input.Code); err != nil {
			return err
		}
		current.channel.IsDefault = true
		current.channel.Version++
		if err := updateStorageChannelAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if err := insertStorageCommand(ctx, tx, input.CommandKey, requestHash, "DEFAULT_CHANNEL", input.Code, current.channel.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "storage.channel.default", ResourceType: "platform.storage.channel", ResourceID: input.Code, Details: map[string]any{"version": current.channel.Version}}); err != nil {
			return err
		}
		output = current.channel
		return bumpStorageCatalogue(ctx, tx)
	})
	return output, err
}

func (r *StorageRepository) Retire(ctx context.Context, scope storagemodel.Scope, input storagemodel.Retire, requestHash string) (storagemodel.Channel, error) {
	var output storagemodel.Channel
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replayStorageCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			output = replay
			return nil
		}
		current, err := requireActiveStorageChannel(ctx, tx, r, input.Code, input.ExpectedVersion)
		if err != nil {
			return err
		}
		current.channel.Lifecycle = storagemodel.LifecycleRetired
		current.channel.Enabled = false
		current.channel.IsDefault = false
		current.channel.Version++
		if err := updateStorageChannelAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if err := insertStorageCommand(ctx, tx, input.CommandKey, requestHash, "RETIRE_CHANNEL", input.Code, current.channel.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "storage.channel.retire", ResourceType: "platform.storage.channel", ResourceID: input.Code, Details: map[string]any{"version": current.channel.Version}}); err != nil {
			return err
		}
		output = current.channel
		return bumpStorageCatalogue(ctx, tx)
	})
	return output, err
}

func (r *StorageRepository) LoadSecrets(ctx context.Context, scope storagemodel.Scope, code string) (storagemodel.ChannelSecrets, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,code,name,driver,enabled,is_default,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks,version,created_at,updated_at
		FROM storage_channel WHERE code=?`, code)
	stored, err := r.scanStoredChannel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return storagemodel.ChannelSecrets{}, storagemodel.ErrNotFound
	}
	if err != nil {
		return storagemodel.ChannelSecrets{}, err
	}
	config := map[string]string{}
	for key, value := range stored.channel.PublicConfig {
		config[key] = value
	}
	for key, value := range stored.secrets {
		config[key] = value
	}
	return storagemodel.ChannelSecrets{Channel: stored.channel, Config: config}, nil
}

type storedStorageChannel struct {
	channel storagemodel.Channel
	secrets map[string]string
	sealed  []byte
}

func (r *StorageRepository) withCatalogue(ctx context.Context, scope storagemodel.Scope, operation func(context.Context, *sql.Tx) error) error {
	return transaction(ctx, r.db, storageTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO storage_catalogue(id,revision) VALUES(1,0) ON DUPLICATE KEY UPDATE id=id`); err != nil {
			return err
		}
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM storage_catalogue WHERE id=1 FOR UPDATE`).Scan(&revision); err != nil {
			return err
		}
		return operation(ctx, tx)
	})
}

func (r *StorageRepository) sealChannel(code string, secrets map[string]string) ([]byte, map[string]string, string, error) {
	masks := storagemodel.MaskSecrets(secrets)
	if len(secrets) == 0 {
		return nil, masks, "", nil
	}
	plain, err := json.Marshal(secrets)
	if err != nil {
		return nil, nil, "", err
	}
	sealed, err := r.box.Seal(plain, []byte(fmt.Sprintf("storage-channel:%d:%s", 0, code)))
	return sealed, masks, r.box.KeyID(), err
}

func (r *StorageRepository) openChannel(code, keyID string, sealed []byte) (map[string]string, error) {
	if len(sealed) == 0 {
		return map[string]string{}, nil
	}
	if keyID != r.box.KeyID() {
		return nil, errors.New("storage channel credential key is unavailable")
	}
	plain, err := r.box.Open(sealed, []byte(fmt.Sprintf("storage-channel:%d:%s", 0, code)))
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

func (r *StorageRepository) scanStoredChannel(row scanner) (storedStorageChannel, error) {
	var item storedStorageChannel
	var publicRaw, masksRaw []byte
	var keyID string
	err := row.Scan(&item.channel.ID, &item.channel.Code, &item.channel.Name, &item.channel.Driver, &item.channel.Enabled, &item.channel.IsDefault, &item.channel.Lifecycle, &publicRaw, &item.sealed, &keyID, &masksRaw, &item.channel.Version, &item.channel.CreatedAt, &item.channel.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err := decodeStorageJSON(publicRaw, masksRaw, &item.channel); err != nil {
		return item, err
	}
	item.channel.CredentialKeyID = keyID
	item.secrets, err = r.openChannel(item.channel.Code, keyID, item.sealed)
	return item, err
}

func scanPublicStorageChannel(row scanner) (storagemodel.Channel, error) {
	var item storagemodel.Channel
	var publicRaw, masksRaw []byte
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Driver, &item.Enabled, &item.IsDefault, &item.Lifecycle, &publicRaw, &item.CredentialKeyID, &masksRaw, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	return item, decodeStorageJSON(publicRaw, masksRaw, &item)
}

func decodeStorageJSON(publicRaw, masksRaw []byte, item *storagemodel.Channel) error {
	if err := json.Unmarshal(publicRaw, &item.PublicConfig); err != nil {
		return err
	}
	if item.PublicConfig == nil {
		item.PublicConfig = map[string]string{}
	}
	if err := json.Unmarshal(masksRaw, &item.SecretMasks); err != nil {
		return err
	}
	if item.SecretMasks == nil {
		item.SecretMasks = map[string]string{}
	}
	return nil
}

func lockStorageChannel(ctx context.Context, tx *sql.Tx, repo *StorageRepository, code string) (*storedStorageChannel, error) {
	item, err := repo.scanStoredChannel(tx.QueryRowContext(ctx, `SELECT id,code,name,driver,enabled,is_default,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks,version,created_at,updated_at FROM storage_channel WHERE code=? FOR UPDATE`, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func requireActiveStorageChannel(ctx context.Context, tx *sql.Tx, repo *StorageRepository, code string, expected int64) (*storedStorageChannel, error) {
	current, err := lockStorageChannel(ctx, tx, repo, code)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, storagemodel.ErrNotFound
	}
	if current.channel.Lifecycle == storagemodel.LifecycleRetired {
		return nil, storagemodel.ErrRetired
	}
	if current.channel.Version != expected {
		return nil, storagemodel.ErrConflict
	}
	return current, nil
}

func clearOtherStorageDefaults(ctx context.Context, tx *sql.Tx, repo *StorageRepository, exceptCode string) error {
	rows, err := tx.QueryContext(ctx, `SELECT id,code,name,driver,enabled,is_default,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks,version,created_at,updated_at FROM storage_channel WHERE is_default=1 AND code<>? FOR UPDATE`, exceptCode)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := make([]storedStorageChannel, 0)
	for rows.Next() {
		item, err := repo.scanStoredChannel(rows)
		if err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range items {
		item.channel.IsDefault = false
		item.channel.Version++
		if err := updateStorageChannelAndSnapshot(ctx, tx, item); err != nil {
			return err
		}
	}
	return nil
}

func insertStorageChannelAndSnapshot(ctx context.Context, tx *sql.Tx, item *storedStorageChannel) error {
	publicRaw, masksRaw, err := storageChannelJSON(*item)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO storage_channel(code,name,driver,enabled,is_default,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks,version)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`, item.channel.Code, item.channel.Name, item.channel.Driver, item.channel.Enabled, item.channel.IsDefault, item.channel.Lifecycle, publicRaw, nullableBytes(item.sealed), item.channel.CredentialKeyID, masksRaw, item.channel.Version)
	if err != nil {
		return err
	}
	item.channel.ID, _ = result.LastInsertId()
	if err := insertStorageChannelSnapshot(ctx, tx, *item, publicRaw, masksRaw); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT created_at,updated_at FROM storage_channel WHERE id=?`, item.channel.ID).Scan(&item.channel.CreatedAt, &item.channel.UpdatedAt)
}

func updateStorageChannelAndSnapshot(ctx context.Context, tx *sql.Tx, item storedStorageChannel) error {
	publicRaw, masksRaw, err := storageChannelJSON(item)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE storage_channel SET name=?,driver=?,enabled=?,is_default=?,lifecycle=?,public_config=?,credential_ciphertext=?,credential_key_id=?,credential_masks=?,version=?,updated_at=CURRENT_TIMESTAMP(3)
		WHERE code=? AND version=?`, item.channel.Name, item.channel.Driver, item.channel.Enabled, item.channel.IsDefault, item.channel.Lifecycle, publicRaw, nullableBytes(item.sealed), item.channel.CredentialKeyID, masksRaw, item.channel.Version, item.channel.Code, item.channel.Version-1)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return storagemodel.ErrConflict
	}
	if err := insertStorageChannelSnapshot(ctx, tx, item, publicRaw, masksRaw); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT updated_at FROM storage_channel WHERE code=?`, item.channel.Code).Scan(&item.channel.UpdatedAt)
}

func insertStorageChannelSnapshot(ctx context.Context, tx *sql.Tx, item storedStorageChannel, publicRaw, masksRaw []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO storage_channel_version(channel_code,version,name,driver,enabled,is_default,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`, item.channel.Code, item.channel.Version, item.channel.Name, item.channel.Driver, item.channel.Enabled, item.channel.IsDefault, item.channel.Lifecycle, publicRaw, nullableBytes(item.sealed), item.channel.CredentialKeyID, masksRaw)
	return err
}

func storageChannelJSON(item storedStorageChannel) ([]byte, []byte, error) {
	publicRaw, err := json.Marshal(item.channel.PublicConfig)
	if err != nil {
		return nil, nil, err
	}
	masksRaw, err := json.Marshal(item.channel.SecretMasks)
	return publicRaw, masksRaw, err
}

func insertStorageCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash, action, resourceID string, version int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO storage_command(command_key,request_hash,action,resource_kind,resource_id,result_version) VALUES(?,?,?,?,?,?)`, commandKey, requestHash, action, "CHANNEL", resourceID, version)
	return err
}

func bumpStorageCatalogue(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `UPDATE storage_catalogue SET revision=revision+1,updated_at=CURRENT_TIMESTAMP(3) WHERE id=1`)
	return err
}

func replayStorageCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash string) (storagemodel.Channel, bool, error) {
	var storedHash, resourceKind, resourceID string
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash,resource_kind,resource_id,result_version FROM storage_command WHERE command_key=? FOR UPDATE`, commandKey).Scan(&storedHash, &resourceKind, &resourceID, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return storagemodel.Channel{}, false, nil
	}
	if err != nil {
		return storagemodel.Channel{}, false, err
	}
	if storedHash != requestHash || resourceKind != "CHANNEL" {
		return storagemodel.Channel{}, false, storagemodel.ErrConflict
	}
	item, err := loadStorageChannelSnapshot(ctx, tx, resourceID, version)
	return item, true, err
}

func loadStorageChannelSnapshot(ctx context.Context, tx *sql.Tx, code string, version int64) (storagemodel.Channel, error) {
	item, err := scanPublicStorageChannel(tx.QueryRowContext(ctx, `SELECT COALESCE(c.id,0),v.channel_code,v.name,v.driver,v.enabled,v.is_default,v.lifecycle,v.public_config,v.credential_key_id,v.credential_masks,v.version,COALESCE(c.created_at,v.created_at),v.created_at
		FROM storage_channel_version v LEFT JOIN storage_channel c ON c.code=v.channel_code
		WHERE v.channel_code=? AND v.version=?`, code, version))
	if errors.Is(err, sql.ErrNoRows) {
		return item, storagemodel.ErrConflict
	}
	return item, err
}
