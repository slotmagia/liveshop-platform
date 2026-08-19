package config

import "testing"

func TestIdentityGRPCWorkloadRejectsRegistryPermission(t *testing.T) {
	cfg := completeWorkloadConfig()
	cfg.WorkloadIdentity.GRPC.Identity.Permissions = []string{"platform.notify-event.dispatch", "platform.registry.active-capabilities.read"}
	if err := validateWorkloadIdentity(cfg); err == nil {
		t.Fatal("Identity gRPC workload with Registry permission was accepted")
	}
}

func completeWorkloadConfig() *Config {
	cfg := &Config{}
	cfg.WorkloadIdentity.Issuer = "issuer"
	cfg.WorkloadIdentity.HTTP.Identity = BearerWorkload{KeyID: "identity", PublicKey: "public", Subject: "identity", Permissions: []string{"platform.notify-event.dispatch"}}
	cfg.WorkloadIdentity.GRPC.Identity = MTLSWorkload{SPIFFEID: "spiffe://liveshop.local/identity", Subject: "identity", Permissions: []string{"platform.notify-event.dispatch"}}
	return cfg
}
