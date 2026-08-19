// Package app assembles Platform dependencies and owns the process lifecycle.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/edge"
	edgemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/email"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/notification"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/sms"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/storage"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry"
	"github.com/liveshop-platform/module-platform/internal/common/edgehttp"
	"github.com/liveshop-platform/module-platform/internal/common/emailsender"
	"github.com/liveshop-platform/module-platform/internal/common/notifysender"
	"github.com/liveshop-platform/module-platform/internal/common/smssender"
	"github.com/liveshop-platform/module-platform/internal/common/storagesender"
	"github.com/liveshop-platform/module-platform/internal/config"
	"github.com/liveshop-platform/module-platform/internal/data/mysql"
	"github.com/liveshop-platform/module-platform/internal/data/secretbox"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
)

// assemblyTimeout bounds the startup round trips to the control-plane
// database, so an unreachable host fails instead of hanging the process.
const assemblyTimeout = 10 * time.Second

// Dependencies is the set of use cases and verifiers assembled for one
// process. It is passed explicitly; there is no process-global accessor.
type Dependencies struct {
	Release      *biz.Release
	Settings     *biz.Settings
	Audit        *biz.Audit
	LiveProvider *liveprovider.UseCase
	SMS          *sms.UseCase
	Email        *email.UseCase
	Storage      *storage.UseCase
	Notification *notification.UseCase
	Localization *localization.UseCase
	Telemetry    *telemetry.UseCase
	Edge         *edge.UseCase

	Workloads      *workloadidentity.Verifier
	ModuleVerifier *modulesession.Verifier

	// Ready reports backing-store readiness to the health endpoints.
	Ready func(context.Context) error

	shutdown func() error
}

func (d Dependencies) Close() error {
	if d.shutdown == nil {
		return nil
	}
	return d.shutdown()
}

// NewDependencies builds the object graph for one process: key material, then
// the connection pool, then the storage adapters, then the use cases. It
// returns nothing until every backing store has answered, so the process never
// starts half-assembled.
func NewDependencies(cfg *config.Config) (Dependencies, error) {
	ctx, cancel := context.WithTimeout(context.Background(), assemblyTimeout)
	defer cancel()

	tokens, err := newTokenServices(cfg)
	if err != nil {
		return Dependencies{}, err
	}
	database, err := openDatabase(cfg)
	if err != nil {
		return Dependencies{}, err
	}
	box, err := secretbox.New(cfg.CredentialEncryption.KeyID, cfg.CredentialEncryption.MasterKey)
	if err != nil {
		_ = database.Close()
		return Dependencies{}, fmt.Errorf("platform: credential encryption: %w", err)
	}
	adapters, err := newStores(ctx, database, box)
	if err != nil {
		_ = database.Close()
		return Dependencies{}, err
	}

	smsUse := sms.New(adapters.sms, smssender.New())
	emailUse := email.New(adapters.email, emailsender.New())
	notifyUse := notification.New(adapters.notification, notifysender.Adapter{SMS: smsUse, Email: emailUse})
	release := biz.NewRelease(adapters.release)
	if revision, items, err := release.ActiveCapabilities(ctx); err == nil {
		_ = notifyUse.ProjectCapabilities(ctx, revision, items)
	}
	settings := biz.NewSettings(adapters.settings)
	edgeUse := edge.New(settings, edgehttp.NewIdentityHTTP(cfg.Edge.IdentityOrigin, cfg.InternalGrant.Token), edgehttp.NewCaddyReloader(cfg.Edge.CaddyAdmin, cfg.Edge.Caddyfile, writeCaddyfile), edge.Config{
		Enabled: cfg.Edge.Enabled, GrantToken: cfg.InternalGrant.Token, AskOrigin: cfg.Edge.AskOrigin,
		ACMEEmail: cfg.Edge.ACMEEmail, CaddyfilePath: cfg.Edge.Caddyfile,
		Upstreams: map[string]string{
			edgemodel.TargetShop: cfg.Edge.Upstreams.Shop, edgemodel.TargetLive: cfg.Edge.Upstreams.Live,
			edgemodel.TargetMerch: cfg.Edge.Upstreams.Merch, edgemodel.TargetAdmin: cfg.Edge.Upstreams.Admin,
			edgemodel.TargetRTS: cfg.Edge.Upstreams.RTS, edgemodel.TargetGateway: cfg.Edge.Upstreams.Gateway,
		},
	})
	return Dependencies{
		Release: release, Settings: settings, Audit: biz.NewAudit(adapters.audit), LiveProvider: liveprovider.New(adapters.liveProvider), SMS: smsUse, Email: emailUse, Storage: storage.New(adapters.storage, storagesender.New()), Notification: notifyUse, Localization: localization.New(adapters.localization, localization.NoopTranslator{}), Telemetry: telemetry.New(adapters.telemetry), Edge: edgeUse,
		Workloads: tokens.workloads, ModuleVerifier: tokens.moduleVerifier,
		Ready: database.PingContext, shutdown: database.Close,
	}, nil
}

