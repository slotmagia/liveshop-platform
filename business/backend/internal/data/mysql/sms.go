package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/sms"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
	"github.com/liveshop-platform/module-platform/internal/data/secretbox"
)

const smsTransactionTimeout = 5 * time.Second

type SMSRepository struct {
	db  *sql.DB
	box *secretbox.Box
}

var _ sms.Repository = (*SMSRepository)(nil)

func NewSMSRepository(db *sql.DB, box *secretbox.Box) *SMSRepository {
	return &SMSRepository{db: db, box: box}
}

func (r *SMSRepository) ListChannels(ctx context.Context, scope smsmodel.Scope, filter smsmodel.ChannelFilter) ([]smsmodel.Channel, error) {
	query := `SELECT id,code,name,driver,region,priority,enabled,lifecycle,public_config,credential_key_id,credential_masks,version,created_at,updated_at
		FROM sms_channel WHERE 1=1`
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
	query += ` ORDER BY lifecycle='ACTIVE' DESC,priority DESC,id ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]smsmodel.Channel, 0)
	for rows.Next() {
		item, err := scanPublicChannel(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SMSRepository) UpsertChannel(ctx context.Context, scope smsmodel.Scope, input smsmodel.UpsertChannel, requestHash string) (smsmodel.Channel, error) {
	var output smsmodel.Channel
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replaySMSCommand(ctx, tx, input.CommandKey, requestHash, "CHANNEL"); err != nil {
			return err
		} else if found {
			output = replay.(smsmodel.Channel)
			return nil
		}
		if input.Region != smsmodel.WildcardRegion {
			if _, err := lockActiveRegionByDial(ctx, tx, input.Region); err != nil {
				return err
			}
		}
		current, err := lockChannel(ctx, tx, r, input.Code)
		if err != nil {
			return err
		}
		if current == nil && input.ExpectedVersion != 0 {
			return smsmodel.ErrConflict
		}
		if current != nil {
			if current.channel.Lifecycle == smsmodel.LifecycleRetired {
				return smsmodel.ErrRetired
			}
			if current.channel.Version != input.ExpectedVersion {
				return smsmodel.ErrConflict
			}
		}
		secrets := map[string]string{}
		if current != nil {
			secrets = current.secrets
		}
		secrets = smsmodel.ApplySecrets(secrets, input.Secrets)
		sealed, masks, keyID, err := r.sealChannel(input.Code, secrets)
		if err != nil {
			return err
		}
		next := storedChannel{channel: smsmodel.Channel{
			Code: input.Code, Name: input.Name, Driver: input.Driver, Region: input.Region, Priority: input.Priority,
			Enabled: current == nil || current.channel.Enabled, Lifecycle: smsmodel.LifecycleActive,
			PublicConfig: input.PublicConfig, SecretMasks: masks, CredentialKeyID: keyID, Version: 1,
		}, secrets: secrets, sealed: sealed}
		if current != nil {
			next.channel.ID = current.channel.ID
			next.channel.Version = current.channel.Version + 1
			next.channel.CreatedAt = current.channel.CreatedAt
			if err := updateChannelAndSnapshot(ctx, tx, next); err != nil {
				return err
			}
		} else if err := insertChannelAndSnapshot(ctx, tx, &next); err != nil {
			return err
		}
		if err := insertSMSCommand(ctx, tx, input.CommandKey, requestHash, "UPSERT_CHANNEL", "CHANNEL", input.Code, next.channel.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "sms.channel.upsert", ResourceType: "platform.sms.channel", ResourceID: input.Code, Details: map[string]any{"version": next.channel.Version, "driver": input.Driver}}); err != nil {
			return err
		}
		output = next.channel
		return bumpSMSCatalogue(ctx, tx)
	})
	return output, err
}

func (r *SMSRepository) SetChannelEnabled(ctx context.Context, scope smsmodel.Scope, input smsmodel.SetEnabled, requestHash string) (smsmodel.Channel, error) {
	var output smsmodel.Channel
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replaySMSCommand(ctx, tx, input.CommandKey, requestHash, "CHANNEL"); err != nil {
			return err
		} else if found {
			output = replay.(smsmodel.Channel)
			return nil
		}
		current, err := requireActiveChannel(ctx, tx, r, input.Code, input.ExpectedVersion)
		if err != nil {
			return err
		}
		current.channel.Enabled = input.Enabled
		current.channel.Version++
		if err := updateChannelAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if err := insertSMSCommand(ctx, tx, input.CommandKey, requestHash, "ENABLE_CHANNEL", "CHANNEL", input.Code, current.channel.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "sms.channel.enabled", ResourceType: "platform.sms.channel", ResourceID: input.Code, Details: map[string]any{"version": current.channel.Version, "enabled": input.Enabled}}); err != nil {
			return err
		}
		output = current.channel
		return bumpSMSCatalogue(ctx, tx)
	})
	return output, err
}

func (r *SMSRepository) RetireChannel(ctx context.Context, scope smsmodel.Scope, input smsmodel.Retire, requestHash string) (smsmodel.Channel, error) {
	var output smsmodel.Channel
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replaySMSCommand(ctx, tx, input.CommandKey, requestHash, "CHANNEL"); err != nil {
			return err
		} else if found {
			output = replay.(smsmodel.Channel)
			return nil
		}
		current, err := requireActiveChannel(ctx, tx, r, input.Code, input.ExpectedVersion)
		if err != nil {
			return err
		}
		current.channel.Lifecycle = smsmodel.LifecycleRetired
		current.channel.Enabled = false
		current.channel.Version++
		if err := updateChannelAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if err := insertSMSCommand(ctx, tx, input.CommandKey, requestHash, "RETIRE_CHANNEL", "CHANNEL", input.Code, current.channel.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "sms.channel.retire", ResourceType: "platform.sms.channel", ResourceID: input.Code, Details: map[string]any{"version": current.channel.Version}}); err != nil {
			return err
		}
		output = current.channel
		return bumpSMSCatalogue(ctx, tx)
	})
	return output, err
}

func (r *SMSRepository) LoadChannelSecrets(ctx context.Context, scope smsmodel.Scope, code string) (smsmodel.ChannelSecrets, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,code,name,driver,region,priority,enabled,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks,version,created_at,updated_at
		FROM sms_channel WHERE code=?`, code)
	stored, err := r.scanStoredChannel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return smsmodel.ChannelSecrets{}, smsmodel.ErrNotFound
	}
	if err != nil {
		return smsmodel.ChannelSecrets{}, err
	}
	config := map[string]string{}
	for key, value := range stored.channel.PublicConfig {
		config[key] = value
	}
	for key, value := range stored.secrets {
		config[key] = value
	}
	return smsmodel.ChannelSecrets{Channel: stored.channel, Config: config}, nil
}

