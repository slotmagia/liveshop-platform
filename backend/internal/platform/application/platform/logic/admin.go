package logic

import (
	"context"
	"errors"
	"net/http"
	"strings"

	apiaudit "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/audit"
	apiiam "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/iam"
	apiidentity "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/identity"
	apisettings "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/settings"
	"github.com/liveshop-platform/module-platform/internal/platform/common/requestctx"
	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
	platformaudit "github.com/liveshop-platform/module-platform/internal/platform/registry/audit"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	platformidentity "github.com/liveshop-platform/module-platform/internal/platform/registry/identity"
	platformsettings "github.com/liveshop-platform/module-platform/internal/platform/registry/settings"
)

func settingScope(ctx context.Context) platformsettings.Scope {
	current := requestctx.Identity(ctx)
	return platformsettings.Scope{Realm: current.Realm, AppID: current.AppID, MerchantID: current.MerchantID, Subject: current.Subject}
}

func (l *Logic) ListSettings(ctx context.Context) (*apisettings.ListRes, error) {
	if l.deps.Settings == nil {
		return nil, unavailable("settings")
	}
	items, err := l.deps.Settings.List(ctx, settingScope(ctx))
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, err)
	}
	result := apisettings.ListRes(items)
	return &result, nil
}

func (l *Logic) GetSetting(ctx context.Context, namespace string) (*apisettings.GetRes, error) {
	if l.deps.Settings == nil {
		return nil, unavailable("settings")
	}
	item, err := l.deps.Settings.Get(ctx, settingScope(ctx), namespace)
	if err != nil {
		return nil, web.Error(http.StatusBadRequest, err)
	}
	result := apisettings.GetRes(item)
	return &result, nil
}

func (l *Logic) PutSetting(ctx context.Context, req *apisettings.PutReq) (*apisettings.PutRes, error) {
	if l.deps.Settings == nil {
		return nil, unavailable("settings")
	}
	item, err := l.deps.Settings.Put(ctx, settingScope(ctx), req.Namespace, req.ExpectedVersion, req.Value)
	if err != nil {
		if errors.Is(err, platformsettings.ErrConflict) {
			return nil, web.Error(http.StatusConflict, err)
		}
		return nil, web.Error(http.StatusBadRequest, err)
	}
	result := apisettings.PutRes(item)
	return &result, nil
}

func (l *Logic) ListAudit(ctx context.Context, limit int) (*apiaudit.ListRes, error) {
	if l.deps.Audit == nil {
		return nil, unavailable("audit")
	}
	current := requestctx.Identity(ctx)
	items, err := l.deps.Audit.List(ctx, platformaudit.Scope{Realm: current.Realm, AppID: current.AppID, MerchantID: current.MerchantID}, limit)
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, err)
	}
	result := apiaudit.ListRes(items)
	return &result, nil
}

func (l *Logic) Accounts(ctx context.Context) (*apiidentity.AccountsRes, error) {
	if l.deps.Identity == nil {
		return nil, unavailable("identity")
	}
	current := requestctx.Identity(ctx)
	items, err := l.deps.Identity.Accounts(ctx, platformidentity.AccountScope{AppID: current.AppID, MerchantID: current.MerchantID})
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, err)
	}
	result := apiidentity.AccountsRes(items)
	return &result, nil
}

func (l *Logic) PutAccount(ctx context.Context, req *apiidentity.PutAccountReq) (*apiidentity.PutAccountRes, error) {
	if l.deps.Identity == nil {
		return nil, unavailable("identity")
	}
	current := requestctx.Identity(ctx)
	realm := strings.ToUpper(strings.TrimSpace(req.Realm))
	subject := strings.TrimSpace(req.Subject)
	if realm == current.Realm && subject == current.Subject && req.Status == "DISABLED" {
		return nil, web.Error(http.StatusForbidden, errors.New("an operator cannot disable their own account"))
	}
	actor := platformidentity.Principal{Realm: current.Realm, AppID: current.AppID, MerchantID: current.MerchantID, Subject: current.Subject}
	input := platformidentity.PutAccountInput{Realm: realm, Subject: subject, Username: req.Username, Password: req.Password, Status: req.Status, ExpectedVersion: req.ExpectedVersion}
	item, err := l.deps.Identity.PutAccount(ctx, actor, platformidentity.AccountScope{AppID: current.AppID, MerchantID: current.MerchantID}, input)
	if err != nil {
		switch {
		case errors.Is(err, platformidentity.ErrAccountConflict):
			return nil, web.Error(http.StatusConflict, err)
		case errors.Is(err, platformidentity.ErrAccountNotFound):
			return nil, web.Error(http.StatusNotFound, err)
		default:
			return nil, web.Error(http.StatusBadRequest, err)
		}
	}
	result := apiidentity.PutAccountRes(item)
	return &result, nil
}