// tokenServices are the signers and verifiers derived from the configured key
// material. Each failure names the configuration section that produced it.
type tokenServices struct {
	moduleVerifier *modulesession.Verifier
	workloads      *workloadidentity.Verifier
}

func newTokenServices(cfg *config.Config) (tokenServices, error) {
	var services tokenServices
	var err error
	if services.moduleVerifier, err = modulesession.NewVerifier(map[string]string{cfg.ModuleCapability.KeyID: cfg.ModuleCapability.PublicKey}, cfg.ModuleCapability.Issuer, moduleCapabilityAudience); err != nil {
		return tokenServices{}, fmt.Errorf("platform: module_capability public key: %w", err)
	}
	if services.workloads, err = workloadidentity.NewVerifier(trustedWorkloads(cfg), cfg.WorkloadIdentity.Issuer, workloadAudience); err != nil {
		return tokenServices{}, fmt.Errorf("platform: workload_identity keys: %w", err)
	}
	return services, nil
}

const (
	moduleCapabilityAudience = "liveshop-module:platform"
	workloadAudience         = "liveshop-platform-internal"
)

// trustedWorkloads maps the configured peers onto their Ed25519 bearer
// credential. The same peers present a SPIFFE identity on gRPC, which the gRPC
// composition root installs separately.
func trustedWorkloads(cfg *config.Config) map[string]workloadidentity.TrustedWorkload {
	peers := map[string]workloadidentity.TrustedWorkload{}
	for _, peer := range []config.BearerWorkload{cfg.WorkloadIdentity.HTTP.Gateway, cfg.WorkloadIdentity.HTTP.Release, cfg.WorkloadIdentity.HTTP.Identity} {
		peers[peer.KeyID] = workloadidentity.TrustedWorkload{
			PublicKey:   peer.PublicKey,
			Subject:     peer.Subject,
			Permissions: peer.Permissions,
		}
	}
	return peers
}

// stores are the MySQL adapters behind the biz ports.
type stores struct {
	release      *mysql.ReleaseRepository
	settings     *mysql.SettingsRepository
	audit        *mysql.AuditRepository
	liveProvider *mysql.LiveProviderRepository
	sms          *mysql.SMSRepository
	email        *mysql.EmailRepository
	storage      *mysql.StorageRepository
	notification *mysql.NotificationRepository
	localization *mysql.LocalizationRepository
	telemetry    *mysql.TelemetryRepository
}

// newStores gates every adapter behind one reachability check and one schema
// check, instead of letting each constructor probe the same pool.
func newStores(ctx context.Context, database *sql.DB, box *secretbox.Box) (stores, error) {
	if err := mysql.Verify(ctx, database); err != nil {
		return stores{}, fmt.Errorf("platform: database is unreachable: %w", err)
	}
	release, err := mysql.NewReleaseRepository(ctx, database)
	if err != nil {
		return stores{}, fmt.Errorf("platform: module registry state: %w", err)
	}
	return stores{
		release:      release,
		settings:     mysql.NewSettingsRepository(database),
		audit:        mysql.NewAuditRepository(database),
		liveProvider: mysql.NewLiveProviderRepository(database, box),
		sms:          mysql.NewSMSRepository(database, box),
		email:        mysql.NewEmailRepository(database, box),
		storage:      mysql.NewStorageRepository(database, box),
		notification: mysql.NewNotificationRepository(database),
		localization: mysql.NewLocalizationRepository(database, box),
		telemetry:    mysql.NewTelemetryRepository(database),
	}, nil
}

func writeCaddyfile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func openDatabase(cfg *config.Config) (*sql.DB, error) {
	database, err := sql.Open("mysql", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("platform: open database: %w", err)
	}
	database.SetMaxOpenConns(cfg.Database.MaxOpenConnections)
	database.SetMaxIdleConns(cfg.Database.MaxIdleConnections)
	database.SetConnMaxLifetime(cfg.ConnectionMaxLifetime())
	database.SetConnMaxIdleTime(cfg.ConnectionMaxIdleTime())
	return database, nil
}
