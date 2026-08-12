package modulemanifest

import (
	"encoding/json"
	"testing"
)

func validManifest() Manifest {
	permission := "example.page.read"
	return Manifest{
		APIVersion: APIVersion,
		Kind:       "ModuleRelease",
		Metadata:   Metadata{ID: "example", Name: "Example", Version: "1.0.0"},
		Spec: Spec{
			Backend: Backend{Service: "example", Origin: "http://example:8090", HTTPRoutes: []HTTPRoute{{
				Surface: "admin", Prefix: "/admin/example", Operations: []HTTPOperation{{
					ID: "example.page.get", Method: "GET", Path: "/admin/example", Summary: "Read example", Description: "Returns the example page data.", Authentication: "module-session", Idempotency: "safe", RequiredPermissions: []string{permission},
					Responses: []CapabilityResponse{{Status: 200, Description: "Example data", Fields: []CapabilityField{{Name: "name", Type: "string", Description: "Example name"}}}},
				}},
			}}},
			Permissions: []PermissionDefinition{{Code: permission, Name: "Read page", Resource: "example.page", Action: "read"}},
			Contributions: []Contribution{{
				ID: "example.page", Surface: "admin", Kind: "page", Route: "/example", Title: "Example", Description: "Example management page.", RequiredPermissions: []string{permission},
				AllowedRoutes: []AllowedRoute{{Methods: []string{"GET"}, Prefix: "/admin/example", RequiredPermissions: []string{permission}}},
				Artifact:      Artifact{Type: "iframe", Name: "example-admin", Version: "1.0.0", Entry: "https://example.invalid", Integrity: "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
				Frontend:      FrontendContract{Component: "ExamplePage"},
			}},
		},
	}
}

func TestDecodeRejectsTrailingJSONDocument(t *testing.T) {
	encoded, err := json.Marshal(validManifest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(append(encoded, []byte(` {}`)...)); err == nil {
		t.Fatal("expected a trailing JSON document to be rejected")
	}
}

func TestManifestRejectsArtifactVersionMismatch(t *testing.T) {
	m := validManifest()
	m.Spec.Contributions[0].Artifact.Version = "2.0.0"
	if err := m.Validate(); err == nil {
		t.Fatal("expected version mismatch")
	}
}

func TestManifestRejectsAllowedRouteOutsideSurface(t *testing.T) {
	m := validManifest()
	m.Spec.Contributions[0].AllowedRoutes[0].Prefix = "/shop/other"
	if err := m.Validate(); err == nil {
		t.Fatal("expected out-of-surface allowed route to be rejected")
	}
}

func TestManifestRejectsUndocumentedHTTPRoute(t *testing.T) {
	m := validManifest()
	m.Spec.Backend.HTTPRoutes[0].Operations = nil
	if err := m.Validate(); err == nil {
		t.Fatal("expected undocumented HTTP route to be rejected")
	}
}

func TestManifestValidatesMachineReadableGRPCAndFrontendCapabilities(t *testing.T) {
	m := validManifest()
	m.Spec.Backend.GRPC = &GRPC{
		Service: "liveshop.example.v1.ExampleService", ContractVersion: "1.0.0", Endpoint: "example:9090", TransportSecurity: "mtls-spiffe",
		Methods: []GRPCMethod{{Name: "GetExample", FullMethod: "/liveshop.example.v1.ExampleService/GetExample", Summary: "Get example", Description: "Reads one example.", Invocation: "unary", Idempotency: "safe", RecommendedDeadlineMS: 1000, RequiredPermissions: []string{"example.page.read"}, RequestFields: []CapabilityField{{Name: "id", Type: "integer", Required: true, Description: "Example ID"}}, ResponseFields: []CapabilityField{{Name: "name", Type: "string", Description: "Example name"}}}},
	}
	m.Spec.Contributions[0].Frontend.Actions = []FrontendAction{{ID: "refresh", Label: "Refresh", Description: "Reload example data.", Invocation: "http", Target: "example.page.get", RequiredPermissions: []string{"example.page.read"}}}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}
