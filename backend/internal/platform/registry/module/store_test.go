package module

import (
	"context"
	"testing"

	"github.com/liveshop-platform/contracts/modulemanifest"
)

func fixture(id, prefix string) modulemanifest.Manifest {
	permission := id + ".item.read"
	return modulemanifest.Manifest{
		APIVersion: modulemanifest.APIVersion,
		Kind:       "ModuleRelease",
		Metadata:   modulemanifest.Metadata{ID: id, Name: id, Version: "1.0.0"},
		Spec: modulemanifest.Spec{
			Backend: modulemanifest.Backend{Service: id, Origin: "http://" + id, HTTPRoutes: []modulemanifest.HTTPRoute{{
				Surface: "admin", Prefix: prefix, Operations: []modulemanifest.HTTPOperation{{
					ID: id + ".item.list", Method: "GET", Path: prefix, Summary: "List items", Description: "Lists module items.", Authentication: "module-session", Idempotency: "safe", RequiredPermissions: []string{permission},
					Responses: []modulemanifest.CapabilityResponse{{Status: 200, Description: "Item list"}},
				}},
			}}},
			Permissions: []modulemanifest.PermissionDefinition{{Code: permission, Name: "Read items", Resource: id + ".item", Action: "read"}},
		},
	}
}

func TestActivationRejectsRouteConflict(t *testing.T) {
	store := NewStore()
	for _, manifest := range []modulemanifest.Manifest{fixture("catalog", "/admin/items"), fixture("order", "/admin/items")} {
		if _, err := store.Register(context.Background(), manifest); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Activate(context.Background(), "catalog", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), "order", "1.0.0"); err == nil {
		t.Fatal("expected route conflict")
	}
}

func TestActivationRejectsOverlappingRoutePrefix(t *testing.T) {
	store := NewStore()
	for _, manifest := range []modulemanifest.Manifest{fixture("catalog", "/admin/items"), fixture("order", "/admin/items/private")} {
		if _, err := store.Register(context.Background(), manifest); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Activate(context.Background(), "catalog", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), "order", "1.0.0"); err == nil {
		t.Fatal("expected overlapping route conflict")
	}
}

func TestDuplicateActivationIsIdempotent(t *testing.T) {
	store := NewStore()
	if _, err := store.Register(context.Background(), fixture("catalog", "/admin/items")); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), "catalog", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	revision, _, err := store.Routes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), "catalog", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	after, _, err := store.Routes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if after != revision {
		t.Fatalf("duplicate activation changed revision: before=%d after=%d", revision, after)
	}
}

func TestModuleListingAndDeactivation(t *testing.T) {
	store := NewStore()
	if _, err := store.Register(context.Background(), fixture("catalog", "/admin/items")); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), "catalog", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	items, err := store.Modules(context.Background())
	if err != nil || len(items) != 1 || items[0].ActiveVersion != "1.0.0" {
		t.Fatalf("unexpected modules: %#v err=%v", items, err)
	}
	if err := store.Deactivate(context.Background(), "catalog"); err != nil {
		t.Fatal(err)
	}
	if _, routes, err := store.Routes(context.Background()); err != nil || len(routes) != 0 {
		t.Fatalf("deactivated routes remain: %#v err=%v", routes, err)
	}
}

func TestCapabilityCatalogIsDerivedFromImmutableRelease(t *testing.T) {
	store := NewStore()
	manifest := fixture("catalog", "/admin/catalog")
	manifest.Metadata.Name = "Catalog Capability Module"
	if _, err := store.Register(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), "catalog", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	revision, catalogs, err := store.CapabilityCatalogs(context.Background())
	if err != nil || revision != 2 || len(catalogs) != 1 || len(catalogs[0].Releases) != 1 {
		t.Fatalf("unexpected capability catalog: revision=%d catalogs=%#v err=%v", revision, catalogs, err)
	}
	release := catalogs[0].Releases[0]
	if catalogs[0].Name != "Catalog Capability Module" || !release.Active || len(release.Backend.HTTPRoutes) != 1 || release.Backend.HTTPRoutes[0].Operations[0].ID != "catalog.item.list" {
		t.Fatalf("registered capabilities were not preserved: %#v", release)
	}

	// Returned slices are decoded/owned by the immutable release snapshot; a
	// caller must not be able to alter the registry's next discovery response.
	release.Backend.HTTPRoutes[0].Operations[0].Summary = "tampered"
	_, again, err := store.CapabilityCatalogs(context.Background())
	if err != nil || again[0].Releases[0].Backend.HTTPRoutes[0].Operations[0].Summary != "List items" {
		t.Fatal("capability catalog response mutated registry state")
	}
}
