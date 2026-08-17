package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
	"github.com/liveshop-platform/module-platform/internal/data/secretbox"
)

const liveProviderTransactionTimeout = 5 * time.Second

type LiveProviderRepository struct {
	db  *sql.DB
	box *secretbox.Box
}

var _ liveprovider.Repository = (*LiveProviderRepository)(nil)

func NewLiveProviderRepository(db *sql.DB, box *secretbox.Box) *LiveProviderRepository {
	return &LiveProviderRepository{db: db, box: box}
}

type providerConfig struct {
	App          string `json:"app,omitempty"`
	PushDomain   string `json:"pushDomain,omitempty"`
	PullDomain   string `json:"pullDomain,omitempty"`
	AgoraAppID   string `json:"agoraAppId,omitempty"`
	Codec        string `json:"codec,omitempty"`
	IngestDomain string `json:"ingestDomain,omitempty"`
	Region       string `json:"region,omitempty"`
}

type credentialMasks struct {
	Secret         string `json:"secret,omitempty"`
	AppCertificate string `json:"appCertificate,omitempty"`
	CustomerKey    string `json:"customerKey,omitempty"`
	CustomerSecret string `json:"customerSecret,omitempty"`
}

type storedProvider struct {
	provider    providermodel.Provider
	config      providerConfig
	credentials providermodel.Credentials
	sealed      []byte
	masks       credentialMasks
}

