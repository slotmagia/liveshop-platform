// Package registry owns process-level dependency initialization and access.
package registry

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/liveshop-platform/module-platform/internal/platform/registry/audit"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/identity"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/module"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/settings"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
)

type SigningConfig struct {
	KeyID      string
	PrivateKey string
	PublicKey  string
	Issuer     string
}

type TrustedWorkload struct {
	KeyID       string
	PublicKey   string
	Subject     string
	Permissions []string
}

type Config struct {
	DatabaseURL           string
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
	ConnectionMaxIdleTime time.Duration
	ModuleSession         SigningConfig
	AccessIdentity        SigningConfig
	WorkloadIssuer        string
	GatewayWorkload       TrustedWorkload
	ReleaseWorkload       TrustedWorkload
}

type Dependencies struct {
	Modules        *module.Store
	IAM            *iam.Store
	Identity       *identity.Service
	Settings       *settings.Store
	Audit          *audit.Store
	Workloads      *workloadidentity.Verifier
	Identities     *accessidentity.Verifier
	ModuleIssuer   *modulesession.Issuer
	ModuleVerifier *modulesession.Verifier
	Ready          func(context.Context) error
}

type runtime struct {
	database     *sql.DB
	dependencies Dependencies
}

var (
	localMu sync.RWMutex
	local   *runtime
)

// Init constructs and registers all process dependencies from validated configuration.
func Init(config Config) (err error) {
	moduleIssuer, err := modulesession.NewIssuer(config.ModuleSession.PrivateKey, config.ModuleSession.KeyID, config.ModuleSession.Issuer)
	if err != nil {
		return err
	}
	moduleVerifier, err := modulesession.NewVerifier(map[string]string{config.ModuleSession.KeyID: config.ModuleSession.PublicKey}, config.ModuleSession.Issuer, "liveshop-module:platform")
	if err != nil {
		return err
	}
	identities, err := accessidentity.NewVerifier(map[string]string{config.AccessIdentity.KeyID: config.AccessIdentity.PublicKey}, config.AccessIdentity.Issuer, "liveshop-platform")
	if err != nil {
		return err
	}
	identityIssuer, err := accessidentity.NewIssuer(config.AccessIdentity.PrivateKey, config.AccessIdentity.KeyID, config.AccessIdentity.Issuer)
	if err != nil {
		return err
	}
	workloads, err := workloadidentity.NewVerifier(map[string]workloadidentity.TrustedWorkload{
		config.GatewayWorkload.KeyID: {
			PublicKey: config.GatewayWorkload.PublicKey, Subject: config.GatewayWorkload.Subject, Permissions: config.GatewayWorkload.Permissions,
		},
		config.ReleaseWorkload.KeyID: {
			PublicKey: config.ReleaseWorkload.PublicKey, Subject: config.ReleaseWorkload.Subject, Permissions: config.ReleaseWorkload.Permissions,
		},
	}, config.WorkloadIssuer, "liveshop-platform-internal")
	if err != nil {
		return err
	}

	database, err := sql.Open("pgx", config.DatabaseURL)
	if err != nil {
		return err
	}
	database.SetMaxOpenConns(config.MaxOpenConnections)
	database.SetMaxIdleConns(config.MaxIdleConnections)
	database.SetConnMaxLifetime(config.ConnectionMaxLifetime)
	database.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)
	initialized := false
	defer func() {
		if !initialized {
			_ = database.Close()
		}
	}()

	modules, err := module.NewPostgresStore(database)
	if err != nil {
		return err
	}
	authorizations, err := iam.NewPostgresStore(database)
	if err != nil {
		return err
	}
	identityService, err := identity.New(database, identityIssuer)
	if err != nil {
		return err
	}
	settingStore, err := settings.New(database)
	if err != nil {
		return err
	}
	auditStore, err := audit.New(database)
	if err != nil {
		return err
	}

	localMu.Lock()
	defer localMu.Unlock()
	if local != nil {
		return fmt.Errorf("platform registry is already initialized")
	}
	local = &runtime{database: database, dependencies: Dependencies{
		Modules: modules, IAM: authorizations, Identity: identityService, Settings: settingStore, Audit: auditStore,
		Workloads: workloads, Identities: identities, ModuleIssuer: moduleIssuer, ModuleVerifier: moduleVerifier,
		Ready: database.PingContext,
	}}
	initialized = true
	return nil
}

func Current() Dependencies {
	localMu.RLock()
	defer localMu.RUnlock()
	if local == nil {
		panic("platform registry is not initialized")
	}
	return local.dependencies
}

func Close() error {
	localMu.Lock()
	defer localMu.Unlock()
	if local == nil {
		return nil
	}
	err := local.database.Close()
	local = nil
	return err
}
