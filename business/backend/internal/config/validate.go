package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type field struct {
	name  string
	value string
}

func Validate(cfg *Config) error {
	validators := []func(*Config) error{
		validateCommon,
		validateServer,
		validateDatabase,
		validateCredentialEncryption,
		validateModuleCapability,
		validateWorkloadIdentity,
		validateRegistry,
		validateHTTP,
		validateGRPC,
		validateEdge,
	}
	for _, validate := range validators {
		if err := validate(cfg); err != nil {
			return err
		}
	}
	return nil
}

func validateCredentialEncryption(cfg *Config) error {
	if err := requireFields([]field{
		{name: "credential_encryption.key_id", value: cfg.CredentialEncryption.KeyID},
		{name: "credential_encryption.master_key", value: cfg.CredentialEncryption.MasterKey},
	}); err != nil {
		return err
	}
	key, err := base64.StdEncoding.DecodeString(cfg.CredentialEncryption.MasterKey)
	if err != nil || len(key) != 32 {
		return fmt.Errorf("platform: config credential_encryption.master_key must be a base64 encoded 32-byte key")
	}
	return nil
}

func validateCommon(cfg *Config) error {
	fields := []field{
		{name: "service", value: cfg.Service},
		{name: "log.level", value: cfg.Log.Level},
		{name: "log.format", value: cfg.Log.Format},
	}
	if err := requireFields(fields); err != nil {
		return err
	}
	if cfg.Log.Format != "text" && cfg.Log.Format != "json" {
		return fmt.Errorf("platform: config log.format must be text or json")
	}
	return nil
}

func validateServer(cfg *Config) error {
	return require("server.http", cfg.Server.HTTP)
}

// validateDatabase also stores the parsed pool durations, so the format is
// owned here and never re-parsed by a caller.
func validateDatabase(cfg *Config) error {
	if err := require("database.url", cfg.Database.URL); err != nil {
		return err
	}
	if cfg.Database.MaxOpenConnections <= 0 || cfg.Database.MaxIdleConnections < 0 || cfg.Database.MaxIdleConnections > cfg.Database.MaxOpenConnections {
		return fmt.Errorf("platform: database connection pool limits are invalid")
	}
	maxLifetime, err := positiveDuration("database.connection_max_lifetime", cfg.Database.ConnectionMaxLifetime)
	if err != nil {
		return err
	}
	maxIdleTime, err := positiveDuration("database.connection_max_idle_time", cfg.Database.ConnectionMaxIdleTime)
	if err != nil {
		return err
	}
	cfg.connectionMaxLifetime = maxLifetime
	cfg.connectionMaxIdleTime = maxIdleTime
	return nil
}

func positiveDuration(name, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("platform: config %s must be a positive duration", name)
	}
	return duration, nil
}

func validateModuleCapability(cfg *Config) error {
	return requireFields([]field{
		{name: "module_capability.key_id", value: cfg.ModuleCapability.KeyID},
		{name: "module_capability.public_key", value: cfg.ModuleCapability.PublicKey},
		{name: "module_capability.issuer", value: cfg.ModuleCapability.Issuer},
	})
}

func validateWorkloadIdentity(cfg *Config) error {
	fields := []field{
		{name: "workload_identity.issuer", value: cfg.WorkloadIdentity.Issuer},
		{name: "workload_identity.http.identity.key_id", value: cfg.WorkloadIdentity.HTTP.Identity.KeyID},
		{name: "workload_identity.http.identity.public_key", value: cfg.WorkloadIdentity.HTTP.Identity.PublicKey},
		{name: "workload_identity.http.identity.subject", value: cfg.WorkloadIdentity.HTTP.Identity.Subject},
		{name: "workload_identity.grpc.identity.spiffe_id", value: cfg.WorkloadIdentity.GRPC.Identity.SPIFFEID},
		{name: "workload_identity.grpc.identity.subject", value: cfg.WorkloadIdentity.GRPC.Identity.Subject},
	}
	if err := requireFields(fields); err != nil {
		return err
	}
	if err := requireSPIFFEID("workload_identity.grpc.identity.spiffe_id", cfg.WorkloadIdentity.GRPC.Identity.SPIFFEID); err != nil {
		return err
	}
	if !exactPermissions(cfg.WorkloadIdentity.HTTP.Identity.Permissions, "platform.notify-event.dispatch") {
		return fmt.Errorf("platform: config workload_identity.http.identity.permissions must contain only platform.notify-event.dispatch")
	}
	if !exactPermissions(cfg.WorkloadIdentity.GRPC.Identity.Permissions, "platform.notify-event.dispatch") {
		return fmt.Errorf("platform: config workload_identity.grpc.identity.permissions must contain only platform.notify-event.dispatch")
	}
	return nil
}

