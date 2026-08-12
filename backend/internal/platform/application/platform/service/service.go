// Package service 声明 Platform HTTP 应用边界。
package service

import (
	"context"

	apiaudit "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/audit"
	apiiam "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/iam"
	apiidentity "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/identity"
	apiregistry "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/registry"
	apiruntime "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/runtime"
	apisettings "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/settings"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/identity"
)

// Auth DTOs are exposed at the application boundary so transports never need
// to depend on the registry implementation package.
type LoginInput = identity.LoginInput
type AuthResult = identity.Result

var ErrInvalidRefresh = identity.ErrInvalidRefresh

type Auth interface {
	Login(context.Context, LoginInput) (AuthResult, error)
	Refresh(context.Context, string) (AuthResult, error)
	Logout(context.Context, string) error
}

type Registry interface {
	RegisterRelease(context.Context, []byte) (*apiregistry.RegisterReleaseRes, error)
	Activate(context.Context, *apiregistry.ActivateReq) error
	Routes(context.Context) (*apiregistry.RoutesRes, error)
	Capabilities(context.Context) (*apiregistry.CapabilitiesRes, error)
	Modules(context.Context) (*apiregistry.ModulesRes, error)
	AdminActivate(context.Context, *apiregistry.AdminActivateReq) error
	AdminDeactivate(context.Context, string) error
}

type Runtime interface {
	Contributions(context.Context, string) (*apiruntime.ContributionsRes, error)
	ModuleSession(context.Context, *apiruntime.ModuleSessionReq) (*apiruntime.ModuleSessionRes, error)
	Authorization(context.Context) iam.Authorization
	ModuleCatalog(context.Context) (*apiruntime.ModuleCatalogRes, error)
}

type Settings interface {
	ListSettings(context.Context) (*apisettings.ListRes, error)
	GetSetting(context.Context, string) (*apisettings.GetRes, error)
	PutSetting(context.Context, *apisettings.PutReq) (*apisettings.PutRes, error)
}

type Audit interface {
	ListAudit(context.Context, int) (*apiaudit.ListRes, error)
}

type Identity interface {
	Accounts(context.Context) (*apiidentity.AccountsRes, error)
	PutAccount(context.Context, *apiidentity.PutAccountReq) (*apiidentity.PutAccountRes, error)
}

type IAM interface {
	Permissions(context.Context) (*apiiam.PermissionsRes, error)
	Departments(context.Context) (*apiiam.DepartmentsRes, error)
	PutDepartment(context.Context, *apiiam.PutDepartmentReq) (*apiiam.PutDepartmentRes, error)
	Roles(context.Context) (*apiiam.RolesRes, error)
	PutRole(context.Context, *apiiam.PutRoleReq) (*apiiam.PutRoleRes, error)
	PutRolePolicy(context.Context, *apiiam.PutRolePolicyReq) (*apiiam.PutRolePolicyRes, error)
	PutSubjectAssignment(context.Context, *apiiam.PutSubjectAssignmentReq) error
	SubjectAccess(context.Context, string) (*apiiam.SubjectAccessRes, error)
}

type Application interface {
	Auth
	Registry
	Runtime
	Settings
	Audit
	Identity
	IAM
}
