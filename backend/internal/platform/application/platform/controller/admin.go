package controller

import (
	"context"

	apiaudit "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/audit"
	apiiam "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/iam"
	apiidentity "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/identity"
	apisettings "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/settings"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/service"
)

type SettingsReaderController struct{ service service.Settings }

func NewSettingsReader(application service.Settings) *SettingsReaderController {
	return &SettingsReaderController{service: application}
}

func (c *SettingsReaderController) List(ctx context.Context, _ *apisettings.ListReq) (*apisettings.ListRes, error) {
	return c.service.ListSettings(ctx)
}

func (c *SettingsReaderController) Get(ctx context.Context, req *apisettings.GetReq) (*apisettings.GetRes, error) {
	return c.service.GetSetting(ctx, req.Namespace)
}

type SettingsWriterController struct{ service service.Settings }

func NewSettingsWriter(application service.Settings) *SettingsWriterController {
	return &SettingsWriterController{service: application}
}

func (c *SettingsWriterController) Put(ctx context.Context, req *apisettings.PutReq) (*apisettings.PutRes, error) {
	return c.service.PutSetting(ctx, req)
}

type AuditController struct{ service service.Audit }

func NewAudit(application service.Audit) *AuditController {
	return &AuditController{service: application}
}
func (c *AuditController) List(ctx context.Context, req *apiaudit.ListReq) (*apiaudit.ListRes, error) {
	return c.service.ListAudit(ctx, req.Limit)
}

type IdentityController struct{ service service.Identity }

func NewIdentity(application service.Identity) *IdentityController {
	return &IdentityController{service: application}
}
func (c *IdentityController) Accounts(ctx context.Context, _ *apiidentity.AccountsReq) (*apiidentity.AccountsRes, error) {
	return c.service.Accounts(ctx)
}
func (c *IdentityController) PutAccount(ctx context.Context, req *apiidentity.PutAccountReq) (*apiidentity.PutAccountRes, error) {
	return c.service.PutAccount(ctx, req)
}

type IAMController struct{ service service.IAM }

func NewIAM(application service.IAM) *IAMController { return &IAMController{service: application} }
func (c *IAMController) Permissions(ctx context.Context, _ *apiiam.PermissionsReq) (*apiiam.PermissionsRes, error) {
	return c.service.Permissions(ctx)
}
func (c *IAMController) Departments(ctx context.Context, _ *apiiam.DepartmentsReq) (*apiiam.DepartmentsRes, error) {
	return c.service.Departments(ctx)
}
func (c *IAMController) PutDepartment(ctx context.Context, req *apiiam.PutDepartmentReq) (*apiiam.PutDepartmentRes, error) {
	return c.service.PutDepartment(ctx, req)
}
func (c *IAMController) Roles(ctx context.Context, _ *apiiam.RolesReq) (*apiiam.RolesRes, error) {
	return c.service.Roles(ctx)
}
func (c *IAMController) PutRole(ctx context.Context, req *apiiam.PutRoleReq) (*apiiam.PutRoleRes, error) {
	return c.service.PutRole(ctx, req)
}
func (c *IAMController) PutRolePolicy(ctx context.Context, req *apiiam.PutRolePolicyReq) (*apiiam.PutRolePolicyRes, error) {
	return c.service.PutRolePolicy(ctx, req)
}
func (c *IAMController) PutSubjectAssignment(ctx context.Context, req *apiiam.PutSubjectAssignmentReq) (*apiiam.PutSubjectAssignmentRes, error) {
	if err := c.service.PutSubjectAssignment(ctx, req); err != nil {
		return nil, err
	}
	return &apiiam.PutSubjectAssignmentRes{}, nil
}
func (c *IAMController) SubjectAccess(ctx context.Context, req *apiiam.SubjectAccessReq) (*apiiam.SubjectAccessRes, error) {
	return c.service.SubjectAccess(ctx, req.Subject)
}