func (r *SMSRepository) ListRegions(ctx context.Context, scope smsmodel.Scope, filter smsmodel.RegionFilter) ([]smsmodel.Region, error) {
	query := `SELECT id,code,dial_code,name,iso2,emoji,sort_order,enabled,lifecycle,version,created_at,updated_at FROM sms_region WHERE 1=1`
	args := []any{}
	if filter.Keyword != "" {
		query += ` AND (code LIKE ? OR dial_code LIKE ? OR name LIKE ? OR iso2 LIKE ?)`
		keyword := "%" + filter.Keyword + "%"
		args = append(args, keyword, keyword, keyword, keyword)
	}
	if filter.Lifecycle != "" {
		query += ` AND lifecycle=?`
		args = append(args, filter.Lifecycle)
	}
	query += ` ORDER BY lifecycle='ACTIVE' DESC,sort_order ASC,id ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]smsmodel.Region, 0)
	for rows.Next() {
		item, err := scanRegion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SMSRepository) UpsertRegion(ctx context.Context, scope smsmodel.Scope, input smsmodel.UpsertRegion, requestHash string) (smsmodel.Region, error) {
	var output smsmodel.Region
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replaySMSCommand(ctx, tx, input.CommandKey, requestHash, "REGION"); err != nil {
			return err
		} else if found {
			output = replay.(smsmodel.Region)
			return nil
		}
		current, err := lockRegion(ctx, tx, input.Code)
		if err != nil {
			return err
		}
		if current == nil && input.ExpectedVersion != 0 {
			return smsmodel.ErrConflict
		}
		if current != nil {
			if current.Lifecycle == smsmodel.LifecycleRetired {
				return smsmodel.ErrRetired
			}
			if current.Version != input.ExpectedVersion {
				return smsmodel.ErrConflict
			}
		}
		next := smsmodel.Region{Code: input.Code, DialCode: input.DialCode, Name: input.Name, ISO2: input.ISO2, Emoji: input.Emoji, Sort: input.Sort, Enabled: current == nil || current.Enabled, Lifecycle: smsmodel.LifecycleActive, Version: 1}
		if current != nil {
			next.ID = current.ID
			next.Version = current.Version + 1
			next.CreatedAt = current.CreatedAt
			if err := updateRegionAndSnapshot(ctx, tx, next); err != nil {
				return err
			}
		} else if err := insertRegionAndSnapshot(ctx, tx, &next); err != nil {
			return err
		}
		if err := insertSMSCommand(ctx, tx, input.CommandKey, requestHash, "UPSERT_REGION", "REGION", input.Code, next.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "sms.region.upsert", ResourceType: "platform.sms.region", ResourceID: input.Code, Details: map[string]any{"version": next.Version, "dialCode": input.DialCode}}); err != nil {
			return err
		}
		output = next
		return bumpSMSCatalogue(ctx, tx)
	})
	return output, err
}

func (r *SMSRepository) SetRegionEnabled(ctx context.Context, scope smsmodel.Scope, input smsmodel.SetEnabled, requestHash string) (smsmodel.Region, error) {
	var output smsmodel.Region
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replaySMSCommand(ctx, tx, input.CommandKey, requestHash, "REGION"); err != nil {
			return err
		} else if found {
			output = replay.(smsmodel.Region)
			return nil
		}
		current, err := requireActiveRegion(ctx, tx, input.Code, input.ExpectedVersion)
		if err != nil {
			return err
		}
		current.Enabled = input.Enabled
		current.Version++
		if err := updateRegionAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if err := insertSMSCommand(ctx, tx, input.CommandKey, requestHash, "ENABLE_REGION", "REGION", input.Code, current.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "sms.region.enabled", ResourceType: "platform.sms.region", ResourceID: input.Code, Details: map[string]any{"version": current.Version, "enabled": input.Enabled}}); err != nil {
			return err
		}
		output = *current
		return bumpSMSCatalogue(ctx, tx)
	})
	return output, err
}

func (r *SMSRepository) RetireRegion(ctx context.Context, scope smsmodel.Scope, input smsmodel.Retire, requestHash string) (smsmodel.Region, error) {
	var output smsmodel.Region
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replaySMSCommand(ctx, tx, input.CommandKey, requestHash, "REGION"); err != nil {
			return err
		} else if found {
			output = replay.(smsmodel.Region)
			return nil
		}
		current, err := requireActiveRegion(ctx, tx, input.Code, input.ExpectedVersion)
		if err != nil {
			return err
		}
		var referenced int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sms_channel WHERE lifecycle='ACTIVE' AND region=?`, current.DialCode).Scan(&referenced); err != nil {
			return err
		}
		if referenced > 0 {
			return smsmodel.ErrInUse
		}
		current.Lifecycle = smsmodel.LifecycleRetired
		current.Enabled = false
		current.Version++
		if err := updateRegionAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if err := insertSMSCommand(ctx, tx, input.CommandKey, requestHash, "RETIRE_REGION", "REGION", input.Code, current.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "sms.region.retire", ResourceType: "platform.sms.region", ResourceID: input.Code, Details: map[string]any{"version": current.Version}}); err != nil {
			return err
		}
		output = *current
		return bumpSMSCatalogue(ctx, tx)
	})
	return output, err
}

