package app

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/logic"
	"github.com/liveshop-platform/module-platform/internal/platform/common/grpcauth"
	platformgrpc "github.com/liveshop-platform/module-platform/internal/platform/common/grpcserver"
	platformserver "github.com/liveshop-platform/module-platform/internal/platform/common/server"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry"
	commonconfig "github.com/liveshop-platform/module-platform/pkg/config"
	"github.com/liveshop-platform/module-platform/pkg/gfinit"
)

type signingConfig struct {
	KeyID      string `yaml:"key_id"`
	PrivateKey string `yaml:"private_key"`
	PublicKey  string `yaml:"public_key"`
	Issuer     string `yaml:"issuer"`
}

type trustedWorkloadConfig struct {
	KeyID       string   `yaml:"key_id"`
	PublicKey   string   `yaml:"public_key"`
	SPIFFEID    string   `yaml:"spiffe_id"`
	Subject     string   `yaml:"subject"`
	Permissions []string `yaml:"permissions"`
}

type configField struct {
	name  string
	value string
}

type runtime struct {
	cfg        *Config
	httpServer *platformserver.Server
	grpcServer *platformgrpc.Server
}

// Config Platform 进程配置，由启动参数指定的完整 YAML 文件提供。
type Config struct {
	commonconfig.Common `yaml:",inline"`
	Server              commonconfig.Server `yaml:"server"`
	Database            struct {
		URL                   string `yaml:"url"`
		MaxOpenConnections    int    `yaml:"max_open_connections"`
		MaxIdleConnections    int    `yaml:"max_idle_connections"`
		ConnectionMaxLifetime string `yaml:"connection_max_lifetime"`
		ConnectionMaxIdleTime string `yaml:"connection_max_idle_time"`
	} `yaml:"database"`
	ModuleSession  signingConfig `yaml:"module_session"`
	AccessIdentity signingConfig `yaml:"access_identity"`
	Workload       struct {
		Issuer  string                `yaml:"issuer"`
		Gateway trustedWorkloadConfig `yaml:"gateway"`
		Release trustedWorkloadConfig `yaml:"release"`
	} `yaml:"workload_identity"`
	HTTP struct {
		AllowedOrigins []string `yaml:"allowed_origins"`
		CookieSecure   bool     `yaml:"cookie_secure"`
	} `yaml:"http"`
	GRPC struct {
		TLS struct {
			CertificateFile string `yaml:"certificate_file"`
			PrivateKeyFile  string `yaml:"private_key_file"`
			ClientCAFile    string `yaml:"client_ca_file"`
		} `yaml:"tls"`
	} `yaml:"grpc"`
}