func (r *LiveProviderRepository) List(ctx context.Context, scope providermodel.Scope, filter providermodel.Filter) ([]providermodel.Provider, error) {
	query := `SELECT id,code,name,kind,driver,config_json,credential_key_id,credential_masks,ttl_seconds,enabled,is_default,lifecycle,health_status,health_message,health_checked_at,version,created_at,updated_at
		FROM live_provider WHERE 1=1`
	args := []any{}
	if filter.Keyword != "" {
		query += ` AND (code LIKE ? OR name LIKE ?)`
		keyword := "%" + filter.Keyword + "%"
		args = append(args, keyword, keyword)
	}
	if filter.Kind != "" {
		query += ` AND kind=?`
		args = append(args, filter.Kind)
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
	items := make([]providermodel.Provider, 0)
	for rows.Next() {
		item, err := scanPublicProvider(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *LiveProviderRepository) Upsert(ctx context.Context, scope providermodel.Scope, input providermodel.Upsert, requestHash string) (providermodel.Provider, error) {
	var output providermodel.Provider
	err := transaction(ctx, r.db, liveProviderTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockLiveProviderAggregate(ctx, tx); err != nil {
			return err
		}
		if replay, found, err := replayCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			output = replay
			return nil
		}
		rows, err := r.lockProviderCatalogue(ctx, tx)
		if err != nil {
			return err
		}
		var current *storedProvider
		for index := range rows {
			if rows[index].provider.Code == input.Code {
				current = &rows[index]
				break
			}
		}
		if current == nil && input.ExpectedVersion != 0 {
			return providermodel.ErrConflict
		}
		if current != nil {
			if current.provider.Lifecycle == providermodel.LifecycleRetired {
				return providermodel.ErrRetired
			}
			if current.provider.Version != input.ExpectedVersion {
				return providermodel.ErrConflict
			}
		}
		enabled := current == nil || current.provider.Enabled
		if input.IsDefault && !enabled {
			return providermodel.ErrInvalid
		}

		credentials := providermodel.Credentials{}
		if current != nil {
			credentials = current.credentials
		}
		applyCredentialChange(&credentials.Secret, nil, input.Secret)
		applyCredentialChange(&credentials.AppCertificate, nil, input.AppCertificate)
		applyCredentialChange(&credentials.CustomerKey, &credentials.CustomerSecret, input.CustomerCredential)
		kind, _ := providermodel.KindFor(input.Driver)
		if kind == providermodel.KindRTMP {
			credentials.AppCertificate, credentials.CustomerKey, credentials.CustomerSecret = "", "", ""
			if input.Driver == providermodel.DriverStatic {
				credentials.Secret = ""
			}
		} else {
			credentials.Secret = ""
		}
		if enabled && input.Driver == providermodel.DriverAgoraMediaGateway && (credentials.CustomerKey == "" || credentials.CustomerSecret == "") {
			return providermodel.ErrInvalid
		}

		if input.IsDefault {
			for index := range rows {
				other := &rows[index]
				if other.provider.Code != input.Code && other.provider.Lifecycle == providermodel.LifecycleActive && other.provider.IsDefault {
					other.provider.IsDefault = false
					other.provider.Version++
					if err := updateCurrentAndSnapshot(ctx, tx, *other); err != nil {
						return err
					}
				}
			}
		}

		config := providerConfig{App: input.App, PushDomain: input.PushDomain, PullDomain: input.PullDomain, AgoraAppID: input.AgoraAppID, Codec: input.Codec, IngestDomain: input.IngestDomain, Region: input.Region}
		if input.Driver == providermodel.DriverAgoraMediaGateway && config.IngestDomain == "" {
			config.IngestDomain = fmt.Sprintf("rtls-ingress-prod-%s.agoramdn.com:1935", config.Region)
		}
		if kind == providermodel.KindRTMP {
			config.AgoraAppID, config.Codec, config.IngestDomain, config.Region = "", "", "", ""
		} else {
			config.PushDomain, config.PullDomain = "", ""
		}
		sealed, masks, keyID, err := r.seal(input.Code, credentials)
		if err != nil {
			return err
		}
		next := storedProvider{provider: providermodel.Provider{
			Code: input.Code, Name: input.Name, Kind: kind, Driver: input.Driver, App: config.App,
			PushDomain: config.PushDomain, PullDomain: config.PullDomain, AgoraAppID: config.AgoraAppID,
			Codec: config.Codec, IngestDomain: config.IngestDomain, Region: config.Region,
			TTLSeconds: input.TTLSeconds, Enabled: enabled, IsDefault: input.IsDefault,
			Lifecycle: providermodel.LifecycleActive, Health: providermodel.HealthUnknown,
			Version: 1,
		}, config: config, credentials: credentials, sealed: sealed, masks: masks}
		next.provider.Credentials = summary(masks, keyID)
		if current != nil {
			next.provider.ID = current.provider.ID
			next.provider.Version = current.provider.Version + 1
			next.provider.CreatedAt = current.provider.CreatedAt
			next.provider.Health = current.provider.Health
			next.provider.HealthMessage = current.provider.HealthMessage
			next.provider.HealthCheckedAt = current.provider.HealthCheckedAt
			if err := updateCurrentAndSnapshot(ctx, tx, next); err != nil {
				return err
			}
		} else if err := insertCurrentAndSnapshot(ctx, tx, &next); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO live_provider_command(command_key,request_hash,action,provider_code,result_version) VALUES(?,?,?,?,?)`, input.CommandKey, requestHash, "UPSERT", input.Code, next.provider.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "live-provider.upsert", ResourceType: "platform.live-provider", ResourceID: input.Code, Details: map[string]any{"version": next.provider.Version, "driver": input.Driver, "default": input.IsDefault}}); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE live_provider_catalogue SET revision=revision+1,updated_at=CURRENT_TIMESTAMP(3) WHERE id=1`); err != nil {
			return err
		}
		output = next.provider
		return nil
	})
	return output, err
}