func (r *SMSRepository) GetMerchantGrant(ctx context.Context, scope smsmodel.Scope, merchantID, shopID int64) (smsmodel.MerchantGrant, error) {
	item, err := scanGrant(r.db.QueryRowContext(ctx, `SELECT id,merchant_id,shop_id,dial_codes,unrestricted,version,created_at,updated_at FROM sms_merchant_grant WHERE merchant_id=? AND shop_id=?`, merchantID, shopID))
	if errors.Is(err, sql.ErrNoRows) {
		return smsmodel.MerchantGrant{MerchantID: merchantID, ShopID: shopID, DialCodes: []string{}, Unrestricted: true}, nil
	}
	return item, err
}

func (r *SMSRepository) PutMerchantGrant(ctx context.Context, scope smsmodel.Scope, input smsmodel.PutMerchantGrant, requestHash string) (smsmodel.MerchantGrant, error) {
	var output smsmodel.MerchantGrant
	resourceID := smsmodel.GrantResourceID(input.MerchantID, input.ShopID)
	err := r.withCatalogue(ctx, scope, func(ctx context.Context, tx *sql.Tx) error {
		if replay, found, err := replaySMSCommand(ctx, tx, input.CommandKey, requestHash, "GRANT"); err != nil {
			return err
		} else if found {
			output = replay.(smsmodel.MerchantGrant)
			return nil
		}
		for _, dial := range input.DialCodes {
			if _, err := lockActiveRegionByDial(ctx, tx, dial); err != nil {
				return err
			}
		}
		current, err := lockGrant(ctx, tx, input.MerchantID, input.ShopID)
		if err != nil {
			return err
		}
		if current == nil && input.ExpectedVersion != 0 {
			return smsmodel.ErrConflict
		}
		if current != nil && current.Version != input.ExpectedVersion {
			return smsmodel.ErrConflict
		}
		next := smsmodel.MerchantGrant{MerchantID: input.MerchantID, ShopID: input.ShopID, DialCodes: append([]string(nil), input.DialCodes...), Unrestricted: len(input.DialCodes) == 0, Version: 1}
		if current != nil {
			next.ID = current.ID
			next.Version = current.Version + 1
			next.CreatedAt = current.CreatedAt
			if err := updateGrantAndSnapshot(ctx, tx, next); err != nil {
				return err
			}
		} else if err := insertGrantAndSnapshot(ctx, tx, &next); err != nil {
			return err
		}
		if err := insertSMSCommand(ctx, tx, input.CommandKey, requestHash, "PUT_GRANT", "GRANT", resourceID, next.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "sms.grant.put", ResourceType: "platform.sms.grant", ResourceID: resourceID, Details: map[string]any{"version": next.Version, "unrestricted": next.Unrestricted}}); err != nil {
			return err
		}
		output = next
		return bumpSMSCatalogue(ctx, tx)
	})
	return output, err
}

