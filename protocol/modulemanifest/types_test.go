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
			Backend: Backend{Service: "example", Origin: "http://example:18090", HTTPRoutes: []HTTPRoute{{
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

func TestPublicHTTPOperationDoesNotInventPermission(t *testing.T) {
	manifest := validManifest()
	manifest.Spec.Backend.HTTPRoutes = append(manifest.Spec.Backend.HTTPRoutes, HTTPRoute{
		Surface: "shop", Prefix: "/public/example", Operations: []HTTPOperation{{
			ID: "example.public.asset", Method: "GET", Path: "/public/example/{id}", Summary: "Read asset",
			Description: "Reads immutable public bytes.", Authentication: "public", Idempotency: "safe",
			RequestFields: []CapabilityField{{Name: "id", Location: "path", Type: "integer", Required: true, Description: "Asset id"}},
			Responses:     []CapabilityResponse{{Status: 200, Description: "Asset bytes"}},
		}},
	})
	if err := manifest.Validate(); err != nil {
		t.Fatalf("public operation should not require an authorization permission: %v", err)
	}
}

func TestGuestContributionAndRouteDoNotInventRolePermissions(t *testing.T) {
	manifest := validManifest()
	manifest.Spec.Backend.HTTPRoutes[0].Surface = "shop"
	manifest.Spec.Backend.HTTPRoutes[0].Prefix = "/shop/example"
	operation := &manifest.Spec.Backend.HTTPRoutes[0].Operations[0]
	operation.ID = "example.shop.page.get"
	operation.Path = "/shop/example"
	operation.Authentication = "guest-session"
	operation.RequiredPermissions = nil
	contribution := &manifest.Spec.Contributions[0]
	contribution.Surface = "shop"
	contribution.RequiredPermissions = nil
	contribution.AllowedRoutes[0].Prefix = "/shop/example"
	contribution.AllowedRoutes[0].RequiredPermissions = nil
	if err := manifest.Validate(); err != nil {
		t.Fatalf("guest contribution should be discoverable without a role grant: %v", err)
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

func TestManifestAllowsStaticPageWithoutBackendRoutes(t *testing.T) {
	m := validManifest()
	m.Spec.Contributions[0].AllowedRoutes = []AllowedRoute{}
	if err := m.Validate(); err != nil {
		t.Fatalf("static page should not need a fabricated backend route: %v", err)
	}
}

func TestManifestAllowsSharedDeleteOperationID(t *testing.T) {
	m := validManifest()
	m.Spec.Backend.HTTPRoutes[0].Prefix = "/shop/example/cart"
	base := m.Spec.Backend.HTTPRoutes[0].Operations[0]
	base.Authentication = "guest-session"
	base.RequiredPermissions = nil
	item := base
	item.ID = "example.shop.cart.items.delete"
	item.Method = "DELETE"
	item.Path = "/shop/example/cart/items/{skuId}"
	item.Summary = "Remove cart item"
	item.Description = "Removes one SKU from the current cart."
	item.RequestFields = []CapabilityField{{Name: "skuId", Type: "integer", Location: "path", Required: true, Description: "SKU ID"}}
	collection := base
	collection.ID = "example.shop.cart.items.delete"
	collection.Method = "DELETE"
	collection.Path = "/shop/example/cart/items"
	collection.Summary = "Remove selected cart items"
	collection.Description = "Atomically removes selected SKU rows from the current cart."
	m.Spec.Backend.HTTPRoutes[0].Operations = []HTTPOperation{item, collection}
	m.Spec.Contributions[0].AllowedRoutes[0].Prefix = "/shop/example/cart"
	if err := m.Validate(); err != nil {
		t.Fatalf("collection and member delete may share an operation id: %v", err)
	}
}

func TestManifestRejectsDuplicateHTTPRoute(t *testing.T) {
	m := validManifest()
	dup := m.Spec.Backend.HTTPRoutes[0].Operations[0]
	dup.ID = "example.page.list"
	m.Spec.Backend.HTTPRoutes[0].Operations = append(m.Spec.Backend.HTTPRoutes[0].Operations, dup)
	if err := m.Validate(); err == nil {
		t.Fatal("expected duplicate method+path to be rejected")
	}
}

func TestManifestRejectsUndocumentedHTTPRoute(t *testing.T) {
	m := validManifest()
	m.Spec.Backend.HTTPRoutes[0].Operations = nil
	if err := m.Validate(); err == nil {
		t.Fatal("expected undocumented HTTP route to be rejected")
	}
}

func TestManifestValidatesPageNavigation(t *testing.T) {
	m := validManifest()
	m.Spec.Contributions[0].Navigation = &Navigation{GroupID: "trade", GroupTitle: "Trade 管理", GroupSort: 100}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}

	m.Spec.Contributions[0].Kind = "widget"
	m.Spec.Contributions[0].Outlet = "admin.dashboard.widgets"
	if err := m.Validate(); err == nil {
		t.Fatal("expected non-page navigation metadata to be rejected")
	}
}

func TestManifestValidatesMachineReadableGRPCAndFrontendCapabilities(t *testing.T) {
	m := validManifest()
	m.Spec.Backend.GRPC = &GRPC{
		Service: "liveshop.example.v1.ExampleService", ContractVersion: "1.0.0", Endpoint: "example:19090", TransportSecurity: "mtls-spiffe",
		Methods: []GRPCMethod{{Name: "GetExample", FullMethod: "/liveshop.example.v1.ExampleService/GetExample", Summary: "Get example", Description: "Reads one example.", Invocation: "unary", Idempotency: "safe", RecommendedDeadlineMS: 1000, RequiredPermissions: []string{"example.page.read"}, RequestFields: []CapabilityField{{Name: "id", Type: "integer", Required: true, Description: "Example ID"}}, ResponseFields: []CapabilityField{{Name: "name", Type: "string", Description: "Example name"}}}},
	}
	m.Spec.Contributions[0].Frontend.Actions = []FrontendAction{{ID: "refresh", Label: "Refresh", Description: "Reload example data.", Invocation: "http", Target: "example.page.get", RequiredPermissions: []string{"example.page.read"}}}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}