func iamMutationContext(ctx context.Context) context.Context {
	current := requestctx.Identity(ctx)
	return iam.WithAuditActor(ctx, current.Realm, current.Subject)
}

func (l *Logic) Permissions(ctx context.Context) (*apiiam.PermissionsRes, error) {
	items, err := l.deps.IAM.Permissions(ctx)
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, err)
	}
	result := apiiam.PermissionsRes(items)
	return &result, nil
}

func (l *Logic) Departments(ctx context.Context) (*apiiam.DepartmentsRes, error) {
	items, err := l.deps.IAM.Departments(ctx, requestctx.Tenant(ctx))
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, err)
	}
	result := apiiam.DepartmentsRes(items)
	return &result, nil
}

func (l *Logic) PutDepartment(ctx context.Context, req *apiiam.PutDepartmentReq) (*apiiam.PutDepartmentRes, error) {
	if req.DepartmentID <= 0 {
		return nil, iamError(iam.ErrInvalid)
	}
	item, err := l.deps.IAM.PutDepartment(iamMutationContext(ctx), requestctx.Tenant(ctx), iam.Department{ID: req.DepartmentID, ParentID: req.ParentID, Name: req.Name, Status: req.Status}, req.ExpectedVersion)
	if err != nil {
		return nil, iamError(err)
	}
	result := apiiam.PutDepartmentRes(item)
	return &result, nil
}

func (l *Logic) Roles(ctx context.Context) (*apiiam.RolesRes, error) {
	items, err := l.deps.IAM.Roles(ctx, requestctx.Tenant(ctx))
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, err)
	}
	result := apiiam.RolesRes(items)
	return &result, nil
}

func (l *Logic) PutRole(ctx context.Context, req *apiiam.PutRoleReq) (*apiiam.PutRoleRes, error) {
	if req.RoleID <= 0 {
		return nil, iamError(iam.ErrInvalid)
	}
	if req.SuperAdmin {
		return nil, web.Error(http.StatusForbidden, errors.New("super administrator roles are provisioned outside delegated IAM administration"))
	}
	item, err := l.deps.IAM.PutRole(iamMutationContext(ctx), requestctx.Tenant(ctx), iam.Role{ID: req.RoleID, Name: req.Name, Status: req.Status, SuperAdmin: req.SuperAdmin}, req.ExpectedVersion)
	if err != nil {
		return nil, iamError(err)
	}
	result := apiiam.PutRoleRes(item)
	return &result, nil
}

func (l *Logic) PutRolePolicy(ctx context.Context, req *apiiam.PutRolePolicyReq) (*apiiam.PutRolePolicyRes, error) {
	if req.RoleID <= 0 {
		return nil, iamError(iam.ErrInvalid)
	}
	item, err := l.deps.IAM.SetRolePolicy(iamMutationContext(ctx), requestctx.Tenant(ctx), req.RoleID, req.ExpectedVersion, iam.RolePolicy{Permissions: req.Permissions, Scopes: req.Scopes})
	if err != nil {
		return nil, iamError(err)
	}
	result := apiiam.PutRolePolicyRes(item)
	return &result, nil
}

func (l *Logic) PutSubjectAssignment(ctx context.Context, req *apiiam.PutSubjectAssignmentReq) error {
	if strings.TrimSpace(req.Subject) == "" {
		return iamError(iam.ErrInvalid)
	}
	input := iam.SubjectAssignment{RoleIDs: req.RoleIDs, Departments: req.Departments}
	if err := l.deps.IAM.SetSubjectAssignment(iamMutationContext(ctx), requestctx.Tenant(ctx), req.Subject, input); err != nil {
		return iamError(err)
	}
	return nil
}

func (l *Logic) SubjectAccess(ctx context.Context, subject string) (*apiiam.SubjectAccessRes, error) {
	authorization, err := l.deps.IAM.Effective(ctx, requestctx.Tenant(ctx), strings.TrimSpace(subject))
	if err != nil {
		return nil, iamError(err)
	}
	result := apiiam.SubjectAccessRes(authorization)
	return &result, nil
}