type storedChannel struct {
	channel smsmodel.Channel
	secrets map[string]string
	sealed  []byte
}

func (r *SMSRepository) withCatalogue(ctx context.Context, scope smsmodel.Scope, operation func(context.Context, *sql.Tx) error) error {
	return transaction(ctx, r.db, smsTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO sms_catalogue(id,revision) VALUES(1,0) ON DUPLICATE KEY UPDATE id=id`); err != nil {
			return err
		}
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM sms_catalogue WHERE id=1 FOR UPDATE`).Scan(&revision); err != nil {
			return err
		}
		return operation(ctx, tx)
	})
}

func (r *SMSRepository) sealChannel(code string, secrets map[string]string) ([]byte, map[string]string, string, error) {
	masks := smsmodel.MaskSecrets(secrets)
	if len(secrets) == 0 {
		return nil, masks, "", nil
	}
	plain, err := json.Marshal(secrets)
	if err != nil {
		return nil, nil, "", err
	}
	sealed, err := r.box.Seal(plain, []byte(fmt.Sprintf("sms-channel:%d:%s", 0, code)))
	return sealed, masks, r.box.KeyID(), err
}

func (r *SMSRepository) openChannel(code, keyID string, sealed []byte) (map[string]string, error) {
	if len(sealed) == 0 {
		return map[string]string{}, nil
	}
	if keyID != r.box.KeyID() {
		return nil, errors.New("sms channel credential key is unavailable")
	}
	plain, err := r.box.Open(sealed, []byte(fmt.Sprintf("sms-channel:%d:%s", 0, code)))
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

func (r *SMSRepository) scanStoredChannel(row scanner) (storedChannel, error) {
	var item storedChannel
	var publicRaw, masksRaw []byte
	var keyID string
	err := row.Scan(&item.channel.ID, &item.channel.Code, &item.channel.Name, &item.channel.Driver, &item.channel.Region, &item.channel.Priority, &item.channel.Enabled, &item.channel.Lifecycle, &publicRaw, &item.sealed, &keyID, &masksRaw, &item.channel.Version, &item.channel.CreatedAt, &item.channel.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(publicRaw, &item.channel.PublicConfig); err != nil {
		return item, err
	}
	if item.channel.PublicConfig == nil {
		item.channel.PublicConfig = map[string]string{}
	}
	if err := json.Unmarshal(masksRaw, &item.channel.SecretMasks); err != nil {
		return item, err
	}
	if item.channel.SecretMasks == nil {
		item.channel.SecretMasks = map[string]string{}
	}
	item.channel.CredentialKeyID = keyID
	item.secrets, err = r.openChannel(item.channel.Code, keyID, item.sealed)
	return item, err
}

func scanPublicChannel(row scanner) (smsmodel.Channel, error) {
	var item smsmodel.Channel
	var publicRaw, masksRaw []byte
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Driver, &item.Region, &item.Priority, &item.Enabled, &item.Lifecycle, &publicRaw, &item.CredentialKeyID, &masksRaw, &item.Version, &item.CreatedAt, &item.UpdatedAt)
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

func scanRegion(row scanner) (smsmodel.Region, error) {
	var item smsmodel.Region
	err := row.Scan(&item.ID, &item.Code, &item.DialCode, &item.Name, &item.ISO2, &item.Emoji, &item.Sort, &item.Enabled, &item.Lifecycle, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanGrant(row scanner) (smsmodel.MerchantGrant, error) {
	var item smsmodel.MerchantGrant
	var codesRaw []byte
	err := row.Scan(&item.ID, &item.MerchantID, &item.ShopID, &codesRaw, &item.Unrestricted, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(codesRaw, &item.DialCodes); err != nil {
		return item, err
	}
	if item.DialCodes == nil {
		item.DialCodes = []string{}
	}
	return item, nil
}

func lockChannel(ctx context.Context, tx *sql.Tx, repo *SMSRepository, code string) (*storedChannel, error) {
	item, err := repo.scanStoredChannel(tx.QueryRowContext(ctx, `SELECT id,code,name,driver,region,priority,enabled,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks,version,created_at,updated_at FROM sms_channel WHERE code=? FOR UPDATE`, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func requireActiveChannel(ctx context.Context, tx *sql.Tx, repo *SMSRepository, code string, expected int64) (*storedChannel, error) {
	current, err := lockChannel(ctx, tx, repo, code)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, smsmodel.ErrNotFound
	}
	if current.channel.Lifecycle == smsmodel.LifecycleRetired {
		return nil, smsmodel.ErrRetired
	}
	if current.channel.Version != expected {
		return nil, smsmodel.ErrConflict
	}
	return current, nil
}

func lockRegion(ctx context.Context, tx *sql.Tx, code string) (*smsmodel.Region, error) {
	item, err := scanRegion(tx.QueryRowContext(ctx, `SELECT id,code,dial_code,name,iso2,emoji,sort_order,enabled,lifecycle,version,created_at,updated_at FROM sms_region WHERE code=? FOR UPDATE`, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func requireActiveRegion(ctx context.Context, tx *sql.Tx, code string, expected int64) (*smsmodel.Region, error) {
	current, err := lockRegion(ctx, tx, code)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, smsmodel.ErrNotFound
	}
	if current.Lifecycle == smsmodel.LifecycleRetired {
		return nil, smsmodel.ErrRetired
	}
	if current.Version != expected {
		return nil, smsmodel.ErrConflict
	}
	return current, nil
}

func lockActiveRegionByDial(ctx context.Context, tx *sql.Tx, dial string) (smsmodel.Region, error) {
	item, err := scanRegion(tx.QueryRowContext(ctx, `SELECT id,code,dial_code,name,iso2,emoji,sort_order,enabled,lifecycle,version,created_at,updated_at FROM sms_region WHERE dial_code=? FOR UPDATE`, dial))
	if errors.Is(err, sql.ErrNoRows) {
		return item, smsmodel.ErrInvalid
	}
	if err != nil {
		return item, err
	}
	if item.Lifecycle != smsmodel.LifecycleActive || !item.Enabled {
		return item, smsmodel.ErrInvalid
	}
	return item, nil
}

func lockGrant(ctx context.Context, tx *sql.Tx, merchantID, shopID int64) (*smsmodel.MerchantGrant, error) {
	item, err := scanGrant(tx.QueryRowContext(ctx, `SELECT id,merchant_id,shop_id,dial_codes,unrestricted,version,created_at,updated_at FROM sms_merchant_grant WHERE merchant_id=? AND shop_id=? FOR UPDATE`, merchantID, shopID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func insertChannelAndSnapshot(ctx context.Context, tx *sql.Tx, item *storedChannel) error {
	publicRaw, masksRaw, err := channelJSON(*item)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO sms_channel(code,name,driver,region,priority,enabled,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks,version)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, item.channel.Code, item.channel.Name, item.channel.Driver, item.channel.Region, item.channel.Priority, item.channel.Enabled, item.channel.Lifecycle, publicRaw, nullableBytes(item.sealed), item.channel.CredentialKeyID, masksRaw, item.channel.Version)
	if err != nil {
		return err
	}
	item.channel.ID, _ = result.LastInsertId()
	if err := insertChannelSnapshot(ctx, tx, *item, publicRaw, masksRaw); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT created_at,updated_at FROM sms_channel WHERE id=?`, item.channel.ID).Scan(&item.channel.CreatedAt, &item.channel.UpdatedAt)
}

func updateChannelAndSnapshot(ctx context.Context, tx *sql.Tx, item storedChannel) error {
	publicRaw, masksRaw, err := channelJSON(item)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE sms_channel SET name=?,driver=?,region=?,priority=?,enabled=?,lifecycle=?,public_config=?,credential_ciphertext=?,credential_key_id=?,credential_masks=?,version=?,updated_at=CURRENT_TIMESTAMP(3)
		WHERE code=? AND version=?`, item.channel.Name, item.channel.Driver, item.channel.Region, item.channel.Priority, item.channel.Enabled, item.channel.Lifecycle, publicRaw, nullableBytes(item.sealed), item.channel.CredentialKeyID, masksRaw, item.channel.Version, item.channel.Code, item.channel.Version-1)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return smsmodel.ErrConflict
	}
	if err := insertChannelSnapshot(ctx, tx, item, publicRaw, masksRaw); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT updated_at FROM sms_channel WHERE code=?`, item.channel.Code).Scan(&item.channel.UpdatedAt)
}

func insertChannelSnapshot(ctx context.Context, tx *sql.Tx, item storedChannel, publicRaw, masksRaw []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO sms_channel_version(channel_code,version,name,driver,region,priority,enabled,lifecycle,public_config,credential_ciphertext,credential_key_id,credential_masks)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, item.channel.Code, item.channel.Version, item.channel.Name, item.channel.Driver, item.channel.Region, item.channel.Priority, item.channel.Enabled, item.channel.Lifecycle, publicRaw, nullableBytes(item.sealed), item.channel.CredentialKeyID, masksRaw)
	return err
}

func insertRegionAndSnapshot(ctx context.Context, tx *sql.Tx, item *smsmodel.Region) error {
	result, err := tx.ExecContext(ctx, `INSERT INTO sms_region(code,dial_code,name,iso2,emoji,sort_order,enabled,lifecycle,version) VALUES(?,?,?,?,?,?,?,?,?)`, item.Code, item.DialCode, item.Name, item.ISO2, item.Emoji, item.Sort, item.Enabled, item.Lifecycle, item.Version)
	if err != nil {
		return err
	}
	item.ID, _ = result.LastInsertId()
	if err := insertRegionSnapshot(ctx, tx, *item); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT created_at,updated_at FROM sms_region WHERE id=?`, item.ID).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func updateRegionAndSnapshot(ctx context.Context, tx *sql.Tx, item smsmodel.Region) error {
	result, err := tx.ExecContext(ctx, `UPDATE sms_region SET dial_code=?,name=?,iso2=?,emoji=?,sort_order=?,enabled=?,lifecycle=?,version=?,updated_at=CURRENT_TIMESTAMP(3) WHERE code=? AND version=?`, item.DialCode, item.Name, item.ISO2, item.Emoji, item.Sort, item.Enabled, item.Lifecycle, item.Version, item.Code, item.Version-1)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return smsmodel.ErrConflict
	}
	return insertRegionSnapshot(ctx, tx, item)
}

func insertRegionSnapshot(ctx context.Context, tx *sql.Tx, item smsmodel.Region) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO sms_region_version(region_code,version,dial_code,name,iso2,emoji,sort_order,enabled,lifecycle) VALUES(?,?,?,?,?,?,?,?,?)`, item.Code, item.Version, item.DialCode, item.Name, item.ISO2, item.Emoji, item.Sort, item.Enabled, item.Lifecycle)
	return err
}

func insertGrantAndSnapshot(ctx context.Context, tx *sql.Tx, item *smsmodel.MerchantGrant) error {
	codes, err := json.Marshal(item.DialCodes)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO sms_merchant_grant(merchant_id,shop_id,dial_codes,unrestricted,version) VALUES(?,?,?,?,?)`, item.MerchantID, item.ShopID, codes, item.Unrestricted, item.Version)
	if err != nil {
		return err
	}
	item.ID, _ = result.LastInsertId()
	if err := insertGrantSnapshot(ctx, tx, *item, codes); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT created_at,updated_at FROM sms_merchant_grant WHERE id=?`, item.ID).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func updateGrantAndSnapshot(ctx context.Context, tx *sql.Tx, item smsmodel.MerchantGrant) error {
	codes, err := json.Marshal(item.DialCodes)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE sms_merchant_grant SET dial_codes=?,unrestricted=?,version=?,updated_at=CURRENT_TIMESTAMP(3) WHERE merchant_id=? AND shop_id=? AND version=?`, codes, item.Unrestricted, item.Version, item.MerchantID, item.ShopID, item.Version-1)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return smsmodel.ErrConflict
	}
	return insertGrantSnapshot(ctx, tx, item, codes)
}

func insertGrantSnapshot(ctx context.Context, tx *sql.Tx, item smsmodel.MerchantGrant, codes []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO sms_merchant_grant_version(merchant_id,shop_id,version,dial_codes,unrestricted) VALUES(?,?,?,?,?)`, item.MerchantID, item.ShopID, item.Version, codes, item.Unrestricted)
	return err
}

func channelJSON(item storedChannel) ([]byte, []byte, error) {
	publicRaw, err := json.Marshal(item.channel.PublicConfig)
	if err != nil {
		return nil, nil, err
	}
	masksRaw, err := json.Marshal(item.channel.SecretMasks)
	return publicRaw, masksRaw, err
}

func insertSMSCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash, action, kind, resourceID string, version int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO sms_command(command_key,request_hash,action,resource_kind,resource_id,result_version) VALUES(?,?,?,?,?,?)`, commandKey, requestHash, action, kind, resourceID, version)
	return err
}

func bumpSMSCatalogue(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `UPDATE sms_catalogue SET revision=revision+1,updated_at=CURRENT_TIMESTAMP(3) WHERE id=1`)
	return err
}

func replaySMSCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash, kind string) (any, bool, error) {
	var storedHash, resourceKind, resourceID string
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash,resource_kind,resource_id,result_version FROM sms_command WHERE command_key=? FOR UPDATE`, commandKey).Scan(&storedHash, &resourceKind, &resourceID, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if storedHash != requestHash || resourceKind != kind {
		return nil, false, smsmodel.ErrConflict
	}
	switch kind {
	case "CHANNEL":
		item, err := loadChannelSnapshot(ctx, tx, resourceID, version)
		return item, true, err
	case "REGION":
		item, err := loadRegionSnapshot(ctx, tx, resourceID, version)
		return item, true, err
	case "GRANT":
		item, err := loadGrantSnapshot(ctx, tx, resourceID, version)
		return item, true, err
	default:
		return nil, false, smsmodel.ErrConflict
	}
}

func loadChannelSnapshot(ctx context.Context, tx *sql.Tx, code string, version int64) (smsmodel.Channel, error) {
	item, err := scanPublicChannel(tx.QueryRowContext(ctx, `SELECT COALESCE(c.id,0),v.channel_code,v.name,v.driver,v.region,v.priority,v.enabled,v.lifecycle,v.public_config,v.credential_key_id,v.credential_masks,v.version,COALESCE(c.created_at,v.created_at),v.created_at
		FROM sms_channel_version v LEFT JOIN sms_channel c ON c.code=v.channel_code
		WHERE v.channel_code=? AND v.version=?`, code, version))
	if errors.Is(err, sql.ErrNoRows) {
		return item, smsmodel.ErrConflict
	}
	return item, err
}

func loadRegionSnapshot(ctx context.Context, tx *sql.Tx, code string, version int64) (smsmodel.Region, error) {
	item, err := scanRegion(tx.QueryRowContext(ctx, `SELECT COALESCE(r.id,0),v.region_code,v.dial_code,v.name,v.iso2,v.emoji,v.sort_order,v.enabled,v.lifecycle,v.version,COALESCE(r.created_at,v.created_at),v.created_at
		FROM sms_region_version v LEFT JOIN sms_region r ON r.code=v.region_code
		WHERE v.region_code=? AND v.version=?`, code, version))
	if errors.Is(err, sql.ErrNoRows) {
		return item, smsmodel.ErrConflict
	}
	return item, err
}

func loadGrantSnapshot(ctx context.Context, tx *sql.Tx, resourceID string, version int64) (smsmodel.MerchantGrant, error) {
	var merchantID, shopID int64
	if _, err := fmt.Sscanf(resourceID, "%d:%d", &merchantID, &shopID); err != nil {
		return smsmodel.MerchantGrant{}, smsmodel.ErrConflict
	}
	item, err := scanGrant(tx.QueryRowContext(ctx, `SELECT COALESCE(g.id,0),v.merchant_id,v.shop_id,v.dial_codes,v.unrestricted,v.version,COALESCE(g.created_at,v.created_at),v.created_at
		FROM sms_merchant_grant_version v LEFT JOIN sms_merchant_grant g ON g.merchant_id=v.merchant_id AND g.shop_id=v.shop_id
		WHERE v.merchant_id=? AND v.shop_id=? AND v.version=?`, merchantID, shopID, version))
	if errors.Is(err, sql.ErrNoRows) {
		return item, smsmodel.ErrConflict
	}
	return item, err
}
