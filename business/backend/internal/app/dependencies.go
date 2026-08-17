// Package app assembles Platform dependencies and owns the process lifecycle.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/email"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/sms"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/storage"
	"github.com/liveshop-platform/module-platform/internal/common/emailsender"
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

	release := biz.NewRelease(adapters.release)
	return Dependencies{
		Release: release, Settings: biz.NewSettings(adapters.settings), Audit: biz.NewAudit(adapters.audit), LiveProvider: liveprovider.New(adapters.liveProvider), SMS: sms.New(adapters.sms, smssender.New()), Email: email.New(adapters.email, emailsender.New()), Storage: storage.New(adapters.storage, storagesender.New()),
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
	for _, peer := range []config.BearerWorkload{cfg.WorkloadIdentity.HTTP.Gateway, cfg.WorkloadIdentity.HTTP.Release} {
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
	}, nil
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