func validateRegistry(cfg *Config) error {
	if err := requireFields([]field{
		{name: "registry.origin_url", value: cfg.Registry.OriginURL},
		{name: "registry.workload.key_id", value: cfg.Registry.Workload.KeyID},
		{name: "registry.workload.private_key", value: cfg.Registry.Workload.PrivateKey},
		{name: "registry.workload.issuer", value: cfg.Registry.Workload.Issuer},
		{name: "registry.workload.subject", value: cfg.Registry.Workload.Subject},
	}); err != nil {
		return err
	}
	parsed, err := url.Parse(cfg.Registry.OriginURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("platform: config registry.origin_url must be an http(s) origin")
	}
	return nil
}

func validateHTTP(cfg *Config) error {
	if len(cfg.HTTP.AllowedOrigins) == 0 {
		return fmt.Errorf("platform: config http.allowed_origins is required")
	}
	return nil
}

func validateGRPC(cfg *Config) error {
	return requireFields([]field{
		{name: "server.grpc", value: cfg.Server.GRPC},
		{name: "grpc.tls.certificate_file", value: cfg.GRPC.TLS.CertificateFile},
		{name: "grpc.tls.private_key_file", value: cfg.GRPC.TLS.PrivateKeyFile},
		{name: "grpc.tls.client_ca_file", value: cfg.GRPC.TLS.ClientCAFile},
	})
}

func validateEdge(cfg *Config) error {
	originChecks := []field{
		{name: "edge.identity_origin", value: cfg.Edge.IdentityOrigin},
		{name: "edge.ask_origin", value: cfg.Edge.AskOrigin},
	}
	for _, item := range originChecks {
		if strings.TrimSpace(item.value) == "" {
			if cfg.Edge.Enabled {
				return require(item.name, item.value)
			}
			continue
		}
		parsed, err := url.Parse(item.value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("platform: config %s must be an absolute URL", item.name)
		}
	}
	upstreams := []field{
		{name: "edge.upstreams.shop", value: cfg.Edge.Upstreams.Shop},
		{name: "edge.upstreams.live", value: cfg.Edge.Upstreams.Live},
		{name: "edge.upstreams.merch", value: cfg.Edge.Upstreams.Merch},
		{name: "edge.upstreams.admin", value: cfg.Edge.Upstreams.Admin},
		{name: "edge.upstreams.rts", value: cfg.Edge.Upstreams.RTS},
		{name: "edge.upstreams.gateway", value: cfg.Edge.Upstreams.Gateway},
	}
	anyUpstream := false
	for _, item := range upstreams {
		if strings.TrimSpace(item.value) != "" {
			anyUpstream = true
			break
		}
	}
	if cfg.Edge.Enabled || anyUpstream {
		if err := requireFields(upstreams); err != nil {
			return err
		}
	}
	if !cfg.Edge.Enabled {
		return nil
	}
	return requireFields([]field{
		{name: "internal_grant.token", value: cfg.InternalGrant.Token},
		{name: "edge.caddy_admin", value: cfg.Edge.CaddyAdmin},
		{name: "edge.caddyfile", value: cfg.Edge.Caddyfile},
		{name: "edge.acme_email", value: cfg.Edge.ACMEEmail},
	})
}

func requireFields(fields []field) error {
	for _, item := range fields {
		if err := require(item.name, item.value); err != nil {
			return err
		}
	}
	return nil
}

func require(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("platform: config %s is required", name)
	}
	return nil
}

func requireSPIFFEID(name, value string) error {
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

func exactPermissions(values []string, expected ...string) bool {
	if len(values) != len(expected) {
		return false
	}
	for _, permission := range expected {
		if !contains(values, permission) {
			return false
		}
	}
	return true
}
