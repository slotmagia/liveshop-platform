// Package iam owns tenant-scoped users-to-roles, organization membership and
// data-scope policy. Authentication identities never carry authorization grants.
package iam

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lvtuopen-ai/kernel-go/apperror"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
)

var (
	ErrInvalid  = apperror.New("platform.iam.invalid", "iam: invalid input")
	ErrNotFound = apperror.New("platform.iam.not_found", "iam: not found")
	ErrConflict = apperror.New("platform.iam.conflict", "iam: concurrent or conflicting update")
)

const (
	StatusActive   = "ACTIVE"
	StatusDisabled = "DISABLED"

	ScopeAll                   = "ALL"
	ScopeDepartmentAndChildren = "DEPARTMENT_AND_CHILDREN"
	ScopeDepartment            = "DEPARTMENT"
	ScopeSelf                  = "SELF"
	ScopeCustom                = "CUSTOM"
)

type Tenant struct {
	AppID      int64 `json:"appId"`
	MerchantID int64 `json:"merchantId"`
}

func (t Tenant) Valid() bool { return t.AppID > 0 && t.MerchantID > 0 }
func (t Tenant) key() string { return fmt.Sprintf("%d:%d", t.AppID, t.MerchantID) }

type Permission struct {
	ModuleID    string `json:"moduleId"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description,omitempty"`
}

type Department struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parentId,omitempty"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Version  int64  `json:"version"`
}

type Role struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	SuperAdmin bool   `json:"superAdmin"`
	Version    int64  `json:"version"`
}

type ScopeRule struct {
	Resource      string  `json:"resource"`
	Type          string  `json:"type"`
	DepartmentIDs []int64 `json:"departmentIds,omitempty"`
}

type RolePolicy struct {
	Permissions []string    `json:"permissions"`
	Scopes      []ScopeRule `json:"scopes"`
}

type DepartmentMembership struct {
	DepartmentID int64 `json:"departmentId"`
	Primary      bool  `json:"primary"`
}

type SubjectAssignment struct {
	RoleIDs     []int64                `json:"roleIds"`
	Departments []DepartmentMembership `json:"departments"`
}

type Authorization struct {
	Revision    uint64                    `json:"revision"`
	Permissions []string                  `json:"permissions"`
	DataScopes  []modulesession.DataScope `json:"dataScopes"`
}

func (a Authorization) Has(required ...string) bool {
	granted := make(map[string]struct{}, len(a.Permissions))
	for _, code := range a.Permissions {
		granted[code] = struct{}{}
	}
	for _, code := range required {
		if _, ok := granted[code]; !ok {
			return false
		}
	}
	return true
}

type tenantState struct {
	revision    uint64
	departments map[int64]Department
	roles       map[int64]Role
	policies    map[int64]RolePolicy
	assignments map[string]SubjectAssignment
}

type Store struct {
	db          *sql.DB
	mu          sync.RWMutex
	permissions map[string]Permission
	tenants     map[string]*tenantState
}

func NewStore() *Store {
	return &Store{permissions: map[string]Permission{}, tenants: map[string]*tenantState{}}
}

func NewPostgresStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("iam database is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) SeedPermission(permission Permission) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissions[permission.Code] = permission
}

func (s *Store) memoryTenant(tenant Tenant) *tenantState {
	state := s.tenants[tenant.key()]
	if state == nil {
		state = &tenantState{revision: 1, departments: map[int64]Department{}, roles: map[int64]Role{}, policies: map[int64]RolePolicy{}, assignments: map[string]SubjectAssignment{}}
		s.tenants[tenant.key()] = state
	}
	return state
}

func validStatus(status string) bool { return status == StatusActive || status == StatusDisabled }
func validScope(scope string) bool {
	return scope == ScopeAll || scope == ScopeDepartmentAndChildren || scope == ScopeDepartment || scope == ScopeSelf || scope == ScopeCustom
}

