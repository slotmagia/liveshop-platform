// Package config owns the Platform process configuration schema and its
// validation. It performs no dependency construction.
package config

import (
	"context"
	"fmt"
	"time"

	commonconfig "github.com/liveshop-platform/module-platform/pkg/config"
	"github.com/liveshop-platform/module-platform/pkg/gfinit"
)

// Verification is a public trust anchor. Platform verifies Identity-issued
// Module Capabilities; only liveshop-identity owns the private key.
type Verification struct {
	KeyID     string `yaml:"key_id"`
	PublicKey string `yaml:"public_key"`
	Issuer    string `yaml:"issuer"`
}

type BearerWorkload struct {
	KeyID       string   `yaml:"key_id"`
	PublicKey   string   `yaml:"public_key"`
	Subject     string   `yaml:"subject"`
	Permissions []string `yaml:"permissions"`
}

type MTLSWorkload struct {
	SPIFFEID    string   `yaml:"spiffe_id"`
	Subject     string   `yaml:"subject"`
	Permissions []string `yaml:"permissions"`
}

// Config is the Platform process configuration provided by the complete YAML
// file named on the command line.
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
	CredentialEncryption struct {
		KeyID     string `yaml:"key_id"`
		MasterKey string `yaml:"master_key"`
	} `yaml:"credential_encryption"`
	ModuleCapability Verification `yaml:"module_capability"`
	// The field name has to spell out the YAML key. The loader maps sections by
	// field name, not by the yaml tag, so a shorter name here silently yields a
	// zero-valued section instead of a parse error.
	WorkloadIdentity struct {
		Issuer string `yaml:"issuer"`
		HTTP   struct {
			Gateway BearerWorkload `yaml:"gateway"`
			Release BearerWorkload `yaml:"release"`
		} `yaml:"http"`
		GRPC struct {
			Gateway  MTLSWorkload `yaml:"gateway"`
			Identity MTLSWorkload `yaml:"identity"`
		} `yaml:"grpc"`
	} `yaml:"workload_identity"`
	InternalGrant struct {
		Token string `yaml:"token"`
	} `yaml:"internal_grant"`
	HTTP struct {
		AllowedOrigins []string `yaml:"allowed_origins"`
	} `yaml:"http"`
	GRPC struct {
		TLS struct {
			CertificateFile string `yaml:"certificate_file"`
			PrivateKeyFile  string `yaml:"private_key_file"`
			ClientCAFile    string `yaml:"client_ca_file"`
		} `yaml:"tls"`
	} `yaml:"grpc"`

	// Durations are parsed once by Validate so no caller repeats the format.
	connectionMaxLifetime time.Duration
	connectionMaxIdleTime time.Duration
}

// ConnectionMaxLifetime is the pool lifetime proven valid by Validate.
func (c *Config) ConnectionMaxLifetime() time.Duration { return c.connectionMaxLifetime }

// ConnectionMaxIdleTime is the pool idle timeout proven valid by Validate.
func (c *Config) ConnectionMaxIdleTime() time.Duration { return c.connectionMaxIdleTime }

// Load reads and validates the configuration file selected at startup.
func Load(ctx context.Context) (*Config, error) {
	cfg, err := gfinit.Load[Config](ctx)
	if err != nil {
		return nil, fmt.Errorf("platform: load config: %w", err)
	}
	if err := Validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