// bootstrap 加载并校验 Platform 配置，然后初始化进程依赖。
func bootstrap(ctx context.Context) (*runtime, error) {
	cfg, err := gfinit.Load[Config](ctx)
	if err != nil {
		return nil, fmt.Errorf("platform: load config: %w", err)
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	if err := platformregistry.Init(registryConfig(cfg)); err != nil {
		return nil, fmt.Errorf("platform: initialize dependencies: %w", err)
	}
	dependencies := platformregistry.Current()
	application := logic.New(dependencies)
	httpServer := platformserver.New(dependencies, application, platformserver.Config{
		CookieSecure:   cfg.HTTP.CookieSecure,
		AllowedOrigins: cfg.HTTP.AllowedOrigins,
	})
	httpServer.SetAddr(cfg.Server.HTTP)
	grpcServer, err := platformgrpc.New(platformgrpc.Config{
		Address: cfg.Server.GRPC,
		TLS: platformgrpc.TLSConfig{
			CertificateFile: cfg.GRPC.TLS.CertificateFile,
			PrivateKeyFile:  cfg.GRPC.TLS.PrivateKeyFile,
			ClientCAFile:    cfg.GRPC.TLS.ClientCAFile,
		},
		Workloads: []grpcauth.Workload{
			grpcWorkload(cfg.Workload.Gateway),
			grpcWorkload(cfg.Workload.Release),
		},
	}, application)
	if err != nil {
		_ = platformregistry.Close()
		return nil, err
	}
	return &runtime{cfg: cfg, httpServer: httpServer, grpcServer: grpcServer}, nil
}

func grpcWorkload(cfg trustedWorkloadConfig) grpcauth.Workload {
	return grpcauth.Workload{
		SPIFFEID:    cfg.SPIFFEID,
		Subject:     cfg.Subject,
		Permissions: cfg.Permissions,
	}
}

func registryConfig(cfg *Config) platformregistry.Config {
	maxLifetime, _ := time.ParseDuration(cfg.Database.ConnectionMaxLifetime)
	maxIdleTime, _ := time.ParseDuration(cfg.Database.ConnectionMaxIdleTime)
	return platformregistry.Config{
		DatabaseURL:           cfg.Database.URL,
		MaxOpenConnections:    cfg.Database.MaxOpenConnections,
		MaxIdleConnections:    cfg.Database.MaxIdleConnections,
		ConnectionMaxLifetime: maxLifetime,
		ConnectionMaxIdleTime: maxIdleTime,
		ModuleSession: platformregistry.SigningConfig{
			KeyID:      cfg.ModuleSession.KeyID,
			PrivateKey: cfg.ModuleSession.PrivateKey,
			PublicKey:  cfg.ModuleSession.PublicKey,
			Issuer:     cfg.ModuleSession.Issuer,
		},
		AccessIdentity: platformregistry.SigningConfig{
			KeyID:      cfg.AccessIdentity.KeyID,
			PrivateKey: cfg.AccessIdentity.PrivateKey,
			PublicKey:  cfg.AccessIdentity.PublicKey,
			Issuer:     cfg.AccessIdentity.Issuer,
		},
		WorkloadIssuer: cfg.Workload.Issuer,
		GatewayWorkload: platformregistry.TrustedWorkload{
			KeyID:       cfg.Workload.Gateway.KeyID,
			PublicKey:   cfg.Workload.Gateway.PublicKey,
			Subject:     cfg.Workload.Gateway.Subject,
			Permissions: cfg.Workload.Gateway.Permissions,
		},
		ReleaseWorkload: platformregistry.TrustedWorkload{
			KeyID:       cfg.Workload.Release.KeyID,
			PublicKey:   cfg.Workload.Release.PublicKey,
			Subject:     cfg.Workload.Release.Subject,
			Permissions: cfg.Workload.Release.Permissions,
		},
	}
}

func validateConfig(cfg *Config) error {
	validators := []func(*Config) error{
		validateCommonConfig,
		validateServerConfig,
		validateDatabaseConfig,
		validateModuleSessionConfig,
		validateAccessIdentityConfig,
		validateWorkloadIdentityConfig,
		validateHTTPConfig,
		validateGRPCConfig,
	}
	for _, validate := range validators {
		if err := validate(cfg); err != nil {
			return err
		}
	}
	return nil
}

func validateCommonConfig(cfg *Config) error {
	fields := []configField{
		{
			name:  "service",
			value: cfg.Service,
		},
		{
			name:  "log.level",
			value: cfg.Log.Level,
		},
		{
			name:  "log.format",
			value: cfg.Log.Format,
		},
	}
	if err := validateRequiredFields(fields); err != nil {
		return err
	}
	if cfg.Log.Format != "text" && cfg.Log.Format != "json" {
		return fmt.Errorf("platform: config log.format must be text or json")
	}
	return nil
}

func validateServerConfig(cfg *Config) error {
	return requireConfig("server.http", cfg.Server.HTTP)
}

func validateDatabaseConfig(cfg *Config) error {
	if err := requireConfig("database.url", cfg.Database.URL); err != nil {
		return err
	}
	if cfg.Database.MaxOpenConnections <= 0 || cfg.Database.MaxIdleConnections < 0 || cfg.Database.MaxIdleConnections > cfg.Database.MaxOpenConnections {
		return fmt.Errorf("platform: database connection pool limits are invalid")
	}
	for name, value := range map[string]string{
		"database.connection_max_lifetime":  cfg.Database.ConnectionMaxLifetime,
		"database.connection_max_idle_time": cfg.Database.ConnectionMaxIdleTime,
	} {
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return fmt.Errorf("platform: config %s must be a positive duration", name)
		}
	}
	return nil
}