func (r *LiveProviderRepository) Retire(ctx context.Context, scope providermodel.Scope, input providermodel.Retire, requestHash string) (providermodel.Provider, error) {
	var output providermodel.Provider
	err := transaction(ctx, r.db, liveProviderTransactionTimeout, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockLiveProviderAggregate(ctx, tx); err != nil {
			return err
		}
		if replay, found, err := replayCommand(ctx, tx, input.CommandKey, requestHash); err != nil {
			return err
		} else if found {
			output = replay
			return nil
		}
		rows, err := r.lockProviderCatalogue(ctx, tx)
		if err != nil {
			return err
		}
		var current *storedProvider
		for index := range rows {
			if rows[index].provider.Code == input.Code {
				current = &rows[index]
				break
			}
		}
		if current == nil {
			return providermodel.ErrNotFound
		}
		if current.provider.Lifecycle == providermodel.LifecycleRetired {
			return providermodel.ErrRetired
		}
		if current.provider.Version != input.ExpectedVersion {
			return providermodel.ErrConflict
		}
		current.provider.Version++
		current.provider.Lifecycle = providermodel.LifecycleRetired
		current.provider.Enabled = false
		current.provider.IsDefault = false
		if err := updateCurrentAndSnapshot(ctx, tx, *current); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO live_provider_command(command_key,request_hash,action,provider_code,result_version) VALUES(?,?,?,?,?)`, input.CommandKey, requestHash, "RETIRE", input.Code, current.provider.Version); err != nil {
			return err
		}
		if err := insertAuditEvent(ctx, tx, auditEntry{Realm: scope.Realm, MerchantID: scope.MerchantID, ActorSubject: scope.Subject, Action: "live-provider.retire", ResourceType: "platform.live-provider", ResourceID: input.Code, Details: map[string]any{"version": current.provider.Version}}); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE live_provider_catalogue SET revision=revision+1,updated_at=CURRENT_TIMESTAMP(3) WHERE id=1`); err != nil {
			return err
		}
		output = current.provider
		return nil
	})
	return output, err
}

func (r *LiveProviderRepository) seal(code string, credentials providermodel.Credentials) ([]byte, credentialMasks, string, error) {
	masks := credentialMasks{Secret: providermodel.MaskSecret(credentials.Secret), AppCertificate: providermodel.MaskSecret(credentials.AppCertificate), CustomerKey: providermodel.MaskSecret(credentials.CustomerKey), CustomerSecret: providermodel.MaskSecret(credentials.CustomerSecret)}
	if credentials == (providermodel.Credentials{}) {
		return nil, masks, "", nil
	}
	plain, err := json.Marshal(credentials)
	if err != nil {
		return nil, masks, "", err
	}
	sealed, err := r.box.Seal(plain, providerAAD(0, code))
	return sealed, masks, r.box.KeyID(), err
}

func (r *LiveProviderRepository) open(code, keyID string, sealed []byte) (providermodel.Credentials, error) {
	if len(sealed) == 0 {
		return providermodel.Credentials{}, nil
	}
	if keyID != r.box.KeyID() {
		return providermodel.Credentials{}, errors.New("live provider credential key is unavailable")
	}
	plain, err := r.box.Open(sealed, providerAAD(0, code))
	if err != nil {
		return providermodel.Credentials{}, err
	}
	var credentials providermodel.Credentials
	err = json.Unmarshal(plain, &credentials)
	return credentials, err
}

func providerAAD(appID int64, code string) []byte {
	return []byte(fmt.Sprintf("live-provider:%d:%s", appID, code))
}