func (s *Store) PutDepartment(ctx context.Context, tenant Tenant, input Department, expectedVersion int64) (Department, error) {
	if s.db != nil {
		return s.putDepartmentSQL(ctx, tenant, input, expectedVersion)
	}
	if !tenant.Valid() || input.ID <= 0 || strings.TrimSpace(input.Name) == "" || !validStatus(input.Status) || expectedVersion < 0 {
		return Department{}, ErrInvalid
	}
	input.Name = strings.TrimSpace(input.Name)
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.memoryTenant(tenant)
	current, exists := state.departments[input.ID]
	if exists && sameDepartment(current, input) && (expectedVersion == 0 || expectedVersion == current.Version || expectedVersion == current.Version-1) {
		return current, nil
	}
	if (!exists && expectedVersion != 0) || (exists && current.Version != expectedVersion) {
		return Department{}, ErrConflict
	}
	if input.ParentID != nil {
		parent, ok := state.departments[*input.ParentID]
		if !ok || parent.Status != StatusActive || createsCycle(state.departments, input.ID, *input.ParentID) {
			return Department{}, ErrInvalid
		}
	}
	input.Version = 1
	if exists {
		input.Version = current.Version + 1
	}
	state.departments[input.ID] = input
	state.revision++
	return input, nil
}

func sameDepartment(a, b Department) bool {
	if a.Name != strings.TrimSpace(b.Name) || a.Status != b.Status {
		return false
	}
	if a.ParentID == nil || b.ParentID == nil {
		return a.ParentID == nil && b.ParentID == nil
	}
	return *a.ParentID == *b.ParentID
}

func createsCycle(departments map[int64]Department, id, parent int64) bool {
	seen := map[int64]bool{id: true}
	for parent > 0 {
		if seen[parent] {
			return true
		}
		seen[parent] = true
		current, ok := departments[parent]
		if !ok || current.ParentID == nil {
			return false
		}
		parent = *current.ParentID
	}
	return false
}

func (s *Store) PutRole(ctx context.Context, tenant Tenant, input Role, expectedVersion int64) (Role, error) {
	if s.db != nil {
		return s.putRoleSQL(ctx, tenant, input, expectedVersion)
	}
	if !tenant.Valid() || input.ID <= 0 || strings.TrimSpace(input.Name) == "" || !validStatus(input.Status) || expectedVersion < 0 {
		return Role{}, ErrInvalid
	}
	input.Name = strings.TrimSpace(input.Name)
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.memoryTenant(tenant)
	current, exists := state.roles[input.ID]
	if exists && current.SuperAdmin != input.SuperAdmin {
		return Role{}, ErrInvalid
	}
	if exists && current.Name == strings.TrimSpace(input.Name) && current.Status == input.Status && current.SuperAdmin == input.SuperAdmin && (expectedVersion == 0 || expectedVersion == current.Version || expectedVersion == current.Version-1) {
		return current, nil
	}
	if (!exists && expectedVersion != 0) || (exists && current.Version != expectedVersion) {
		return Role{}, ErrConflict
	}
	input.Version = 1
	if exists {
		input.Version = current.Version + 1
	}
	state.roles[input.ID] = input
	state.revision++
	return input, nil
}

