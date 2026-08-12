package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/logic"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	moduleregistry "github.com/liveshop-platform/module-platform/internal/platform/registry/module"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
)

const identityPrivateKey = "TM0Imyj_ltqdtsNG7BFOD1uKMZ81q6Yk2oz27U-4pvs"
const identityPublicKey = "PUAXw-hDiVqStwqnTRt-vJyYLM8uxJaMwM1V8Sr0Zgw"
const sessionPrivateKey = "nWGxne_9WmC6hEr0kuwsxERJxWl7MmkZcDusAxyuf2A"
const sessionPublicKey = "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo"

func catalogTestManifest() modulemanifest.Manifest {
	permission := "catalog.product.read"
	return modulemanifest.Manifest{
		APIVersion: modulemanifest.APIVersion,
		Kind:       "ModuleRelease",
		Metadata:   modulemanifest.Metadata{ID: "catalog", Name: "Catalog", Version: "1.0.0"},
		Spec: modulemanifest.Spec{
			Backend: modulemanifest.Backend{Service: "catalog", Origin: "http://catalog:8090", HTTPRoutes: []modulemanifest.HTTPRoute{{
				Surface: "shop", Prefix: "/shop/catalog", Operations: []modulemanifest.HTTPOperation{{
					ID: "catalog.product.list", Method: "GET", Path: "/shop/catalog/products", Summary: "List products", Description: "Lists storefront products.", Authentication: "module-session", Idempotency: "safe", RequiredPermissions: []string{permission},
					Responses: []modulemanifest.CapabilityResponse{{Status: 200, Description: "Product list"}},
				}},
			}}},
			Permissions: []modulemanifest.PermissionDefinition{{Code: permission, Name: "View products", Resource: "catalog.product", Action: "read"}},
			Contributions: []modulemanifest.Contribution{{
				ID: "catalog.shop", Surface: "shop", Kind: "page", Route: "/catalog", Title: "Catalog", Description: "Catalog storefront page.", RequiredPermissions: []string{permission},
				AllowedRoutes: []modulemanifest.AllowedRoute{{Methods: []string{"GET"}, Prefix: "/shop/catalog", RequiredPermissions: []string{permission}}},
				Artifact:      modulemanifest.Artifact{Type: "iframe", Name: "catalog", Version: "1.0.0", Entry: "https://catalog.example/index.html", Integrity: "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
				Frontend:      modulemanifest.FrontendContract{Component: "CatalogPage"},
			}},
		},
	}
}

func testRuntime(t *testing.T) (http.Handler, *accessidentity.Issuer, *modulesession.Verifier) {
	t.Helper()
	identityIssuer, _ := accessidentity.NewIssuer(identityPrivateKey, "identity-1", "test-iam")
	identityVerifier, _ := accessidentity.NewVerifier(map[string]string{"identity-1": identityPublicKey}, "test-iam", "liveshop-platform")
	sessionIssuer, _ := modulesession.NewIssuer(sessionPrivateKey, "session-1", "test-platform")
	sessionVerifier, _ := modulesession.NewVerifier(map[string]string{"session-1": sessionPublicKey}, "test-platform", "liveshop-module:catalog")
	platformSessionVerifier, _ := modulesession.NewVerifier(map[string]string{"session-1": sessionPublicKey}, "test-platform", "liveshop-module:platform")
	store := moduleregistry.NewStore()
	manifest := catalogTestManifest()
	if _, err := store.Register(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), "catalog", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	authorizations := iam.NewStore()
	authorizations.SeedPermission(iam.Permission{ModuleID: "catalog", Code: "catalog.product.read", Name: "View products", Resource: "catalog.product", Action: "read"})
	authorizations.SeedPermission(iam.Permission{ModuleID: "platform", Code: "platform.iam.manage", Name: "Manage IAM", Resource: "platform.iam", Action: "manage"})
	authorizations.SeedPermission(iam.Permission{ModuleID: "platform", Code: "platform.registry.manage", Name: "Manage registry", Resource: "platform.registry", Action: "manage"})
	role, err := authorizations.PutRole(context.Background(), iam.Tenant{AppID: 1001, MerchantID: 2001}, iam.Role{ID: 1, Name: "Reader", Status: iam.StatusActive, SuperAdmin: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := authorizations.SetSubjectAssignment(context.Background(), iam.Tenant{AppID: 1001, MerchantID: 2001}, "user-1", iam.SubjectAssignment{RoleIDs: []int64{role.ID}}); err != nil {
		t.Fatal(err)
	}
	dependencies := platformregistry.Dependencies{
		Modules:        store,
		IAM:            authorizations,
		Identities:     identityVerifier,
		ModuleIssuer:   sessionIssuer,
		ModuleVerifier: platformSessionVerifier,
	}
	server := New(dependencies, logic.New(dependencies), Config{AllowedOrigins: []string{"https://host.example"}})
	return startTestServer(t, server), identityIssuer, sessionVerifier
}

func startTestServer(t *testing.T, server *Server) http.Handler {
	t.Helper()
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Shutdown(); err != nil {
			t.Errorf("shutdown test server: %v", err)
		}
	})
	return server.Handler()
}

func TestGoFrameTransportContracts(t *testing.T) {
	dependencies := platformregistry.Dependencies{}
	handler := startTestServer(t, New(dependencies, logic.New(dependencies), Config{AllowedOrigins: []string{"https://host.example"}}))

	t.Run("health", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
		if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"ok"`)) {
			t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("cors rejects unknown origin", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		request.Header.Set("Origin", "https://unknown.example")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("CORS status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("malformed JSON is a client error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"realm":`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("malformed JSON status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("refresh requires cookie", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/refresh", nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("refresh status=%d body=%s", response.Code, response.Body.String())
		}
	})
}