func lockLiveProviderAggregate(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO live_provider_catalogue(id,revision) VALUES(1,0) ON DUPLICATE KEY UPDATE id=id`); err != nil {
		return err
	}
	var revision int64
	return tx.QueryRowContext(ctx, `SELECT revision FROM live_provider_catalogue WHERE id=1 FOR UPDATE`).Scan(&revision)
}

func applyCredentialChange(primary *string, secondary *string, change providermodel.CredentialChange) {
	switch change.Mode {
	case providermodel.SecretReplace:
		*primary = change.Value
		if secondary != nil {
			*secondary = change.SecondaryValue
		}
	case providermodel.SecretClear:
		*primary = ""
		if secondary != nil {
			*secondary = ""
		}
	}
}

func (r *LiveProviderRepository) scanStored(rows *sql.Rows) (storedProvider, error) {
	var item storedProvider
	var configRaw, masksRaw []byte
	var keyID string
	var enabled, isDefault bool
	var checked sql.NullTime
	err := rows.Scan(&item.provider.ID, &item.provider.Code, &item.provider.Name, &item.provider.Kind, &item.provider.Driver, &configRaw, &item.sealed, &keyID, &masksRaw, &item.provider.TTLSeconds, &enabled, &isDefault, &item.provider.Lifecycle, &item.provider.Health, &item.provider.HealthMessage, &checked, &item.provider.Version, &item.provider.CreatedAt, &item.provider.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.provider.Enabled, item.provider.IsDefault = enabled, isDefault
	if checked.Valid {
		item.provider.HealthCheckedAt = &checked.Time
	}
	if err := json.Unmarshal(configRaw, &item.config); err != nil {
		return item, err
	}
	if err := json.Unmarshal(masksRaw, &item.masks); err != nil {
		return item, err
	}
	item.credentials, err = r.open(item.provider.Code, keyID, item.sealed)
	if err != nil {
		return item, err
	}
	item.provider = hydrate(item.provider, item.config, item.masks, keyID)
	return item, nil
}

func (r *LiveProviderRepository) lockProviderCatalogue(ctx context.Context, tx *sql.Tx) ([]storedProvider, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,code,name,kind,driver,config_json,credential_ciphertext,credential_key_id,credential_masks,ttl_seconds,enabled,is_default,lifecycle,health_status,health_message,health_checked_at,version,created_at,updated_at FROM live_provider ORDER BY id FOR UPDATE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]storedProvider, 0)
	for rows.Next() {
		item, err := r.scanStored(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanPublicProvider(row scanner) (providermodel.Provider, error) {
	var item providermodel.Provider
	var configRaw, masksRaw []byte
	var masks credentialMasks
	var config providerConfig
	var keyID string
	var enabled, isDefault bool
	var checked sql.NullTime
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Kind, &item.Driver, &configRaw, &keyID, &masksRaw, &item.TTLSeconds, &enabled, &isDefault, &item.Lifecycle, &item.Health, &item.HealthMessage, &checked, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.Enabled, item.IsDefault = enabled, isDefault
	if checked.Valid {
		item.HealthCheckedAt = &checked.Time
	}
	if err := json.Unmarshal(configRaw, &config); err != nil {
		return item, err
	}
	if err := json.Unmarshal(masksRaw, &masks); err != nil {
		return item, err
	}
	return hydrate(item, config, masks, keyID), nil
}

func hydrate(item providermodel.Provider, config providerConfig, masks credentialMasks, keyID string) providermodel.Provider {
	item.App, item.PushDomain, item.PullDomain = config.App, config.PushDomain, config.PullDomain
	item.AgoraAppID, item.Codec = config.AgoraAppID, config.Codec
	item.IngestDomain, item.Region = config.IngestDomain, config.Region
	item.Credentials = summary(masks, keyID)
	return item
}

func summary(masks credentialMasks, keyID string) providermodel.CredentialSummary {
	return providermodel.CredentialSummary{
		SecretSet: masks.Secret != "", SecretMask: masks.Secret,
		AppCertificateSet: masks.AppCertificate != "", AppCertificateMask: masks.AppCertificate,
		CustomerCredentialSet: masks.CustomerKey != "" && masks.CustomerSecret != "",
		CustomerKeyMask:       masks.CustomerKey, CustomerSecretMask: masks.CustomerSecret, KeyID: keyID,
	}
}

func providerJSON(item storedProvider) ([]byte, []byte, error) {
	config, err := json.Marshal(item.config)
	if err != nil {
		return nil, nil, err
	}
	masks, err := json.Marshal(item.masks)
	return config, masks, err
}

func insertCurrentAndSnapshot(ctx context.Context, tx *sql.Tx, item *storedProvider) error {
	config, masks, err := providerJSON(*item)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO live_provider(code,name,kind,driver,config_json,credential_ciphertext,credential_key_id,credential_masks,ttl_seconds,enabled,is_default,lifecycle,health_status,health_message,health_checked_at,version)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.provider.Code, item.provider.Name, item.provider.Kind, item.provider.Driver, config, nullableBytes(item.sealed), item.provider.Credentials.KeyID, masks, item.provider.TTLSeconds, item.provider.Enabled, item.provider.IsDefault, item.provider.Lifecycle, item.provider.Health, item.provider.HealthMessage, nullableTime(item.provider.HealthCheckedAt), item.provider.Version)
	if err != nil {
		return err
	}
	item.provider.ID, _ = result.LastInsertId()
	if err := insertSnapshot(ctx, tx, *item, config, masks); err != nil {
		return err
	}
	return tx.QueryRowContext(ctx, `SELECT created_at,updated_at FROM live_provider WHERE id=?`, item.provider.ID).Scan(&item.provider.CreatedAt, &item.provider.UpdatedAt)
}

func updateCurrentAndSnapshot(ctx context.Context, tx *sql.Tx, item storedProvider) error {
	config, masks, err := providerJSON(item)
	if err != nil {
		return err
	}
	expected := item.provider.Version - 1
	result, err := tx.ExecContext(ctx, `UPDATE live_provider SET name=?,kind=?,driver=?,config_json=?,credential_ciphertext=?,credential_key_id=?,credential_masks=?,ttl_seconds=?,enabled=?,is_default=?,lifecycle=?,health_status=?,health_message=?,health_checked_at=?,version=?,updated_at=CURRENT_TIMESTAMP(3)
		WHERE code=? AND version=?`, item.provider.Name, item.provider.Kind, item.provider.Driver, config, nullableBytes(item.sealed), item.provider.Credentials.KeyID, masks, item.provider.TTLSeconds, item.provider.Enabled, item.provider.IsDefault, item.provider.Lifecycle, item.provider.Health, item.provider.HealthMessage, nullableTime(item.provider.HealthCheckedAt), item.provider.Version, item.provider.Code, expected)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return providermodel.ErrConflict
	}
	return insertSnapshot(ctx, tx, item, config, masks)
}

func insertSnapshot(ctx context.Context, tx *sql.Tx, item storedProvider, config, masks []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO live_provider_version(provider_code,version,name,kind,driver,config_json,credential_ciphertext,credential_key_id,credential_masks,ttl_seconds,enabled,is_default,lifecycle,health_status,health_message,health_checked_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.provider.Code, item.provider.Version, item.provider.Name, item.provider.Kind, item.provider.Driver, config, nullableBytes(item.sealed), item.provider.Credentials.KeyID, masks, item.provider.TTLSeconds, item.provider.Enabled, item.provider.IsDefault, item.provider.Lifecycle, item.provider.Health, item.provider.HealthMessage, nullableTime(item.provider.HealthCheckedAt))
	return err
}

func replayCommand(ctx context.Context, tx *sql.Tx, commandKey, requestHash string) (providermodel.Provider, bool, error) {
	var storedHash, code string
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash,provider_code,result_version FROM live_provider_command WHERE command_key=? FOR UPDATE`, commandKey).Scan(&storedHash, &code, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return providermodel.Provider{}, false, nil
	}
	if err != nil {
		return providermodel.Provider{}, false, err
	}
	if storedHash != requestHash {
		return providermodel.Provider{}, false, providermodel.ErrConflict
	}
	item, err := loadSnapshot(ctx, tx, code, version)
	return item, true, err
}

func loadSnapshot(ctx context.Context, tx *sql.Tx, code string, version int64) (providermodel.Provider, error) {
	row := tx.QueryRowContext(ctx, `SELECT p.id,v.provider_code,v.name,v.kind,v.driver,v.config_json,v.credential_key_id,v.credential_masks,v.ttl_seconds,v.enabled,v.is_default,v.lifecycle,v.health_status,v.health_message,v.health_checked_at,v.version,COALESCE(p.created_at,v.created_at),v.created_at
		FROM live_provider_version v
		LEFT JOIN live_provider p ON p.code=v.provider_code
		WHERE v.provider_code=? AND v.version=?`, code, version)
	item, err := scanPublicProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		return item, providermodel.ErrConflict
	}
	return item, err
}

func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