func (s *Store) SetRolePolicy(ctx context.Context, tenant Tenant, roleID, expectedVersion int64, policy RolePolicy) (Role, error) {
	if s.db != nil {
		return s.setRolePolicySQL(ctx, tenant, roleID, expectedVersion, policy)
	}
	if !tenant.Valid() || roleID <= 0 || expectedVersion <= 0 {
		return Role{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if validatePolicy(policy, s.permissions) != nil {
		return Role{}, ErrInvalid
	}
	state := s.memoryTenant(tenant)
	role, ok := state.roles[roleID]
	if !ok {
		return Role{}, ErrNotFound
	}
	policy = normalizePolicy(policy)
	if policiesEqual(state.policies[roleID], policy) && (expectedVersion == role.Version || expectedVersion == role.Version-1) {
		return role, nil
	}
	if role.Version != expectedVersion {
		return Role{}, ErrConflict
	}
	for _, scope := range policy.Scopes {
		for _, departmentID := range scope.DepartmentIDs {
			if _, ok := state.departments[departmentID]; !ok {
				return Role{}, ErrInvalid
			}
		}
	}
	role.Version++
	state.roles[roleID] = role
	state.policies[roleID] = policy
	state.revision++
	return role, nil
}

func validatePolicy(policy RolePolicy, permissions map[string]Permission) error {
	for _, code := range policy.Permissions {
		if _, ok := permissions[code]; !ok {
			return ErrInvalid
		}
	}
	seen := map[string]bool{}
	for _, scope := range policy.Scopes {
		if scope.Resource == "" || !validScope(scope.Type) || seen[scope.Resource] || (scope.Type == ScopeCustom && len(scope.DepartmentIDs) == 0) {
			return ErrInvalid
		}
		seen[scope.Resource] = true
	}
	return nil
}

func normalizePolicy(policy RolePolicy) RolePolicy {
	policy.Permissions = uniqueStrings(policy.Permissions)
	for i := range policy.Scopes {
		policy.Scopes[i].DepartmentIDs = uniqueInt64s(policy.Scopes[i].DepartmentIDs)
	}
	sort.Slice(policy.Scopes, func(i, j int) bool { return policy.Scopes[i].Resource < policy.Scopes[j].Resource })
	return policy
}

func policiesEqual(a, b RolePolicy) bool {
	return fmt.Sprintf("%#v", normalizePolicy(a)) == fmt.Sprintf("%#v", normalizePolicy(b))
}

func (s *Store) SetSubjectAssignment(ctx context.Context, tenant Tenant, subject string, assignment SubjectAssignment) error {
	if s.db != nil {
		return s.setSubjectAssignmentSQL(ctx, tenant, subject, assignment)
	}
	if !tenant.Valid() || strings.TrimSpace(subject) == "" {
		return ErrInvalid
	}
	assignment.RoleIDs = uniqueInt64s(assignment.RoleIDs)
	sort.Slice(assignment.Departments, func(i, j int) bool {
		return assignment.Departments[i].DepartmentID < assignment.Departments[j].DepartmentID
	})
	primary := 0
	seenDepartments := map[int64]bool{}
	for _, membership := range assignment.Departments {
		if membership.Primary {
			primary++
		}
		if seenDepartments[membership.DepartmentID] {
			return ErrInvalid
		}
		seenDepartments[membership.DepartmentID] = true
	}
	if primary > 1 {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.memoryTenant(tenant)
	for _, roleID := range assignment.RoleIDs {
		if role, ok := state.roles[roleID]; !ok || role.Status != StatusActive {
			return ErrInvalid
		}
	}
	for departmentID := range seenDepartments {
		if department, ok := state.departments[departmentID]; !ok || department.Status != StatusActive {
			return ErrInvalid
		}
	}
	if fmt.Sprintf("%#v", state.assignments[subject]) == fmt.Sprintf("%#v", assignment) {
		return nil
	}
	state.assignments[subject] = assignment
	state.revision++
	return nil
}

func (s *Store) Effective(ctx context.Context, tenant Tenant, subject string) (Authorization, error) {
	if s.db != nil {
		return s.effectiveSQL(ctx, tenant, subject)
	}
	if !tenant.Valid() || strings.TrimSpace(subject) == "" {
		return Authorization{}, ErrInvalid
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.tenants[tenant.key()]
	if state == nil {
		return Authorization{Revision: 1}, nil
	}
	assignment := state.assignments[subject]
	permissions := map[string]Permission{}
	rules := make([]ScopeRule, 0)
	for _, roleID := range assignment.RoleIDs {
		role, ok := state.roles[roleID]
		if !ok || role.Status != StatusActive {
			continue
		}
		if role.SuperAdmin {
			for code, permission := range s.permissions {
				permissions[code] = permission
			}
			for _, permission := range s.permissions {
				rules = append(rules, ScopeRule{Resource: permission.Resource, Type: ScopeAll})
			}
			continue
		}
		policy := state.policies[roleID]
		for _, code := range policy.Permissions {
			if permission, ok := s.permissions[code]; ok {
				permissions[code] = permission
			}
		}
		for _, rule := range policy.Scopes {
			if roleGrantsResource(policy, s.permissions, rule.Resource) {
				rules = append(rules, rule)
			}
		}
	}
	return resolveAuthorization(state.revision, subject, permissions, rules, assignment.Departments, state.departments), nil
}

func roleGrantsResource(policy RolePolicy, catalog map[string]Permission, resource string) bool {
	for _, code := range policy.Permissions {
		if catalog[code].Resource == resource {
			return true
		}
	}
	return false
}

func resolveAuthorization(revision uint64, subject string, permissions map[string]Permission, rules []ScopeRule, memberships []DepartmentMembership, departments map[int64]Department) Authorization {
	codes := make([]string, 0, len(permissions))
	resources := map[string]bool{}
	for code, permission := range permissions {
		codes = append(codes, code)
		resources[permission.Resource] = true
	}
	sort.Strings(codes)
	scopes := make([]modulesession.DataScope, 0, len(resources))
	for resource := range resources {
		all, self := false, false
		departmentIDs := map[int64]bool{}
		for _, rule := range rules {
			if rule.Resource != resource {
				continue
			}
			switch rule.Type {
			case ScopeAll:
				all = true
			case ScopeSelf:
				self = true
			case ScopeDepartment:
				for _, membership := range memberships {
					if department, ok := departments[membership.DepartmentID]; ok && department.Status == StatusActive {
						departmentIDs[membership.DepartmentID] = true
					}
				}
			case ScopeDepartmentAndChildren:
				for _, membership := range memberships {
					addDescendants(departmentIDs, departments, membership.DepartmentID)
				}
			case ScopeCustom:
				for _, id := range rule.DepartmentIDs {
					if department, ok := departments[id]; ok && department.Status == StatusActive {
						departmentIDs[id] = true
					}
				}
			}
		}
		scope := modulesession.DataScope{Resource: resource, Type: modulesession.DataScopeNone}
		if all {
			scope.Type = modulesession.DataScopeAll
		} else if len(departmentIDs) > 0 {
			scope.Type = modulesession.DataScopeDepartments
			scope.DepartmentIDs = mapKeys(departmentIDs)
			if self {
				scope.Subject = subject
			}
		} else if self {
			scope.Type = modulesession.DataScopeSelf
			scope.Subject = subject
		}
		scopes = append(scopes, scope)
	}
	sort.Slice(scopes, func(i, j int) bool { return scopes[i].Resource < scopes[j].Resource })
	return Authorization{Revision: revision, Permissions: codes, DataScopes: scopes}
}

func addDescendants(output map[int64]bool, departments map[int64]Department, root int64) {
	if department, ok := departments[root]; !ok || department.Status != StatusActive {
		return
	}
	output[root] = true
	changed := true
	for changed {
		changed = false
		for id, department := range departments {
			if department.Status == StatusActive && department.ParentID != nil && output[*department.ParentID] && !output[id] {
				output[id] = true
				changed = true
			}
		}
	}
}

func mapKeys(values map[int64]bool) []int64 {
	output := make([]int64, 0, len(values))
	for value := range values {
		output = append(output, value)
	}
	sort.Slice(output, func(i, j int) bool { return output[i] < output[j] })
	return output
}
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	output := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			output = append(output, value)
		}
	}
	sort.Strings(output)
	return output
}
func uniqueInt64s(values []int64) []int64 {
	seen := map[int64]bool{}
	output := make([]int64, 0, len(values))
	for _, value := range values {
		if value > 0 && !seen[value] {
			seen[value] = true
			output = append(output, value)
		}
	}
	sort.Slice(output, func(i, j int) bool { return output[i] < output[j] })
	return output
}

func (s *Store) Permissions(ctx context.Context) ([]Permission, error) {
	if s.db != nil {
		return s.permissionsSQL(ctx)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	output := make([]Permission, 0, len(s.permissions))
	for _, item := range s.permissions {
		output = append(output, item)
	}
	sort.Slice(output, func(i, j int) bool { return output[i].Code < output[j].Code })
	return output, nil
}

func (s *Store) Departments(ctx context.Context, tenant Tenant) ([]Department, error) {
	if s.db != nil {
		return s.departmentsSQL(ctx, tenant)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.tenants[tenant.key()]
	if state == nil {
		return nil, nil
	}
	output := make([]Department, 0, len(state.departments))
	for _, item := range state.departments {
		output = append(output, item)
	}
	sort.Slice(output, func(i, j int) bool { return output[i].ID < output[j].ID })
	return output, nil
}

func (s *Store) Roles(ctx context.Context, tenant Tenant) ([]Role, error) {
	if s.db != nil {
		return s.rolesSQL(ctx, tenant)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.tenants[tenant.key()]
	if state == nil {
		return nil, nil
	}
	output := make([]Role, 0, len(state.roles))
	for _, item := range state.roles {
		output = append(output, item)
	}
	sort.Slice(output, func(i, j int) bool { return output[i].ID < output[j].ID })
	return output, nil
}
