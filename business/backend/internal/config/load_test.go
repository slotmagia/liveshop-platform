package config

import (
	"testing"

	"github.com/liveshop-platform/module-platform/pkg/gfinit"
)

// The loader resolves a YAML section to a struct field by field name, not by
// the yaml tag on it. A field whose name does not spell out the key therefore
// produces a zero-valued section instead of an error, and the process only
// fails later with a misleading "config X is required". Nothing catches that
// except loading the files this repository actually ships: a unit test that
// builds a Config in memory never exercises the mapping.
func TestShippedConfigsMapEverySection(t *testing.T) {
	for _, test := range []struct {
		name     string
		file     string
		complete bool
	}{
		{name: "compose", file: "../../configs/platform.compose.yaml", complete: true},
		{name: "template", file: "../../configs/platform.yaml"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := gfinit.MustInit(gfinit.Options{Service: "platform", ConfigFile: test.file})
			cfg, err := gfinit.Load[Config](ctx)
			if err != nil {
				t.Fatalf("load %s: %v", test.file, err)
			}
			// One assertion per section that carries a value in both files. An
			// unmapped section is indistinguishable from an absent one at the
			// call site, so each has to be named explicitly.
			if cfg.Server.HTTP == "" {
				t.Error("server did not map")
			}
			if cfg.ModuleCapability.Issuer == "" {
				t.Error("module_capability did not map")
			}
			if cfg.WorkloadIdentity.Issuer == "" {
				t.Error("workload_identity did not map")
			}
			if cfg.WorkloadIdentity.HTTP.Identity.Subject == "" {
				t.Error("workload_identity.http.identity did not map")
			}
			if cfg.WorkloadIdentity.GRPC.Identity.SPIFFEID == "" || len(cfg.WorkloadIdentity.GRPC.Identity.Permissions) == 0 {
				t.Error("workload_identity.grpc.identity did not map")
			}
			if cfg.Registry.Workload.Issuer == "" {
				t.Error("registry did not map")
			}
			// The template deliberately leaves deployment-supplied values
			// blank, so only the file the containers boot with is checked for
			// them. Otherwise the container fails at startup rather than here.
			if !test.complete {
				return
			}
			if cfg.InternalGrant.Token == "" {
				t.Error("internal_grant did not map")
			}
			if cfg.Edge.IdentityOrigin == "" || cfg.Edge.Upstreams.Shop == "" || cfg.Edge.Upstreams.Gateway == "" {
				t.Error("edge did not map")
			}
			if cfg.Database.URL == "" {
				t.Error("database did not map")
			}
			if cfg.CredentialEncryption.KeyID == "" || cfg.CredentialEncryption.MasterKey == "" {
				t.Error("credential_encryption did not map")
			}
			if len(cfg.HTTP.AllowedOrigins) == 0 {
				t.Error("http did not map")
			}
			if cfg.GRPC.TLS.CertificateFile == "" {
				t.Error("grpc.tls did not map")
			}
			if cfg.Registry.OriginURL == "" || cfg.Registry.Workload.PrivateKey == "" {
				t.Error("registry origin did not map")
			}
			if err := Validate(cfg); err != nil {
				t.Fatalf("validate %s: %v", test.file, err)
			}
		})
	}
}
