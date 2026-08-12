package iam

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/lvtuopen-ai/kernel-go/modulesession"
)

func TestEffectiveAuthorizationCombinesRBACAndDepartmentDescendants(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	store.SeedPermission(Permission{ModuleID: "catalog", Code: "catalog.product.read", Name: "Read products", Resource: "catalog.product", Action: "read"})
	tenant := Tenant{AppID: 1001, MerchantID: 2001}
	root, _ := store.PutDepartment(ctx, tenant, Department{ID: 10, Name: "Sales", Status: StatusActive}, 0)
	_, _ = store.PutDepartment(ctx, tenant, Department{ID: 20, ParentID: &root.ID, Name: "Live", Status: StatusActive}, 0)
	role, _ := store.PutRole(ctx, tenant, Role{ID: 30, Name: "Sales reader", Status: StatusActive}, 0)
	role, err := store.SetRolePolicy(ctx, tenant, role.ID, role.Version, RolePolicy{Permissions: []string{"catalog.product.read"}, Scopes: []ScopeRule{{Resource: "catalog.product", Type: ScopeDepartmentAndChildren}}})
	if err != nil || role.Version != 2 {
		t.Fatalf("policy update failed: role=%#v err=%v", role, err)
	}
	if err := store.SetSubjectAssignment(ctx, tenant, "user-1", SubjectAssignment{RoleIDs: []int64{role.ID}, Departments: []DepartmentMembership{{DepartmentID: root.ID, Primary: true}}}); err != nil {
		t.Fatal(err)
	}

	authorization, err := store.Effective(ctx, tenant, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !authorization.Has("catalog.product.read") {
		t.Fatal("role permission was not granted")
	}
	if len(authorization.DataScopes) != 1 || authorization.DataScopes[0].Type != modulesession.DataScopeDepartments || len(authorization.DataScopes[0].DepartmentIDs) != 2 {
		t.Fatalf("unexpected scope: %#v", authorization.DataScopes)
	}
}

func TestConcurrentDepartmentUpdatesAllowOneWinner(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	tenant := Tenant{AppID: 1, MerchantID: 2}
	department, err := store.PutDepartment(ctx, tenant, Department{ID: 10, Name: "Original", Status: StatusActive}, 0)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, name := range []string{"First", "Second"} {
		wait.Add(1)
		go func(name string) {
			defer wait.Done()
			<-start
			_, current := store.PutDepartment(ctx, tenant, Department{ID: 10, Name: name, Status: StatusActive}, department.Version)
			results <- current
		}(name)
	}
	close(start)
	wait.Wait()
	close(results)
	successes, conflicts := 0, 0
	for result := range results {
		if result == nil {
			successes++
		} else if errors.Is(result, ErrConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected result: %v", result)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestDepartmentRejectsCycleAndStaleOverwrite(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	tenant := Tenant{AppID: 1, MerchantID: 2}
	root, _ := store.PutDepartment(ctx, tenant, Department{ID: 10, Name: "Root", Status: StatusActive}, 0)
	child, _ := store.PutDepartment(ctx, tenant, Department{ID: 20, ParentID: &root.ID, Name: "Child", Status: StatusActive}, 0)
	if _, err := store.PutDepartment(ctx, tenant, Department{ID: root.ID, ParentID: &child.ID, Name: root.Name, Status: root.Status}, root.Version); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cycle got %v", err)
	}
	updated, err := store.PutDepartment(ctx, tenant, Department{ID: child.ID, ParentID: child.ParentID, Name: "Updated", Status: child.Status}, child.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutDepartment(ctx, tenant, Department{ID: child.ID, ParentID: child.ParentID, Name: "Lost update", Status: child.Status}, child.Version); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update got %#v %v", updated, err)
	}
}

func TestRolePolicyRetryIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	tenant := Tenant{AppID: 1, MerchantID: 2}
	store.SeedPermission(Permission{ModuleID: "catalog", Code: "catalog.product.read", Name: "Read", Resource: "catalog.product", Action: "read"})
	role, _ := store.PutRole(ctx, tenant, Role{ID: 1, Name: "Reader", Status: StatusActive}, 0)
	policy := RolePolicy{Permissions: []string{"catalog.product.read"}, Scopes: []ScopeRule{{Resource: "catalog.product", Type: ScopeAll}}}
	first, err := store.SetRolePolicy(ctx, tenant, role.ID, role.Version, policy)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := store.SetRolePolicy(ctx, tenant, role.ID, role.Version, policy)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Version != first.Version {
		t.Fatalf("retry advanced version: first=%d retry=%d", first.Version, retry.Version)
	}
}