func TestLivenessIsIndependentFromDatabaseReadiness(t *testing.T) {
	dependencies := platformregistry.Dependencies{Ready: func(context.Context) error { return errors.New("database unavailable") }}
	handler := startTestServer(t, New(dependencies, logic.New(dependencies), Config{}))

	for path, want := range map[string]int{"/livez": http.StatusOK, "/readyz": http.StatusServiceUnavailable, "/health": http.StatusServiceUnavailable} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != want {
			t.Errorf("%s status=%d want=%d body=%s", path, response.Code, want, response.Body.String())
		}
	}
}

func TestIAMAdministrationDrivesModuleSessionDataScope(t *testing.T) {
	handler, identityIssuer, sessionVerifier := testRuntime(t)
	adminSigner, _ := modulesession.NewIssuer(sessionPrivateKey, "session-1", "test-platform")
	adminToken, _ := adminSigner.Sign(modulesession.Claims{Subject: "user-1", Realm: accessidentity.RealmPlatform, ModuleID: "platform", ModuleVersion: "1.0.0", Surface: "admin", ContributionID: "platform.admin.iam", AppID: 1001, MerchantID: 2001, AuthorizationRevision: 1, Permissions: []string{"platform.iam.manage"}, DataScopes: []modulesession.DataScope{{Resource: "platform.iam", Type: modulesession.DataScopeAll}}, AllowedRoutes: []modulesession.RouteScope{{Methods: []string{"GET", "PUT"}, Prefix: "/admin/platform/iam"}}}, time.Minute)
	put := func(path, body string, want int) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(body))
		request.Header.Set("Authorization", "Bearer "+adminToken)
		request.Header.Set("X-Liveshop-Surface", "admin")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != want {
			t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
	put("/admin/platform/iam/departments/10", `{"expectedVersion":0,"name":"Sales","status":"ACTIVE"}`, http.StatusOK)
	put("/admin/platform/iam/roles/2", `{"expectedVersion":0,"name":"Department reader","status":"ACTIVE"}`, http.StatusOK)
	put("/admin/platform/iam/roles/2/policy", `{"expectedVersion":1,"permissions":["catalog.product.read"],"scopes":[{"resource":"catalog.product","type":"CUSTOM","departmentIds":[10]}]}`, http.StatusOK)
	put("/admin/platform/iam/subjects/operator-1/assignment", `{"roleIds":[2],"departments":[{"departmentId":10,"primary":true}]}`, http.StatusOK)
	put("/admin/platform/iam/roles/3", `{"expectedVersion":0,"name":"Illegal super","status":"ACTIVE","superAdmin":true}`, http.StatusForbidden)

	operatorToken, _ := identityIssuer.Sign(accessidentity.Claims{Subject: "operator-1", Realm: accessidentity.RealmMerchant, AppID: 1001, MerchantID: 2001}, time.Hour)
	body := bytes.NewBufferString(`{"moduleId":"catalog","moduleVersion":"1.0.0","contributionId":"catalog.shop","surface":"shop"}`)
	request := httptest.NewRequest(http.MethodPost, "/runtime/v1/module-sessions", body)
	request.Header.Set("Authorization", "Bearer "+operatorToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("session status=%d body=%s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	claims, err := sessionVerifier.Verify(envelope.Data.Token)
	if err != nil {
		t.Fatal(err)
	}
	scope := modulesession.ScopeFor(claims, "catalog.product")
	if scope.Type != modulesession.DataScopeDepartments || len(scope.DepartmentIDs) != 1 || scope.DepartmentIDs[0] != 10 {
		t.Fatalf("unexpected scope: %#v", scope)
	}
}

func TestModuleSessionUsesVerifiedIdentityTenant(t *testing.T) {
	handler, identityIssuer, sessionVerifier := testRuntime(t)
	accessToken, _ := identityIssuer.Sign(accessidentity.Claims{Subject: "user-1", Realm: accessidentity.RealmMerchant, AppID: 1001, MerchantID: 2001}, time.Hour)
	body := bytes.NewBufferString(`{"moduleId":"catalog","moduleVersion":"1.0.0","contributionId":"catalog.shop","surface":"shop","appId":9999,"merchantId":9999}`)
	request := httptest.NewRequest(http.MethodPost, "/runtime/v1/module-sessions", body)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	claims, err := sessionVerifier.Verify(envelope.Data.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.AppID != 1001 || claims.MerchantID != 2001 || claims.Subject != "user-1" {
		t.Fatalf("untrusted tenant reached session: %#v", claims)
	}
}

func TestRuntimeFiltersAndRejectsMissingPermission(t *testing.T) {
	handler, identityIssuer, _ := testRuntime(t)
	accessToken, _ := identityIssuer.Sign(accessidentity.Claims{Subject: "unassigned-user", Realm: accessidentity.RealmMerchant, AppID: 1001, MerchantID: 2001}, time.Hour)
	request := httptest.NewRequest(http.MethodGet, "/runtime/v1/contributions?surface=shop", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"items":[]`)) {
		t.Fatalf("unexpected contribution response: %s", response.Body.String())
	}

	body := bytes.NewBufferString(`{"moduleId":"catalog","moduleVersion":"1.0.0","contributionId":"catalog.shop","surface":"shop"}`)
	request = httptest.NewRequest(http.MethodPost, "/runtime/v1/module-sessions", body)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("got status %d", response.Code)
	}
}

func TestPlatformOperatorCanDiscoverMachineReadableModuleCapabilities(t *testing.T) {
	handler, identityIssuer, _ := testRuntime(t)
	accessToken, _ := identityIssuer.Sign(accessidentity.Claims{Subject: "user-1", Realm: accessidentity.RealmPlatform, AppID: 1001, MerchantID: 2001}, time.Hour)
	request := httptest.NewRequest(http.MethodGet, "/runtime/v1/module-catalog", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"catalog.product.list"`)) {
		t.Fatalf("capability catalog status=%d body=%s", response.Code, response.Body.String())
	}

	merchantToken, _ := identityIssuer.Sign(accessidentity.Claims{Subject: "user-1", Realm: accessidentity.RealmMerchant, AppID: 1001, MerchantID: 2001}, time.Hour)
	request = httptest.NewRequest(http.MethodGet, "/runtime/v1/module-catalog", nil)
	request.Header.Set("Authorization", "Bearer "+merchantToken)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("merchant capability catalog status=%d", response.Code)
	}
}

func TestInternalEndpointsEnforceWorkloadPermissions(t *testing.T) {
	gatewayIssuer, _ := workloadidentity.NewIssuer("k51SIJI3oT-PJGXf2uWjL6jyTiDJ1Nwmk6l1ehqDrqA", "gateway-1", "test-workloads", "liveshop-gateway", "liveshop-platform-internal")
	ciIssuer, _ := workloadidentity.NewIssuer("MEdxJQh5ZzEe9NhL8TQ7G5rCqZ1Cr00n6DVMiCayO_8", "ci-1", "test-workloads", "module-release-ci", "liveshop-platform-internal")
	workloads, _ := workloadidentity.NewVerifier(map[string]workloadidentity.TrustedWorkload{
		"gateway-1": {PublicKey: "ky88xYQS66lbhNA-cUpijVuxRWcWAdFRgMIHFKF7PkA", Subject: "liveshop-gateway", Permissions: []string{"registry.routes.read", "registry.capabilities.read"}},
		"ci-1":      {PublicKey: "fkfxuRj0sxDYBT3U_qghrTrtjfv4y3djZObZ-EL_Zho", Subject: "module-release-ci", Permissions: []string{"registry.release.write", "registry.activation.write"}},
	}, "test-workloads", "liveshop-platform-internal")
	dependencies := platformregistry.Dependencies{Modules: moduleregistry.NewStore(), Workloads: workloads}
	handler := startTestServer(t, New(dependencies, logic.New(dependencies), Config{}))

	gatewayToken, _ := gatewayIssuer.Sign(time.Minute)
	request := httptest.NewRequest(http.MethodGet, "/internal/v1/module-registry/routes", nil)
	request.Header.Set("Authorization", "Bearer "+gatewayToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("gateway route read got %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/internal/v1/module-registry/capabilities", nil)
	request.Header.Set("Authorization", "Bearer "+gatewayToken)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("gateway capability read got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/internal/v1/module-registry/releases", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer "+gatewayToken)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("gateway release write got %d", response.Code)
	}

	ciToken, _ := ciIssuer.Sign(time.Minute)
	request = httptest.NewRequest(http.MethodGet, "/internal/v1/module-registry/routes", nil)
	request.Header.Set("Authorization", "Bearer "+ciToken)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("CI route read got %d", response.Code)
	}
}

func TestApplicationErrorReasonIsExposed(t *testing.T) {
	handler, _, _ := testRuntime(t)
	adminSigner, _ := modulesession.NewIssuer(sessionPrivateKey, "session-1", "test-platform")
	adminToken, _ := adminSigner.Sign(modulesession.Claims{
		Subject: "user-1", Realm: accessidentity.RealmPlatform, ModuleID: "platform", ModuleVersion: "1.0.0", Surface: "admin",
		ContributionID: "platform.admin.iam", AppID: 1001, MerchantID: 2001,
		Permissions:   []string{"platform.iam.manage"},
		AllowedRoutes: []modulesession.RouteScope{{Methods: []string{"PUT"}, Prefix: "/admin/platform/iam"}},
	}, time.Minute)
	request := httptest.NewRequest(http.MethodPut, "/admin/platform/iam/departments/not-an-id", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer "+adminToken)
	request.Header.Set("X-Liveshop-Surface", "admin")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte(`"reason":"platform.iam.invalid"`)) {
		t.Fatalf("application error contract missing: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestPlatformControlPlaneCannotDeactivateItself(t *testing.T) {
	handler, _, _ := testRuntime(t)
	adminSigner, _ := modulesession.NewIssuer(sessionPrivateKey, "session-1", "test-platform")
	adminToken, _ := adminSigner.Sign(modulesession.Claims{
		Subject: "user-1", Realm: accessidentity.RealmPlatform, ModuleID: "platform", ModuleVersion: "1.1.1", Surface: "admin",
		ContributionID: "platform.admin.registry", AppID: 1001, MerchantID: 2001,
		Permissions:   []string{"platform.registry.manage"},
		AllowedRoutes: []modulesession.RouteScope{{Methods: []string{"DELETE"}, Prefix: "/admin/platform/registry"}},
	}, time.Minute)
	request := httptest.NewRequest(http.MethodDelete, "/admin/platform/registry/modules/platform/activation", nil)
	request.Header.Set("Authorization", "Bearer "+adminToken)
	request.Header.Set("X-Liveshop-Surface", "admin")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("platform self-deactivation status=%d body=%s", response.Code, response.Body.String())
	}
}
