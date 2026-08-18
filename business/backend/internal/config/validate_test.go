package config

import "testing"

func TestIdentityWorkloadRequiresActiveCapabilityRead(t *testing.T) {
	cfg := &Config{}
	cfg.WorkloadIdentity.GRPC.Identity.Permissions = []string{"platform.registry.routes.read"}
	if err := validateWorkloadIdentity(cfg); err == nil {
		t.Fatal("identity workload without active capability read was accepted")
	}
}

func TestIdentityGRPCWorkloadRejectsGatewayPermission(t *testing.T) {
	cfg := completeWorkloadConfig()
	cfg.WorkloadIdentity.GRPC.Identity.Permissions = []string{"platform.registry.active-capabilities.read", "platform.notify-event.dispatch", "platform.registry.routes.read"}
	if err := validateWorkloadIdentity(cfg); err == nil {
		t.Fatal("Identity gRPC workload with Gateway route permission was accepted")
	}
}

func completeWorkloadConfig() *Config {
	cfg := &Config{}
	cfg.WorkloadIdentity.Issuer = "issuer"
	cfg.WorkloadIdentity.HTTP.Gateway = BearerWorkload{KeyID: "gateway", PublicKey: "public", Subject: "gateway", Permissions: []string{"registry.routes.read"}}
	cfg.WorkloadIdentity.HTTP.Release = BearerWorkload{KeyID: "release", PublicKey: "public", Subject: "release", Permissions: []string{"registry.release.write", "registry.activation.write"}}
	cfg.WorkloadIdentity.HTTP.Identity = BearerWorkload{KeyID: "identity", PublicKey: "public", Subject: "identity", Permissions: []string{"platform.notify-event.dispatch"}}
	cfg.WorkloadIdentity.GRPC.Gateway = MTLSWorkload{SPIFFEID: "spiffe://liveshop.local/gateway", Subject: "gateway", Permissions: []string{"platform.registry.routes.read"}}
	cfg.WorkloadIdentity.GRPC.Identity = MTLSWorkload{SPIFFEID: "spiffe://liveshop.local/identity", Subject: "identity", Permissions: []string{"platform.registry.active-capabilities.read", "platform.notify-event.dispatch"}}
	return cfg
}
