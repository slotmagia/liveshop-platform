package app

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{name: "valid config"},
		{
			name:    "missing service name",
			mutate:  func(cfg *Config) { cfg.Service = "" },
			wantErr: "platform: config service is required",
		},
		{
			name:    "missing log level",
			mutate:  func(cfg *Config) { cfg.Log.Level = "" },
			wantErr: "platform: config log.level is required",
		},
		{
			name:    "missing log format",
			mutate:  func(cfg *Config) { cfg.Log.Format = "" },
			wantErr: "platform: config log.format is required",
		},
		{
			name:    "invalid log format",
			mutate:  func(cfg *Config) { cfg.Log.Format = "console" },
			wantErr: "platform: config log.format must be text or json",
		},
		{
			name:    "missing HTTP address",
			mutate:  func(cfg *Config) { cfg.Server.HTTP = "" },
			wantErr: "platform: config server.http is required",
		},
		{
			name:    "missing database URL",
			mutate:  func(cfg *Config) { cfg.Database.URL = "" },
			wantErr: "platform: config database.url is required",
		},
		{
			name:    "invalid database pool",
			mutate:  func(cfg *Config) { cfg.Database.MaxIdleConnections = cfg.Database.MaxOpenConnections + 1 },
			wantErr: "platform: database connection pool limits are invalid",
		},
		{
			name:    "invalid database connection lifetime",
			mutate:  func(cfg *Config) { cfg.Database.ConnectionMaxLifetime = "never" },
			wantErr: "platform: config database.connection_max_lifetime must be a positive duration",
		},
		{
			name:    "missing module key ID",
			mutate:  func(cfg *Config) { cfg.ModuleSession.KeyID = "" },
			wantErr: "platform: config module_session.key_id is required",
		},
		{
			name:    "missing module private key",
			mutate:  func(cfg *Config) { cfg.ModuleSession.PrivateKey = "" },
			wantErr: "platform: config module_session.private_key is required",
		},
		{
			name:    "missing module public key",
			mutate:  func(cfg *Config) { cfg.ModuleSession.PublicKey = "" },
			wantErr: "platform: config module_session.public_key is required",
		},
		{
			name:    "missing module issuer",
			mutate:  func(cfg *Config) { cfg.ModuleSession.Issuer = "" },
			wantErr: "platform: config module_session.issuer is required",
		},
		{
			name:    "missing access key ID",
			mutate:  func(cfg *Config) { cfg.AccessIdentity.KeyID = "" },
			wantErr: "platform: config access_identity.key_id is required",
		},
		{
			name:    "missing access private key",
			mutate:  func(cfg *Config) { cfg.AccessIdentity.PrivateKey = "" },
			wantErr: "platform: config access_identity.private_key is required",
		},
		{
			name:    "missing access public key",
			mutate:  func(cfg *Config) { cfg.AccessIdentity.PublicKey = "" },
			wantErr: "platform: config access_identity.public_key is required",
		},
		{
			name:    "missing access issuer",
			mutate:  func(cfg *Config) { cfg.AccessIdentity.Issuer = "" },
			wantErr: "platform: config access_identity.issuer is required",
		},
		{
			name:    "missing workload issuer",
			mutate:  func(cfg *Config) { cfg.Workload.Issuer = "" },
			wantErr: "platform: config workload_identity.issuer is required",
		},
		{
			name:    "missing gateway workload key ID",
			mutate:  func(cfg *Config) { cfg.Workload.Gateway.KeyID = "" },
			wantErr: "platform: config workload_identity.gateway.key_id is required",
		},
		{
			name:    "missing gateway workload public key",
			mutate:  func(cfg *Config) { cfg.Workload.Gateway.PublicKey = "" },
			wantErr: "platform: config workload_identity.gateway.public_key is required",
		},
		{
			name:    "missing gateway workload SPIFFE ID",
			mutate:  func(cfg *Config) { cfg.Workload.Gateway.SPIFFEID = "" },
			wantErr: "platform: config workload_identity.gateway.spiffe_id is required",
		},
		{
			name:    "invalid gateway workload SPIFFE ID",
			mutate:  func(cfg *Config) { cfg.Workload.Gateway.SPIFFEID = "https://gateway.example" },
			wantErr: "platform: config workload_identity.gateway.spiffe_id must be a SPIFFE ID",
		},
		{
			name:    "missing release workload key ID",
			mutate:  func(cfg *Config) { cfg.Workload.Release.KeyID = "" },
			wantErr: "platform: config workload_identity.release.key_id is required",
		},
		{
			name:    "missing release workload public key",
			mutate:  func(cfg *Config) { cfg.Workload.Release.PublicKey = "" },
			wantErr: "platform: config workload_identity.release.public_key is required",
		},
		{
			name:    "missing release workload SPIFFE ID",
			mutate:  func(cfg *Config) { cfg.Workload.Release.SPIFFEID = "" },
			wantErr: "platform: config workload_identity.release.spiffe_id is required",
		},
		{
			name: "missing gRPC route permission",
			mutate: func(cfg *Config) {
				cfg.Workload.Gateway.Permissions = []string{"platform.registry.capabilities.read"}
			},
			wantErr: "platform: config workload_identity.gateway.permissions must include platform.registry.routes.read",
		},
		{
			name:    "missing allowed origins",
			mutate:  func(cfg *Config) { cfg.HTTP.AllowedOrigins = nil },
			wantErr: "platform: config http.allowed_origins is required",
		},
		{
			name:    "missing gRPC address",
			mutate:  func(cfg *Config) { cfg.Server.GRPC = "" },
			wantErr: "platform: config server.grpc is required",
		},
		{
			name:    "missing gRPC certificate",
			mutate:  func(cfg *Config) { cfg.GRPC.TLS.CertificateFile = "" },
			wantErr: "platform: config grpc.tls.certificate_file is required",
		},
		{
			name:    "missing gRPC private key",
			mutate:  func(cfg *Config) { cfg.GRPC.TLS.PrivateKeyFile = "" },
			wantErr: "platform: config grpc.tls.private_key_file is required",
		},
		{
			name:    "missing gRPC client CA",
			mutate:  func(cfg *Config) { cfg.GRPC.TLS.ClientCAFile = "" },
			wantErr: "platform: config grpc.tls.client_ca_file is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			if test.mutate != nil {
				test.mutate(cfg)
			}
			err := validateConfig(cfg)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("valid Platform config: %v", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantErr {
				t.Fatalf("validateConfig() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func validConfig() *Config {
	cfg := &Config{}
	cfg.Service = "platform"
	cfg.Log.Level = "info"
	cfg.Log.Format = "text"
	cfg.Server.HTTP = ":8082"
	cfg.Server.GRPC = ":9082"
	cfg.Database.URL = "postgres://configured"
	cfg.Database.MaxOpenConnections = 40
	cfg.Database.MaxIdleConnections = 10
	cfg.Database.ConnectionMaxLifetime = "30m"
	cfg.Database.ConnectionMaxIdleTime = "5m"
	cfg.ModuleSession = signingConfig{
		KeyID:      "module-key",
		PrivateKey: "module-private",
		PublicKey:  "module-public",
		Issuer:     "module-issuer",
	}
	cfg.AccessIdentity = signingConfig{
		KeyID:      "access-key",
		PrivateKey: "access-private",
		PublicKey:  "access-public",
		Issuer:     "access-issuer",
	}
	cfg.Workload.Issuer = "workload-issuer"
	cfg.Workload.Gateway = trustedWorkloadConfig{
		KeyID:       "gateway-key",
		PublicKey:   "gateway-public",
		SPIFFEID:    "spiffe://configured/gateway",
		Subject:     "liveshop-gateway",
		Permissions: []string{"registry.routes.read", "platform.registry.routes.read", "platform.registry.capabilities.read"},
	}
	cfg.Workload.Release = trustedWorkloadConfig{
		KeyID:       "release-key",
		PublicKey:   "release-public",
		SPIFFEID:    "spiffe://configured/release",
		Subject:     "module-release-ci",
		Permissions: []string{"registry.release.write"},
	}
	cfg.HTTP.AllowedOrigins = []string{"https://configured.example"}
	cfg.HTTP.CookieSecure = true
	cfg.GRPC.TLS.CertificateFile = "server.crt"
	cfg.GRPC.TLS.PrivateKeyFile = "server.key"
	cfg.GRPC.TLS.ClientCAFile = "ca.crt"
	return cfg
}