func validateModuleSessionConfig(cfg *Config) error {
	fields := []configField{
		{
			name:  "module_session.key_id",
			value: cfg.ModuleSession.KeyID,
		},
		{
			name:  "module_session.private_key",
			value: cfg.ModuleSession.PrivateKey,
		},
		{
			name:  "module_session.public_key",
			value: cfg.ModuleSession.PublicKey,
		},
		{
			name:  "module_session.issuer",
			value: cfg.ModuleSession.Issuer,
		},
	}
	return validateRequiredFields(fields)
}

func validateAccessIdentityConfig(cfg *Config) error {
	fields := []configField{
		{
			name:  "access_identity.key_id",
			value: cfg.AccessIdentity.KeyID,
		},
		{
			name:  "access_identity.private_key",
			value: cfg.AccessIdentity.PrivateKey,
		},
		{
			name:  "access_identity.public_key",
			value: cfg.AccessIdentity.PublicKey,
		},
		{
			name:  "access_identity.issuer",
			value: cfg.AccessIdentity.Issuer,
		},
	}
	return validateRequiredFields(fields)
}

func validateWorkloadIdentityConfig(cfg *Config) error {
	fields := []configField{
		{
			name:  "workload_identity.issuer",
			value: cfg.Workload.Issuer,
		},
		{
			name:  "workload_identity.gateway.key_id",
			value: cfg.Workload.Gateway.KeyID,
		},
		{
			name:  "workload_identity.gateway.public_key",
			value: cfg.Workload.Gateway.PublicKey,
		},
		{
			name:  "workload_identity.gateway.spiffe_id",
			value: cfg.Workload.Gateway.SPIFFEID,
		},
		{
			name:  "workload_identity.release.key_id",
			value: cfg.Workload.Release.KeyID,
		},
		{
			name:  "workload_identity.release.public_key",
			value: cfg.Workload.Release.PublicKey,
		},
		{
			name:  "workload_identity.release.spiffe_id",
			value: cfg.Workload.Release.SPIFFEID,
		},
	}
	if err := validateRequiredFields(fields); err != nil {
		return err
	}
	if err := validateSPIFFEID("workload_identity.gateway.spiffe_id", cfg.Workload.Gateway.SPIFFEID); err != nil {
		return err
	}
	if err := validateSPIFFEID("workload_identity.release.spiffe_id", cfg.Workload.Release.SPIFFEID); err != nil {
		return err
	}
	for _, required := range []string{"platform.registry.routes.read", "platform.registry.capabilities.read"} {
		if !contains(cfg.Workload.Gateway.Permissions, required) {
			return fmt.Errorf("platform: config workload_identity.gateway.permissions must include %s", required)
		}
	}
	return nil
}

func validateHTTPConfig(cfg *Config) error {
	if len(cfg.HTTP.AllowedOrigins) == 0 {
		return fmt.Errorf("platform: config http.allowed_origins is required")
	}
	return nil
}

func validateGRPCConfig(cfg *Config) error {
	fields := []configField{
		{
			name:  "server.grpc",
			value: cfg.Server.GRPC,
		},
		{
			name:  "grpc.tls.certificate_file",
			value: cfg.GRPC.TLS.CertificateFile,
		},
		{
			name:  "grpc.tls.private_key_file",
			value: cfg.GRPC.TLS.PrivateKeyFile,
		},
		{
			name:  "grpc.tls.client_ca_file",
			value: cfg.GRPC.TLS.ClientCAFile,
		},
	}
	return validateRequiredFields(fields)
}

func validateRequiredFields(fields []configField) error {
	for _, field := range fields {
		if err := requireConfig(field.name, field.value); err != nil {
			return err
		}
	}
	return nil
}

func requireConfig(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("platform: config %s is required", name)
	}
	return nil
}

func validateSPIFFEID(name, value string) error {
	identity, err := url.Parse(value)
	if err != nil || identity.Scheme != "spiffe" || identity.Host == "" || identity.User != nil || identity.RawQuery != "" || identity.Fragment != "" {
		return fmt.Errorf("platform: config %s must be a SPIFFE ID", name)
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
