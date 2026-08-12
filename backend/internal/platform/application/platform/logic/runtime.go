package logic

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/liveshop-platform/contracts/modulemanifest"
	apiruntime "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/runtime"
	"github.com/liveshop-platform/module-platform/internal/platform/common/requestctx"
	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry/module"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
)

func (l *Logic) Contributions(ctx context.Context, surface string) (*apiruntime.ContributionsRes, error) {
	if surface != "admin" && surface != "merch" && surface != "shop" && surface != "live" {
		return nil, web.Error(http.StatusBadRequest, errors.New("invalid surface"))
	}
	if !accessidentity.RealmAllowsSurface(requestctx.Identity(ctx).Realm, surface) {
		return nil, web.Error(http.StatusForbidden, errors.New("identity realm does not authorize this surface"))
	}
	authorization := requestctx.Authorization(ctx)
	revision, all, err := l.deps.Modules.Contributions(ctx, surface)
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, errors.New("module registry is unavailable"))
	}
	items := make([]modulemanifest.RuntimeContribution, 0, len(all))
	for _, item := range all {
		if authorization.Has(item.Contribution.RequiredPermissions...) {
			items = append(items, item)
		}
	}
	return &apiruntime.ContributionsRes{Revision: revision, Items: items}, nil
}

func (l *Logic) ModuleSession(ctx context.Context, input *apiruntime.ModuleSessionReq) (*apiruntime.ModuleSessionRes, error) {
	contribution, err := l.deps.Modules.Contribution(ctx, input.ModuleID, input.ModuleVersion, input.ContributionID)
	if err != nil || contribution.Surface != input.Surface {
		return nil, web.Error(http.StatusNotFound, platformregistry.ErrNotFound)
	}
	identity := requestctx.Identity(ctx)
	if !accessidentity.RealmAllowsSurface(identity.Realm, input.Surface) {
		return nil, web.Error(http.StatusForbidden, errors.New("identity realm does not authorize this surface"))
	}
	authorization := requestctx.Authorization(ctx)
	if !authorization.Has(contribution.RequiredPermissions...) {
		return nil, web.Error(http.StatusForbidden, errors.New("required contribution permissions are not granted"))
	}
	routes := make([]modulesession.RouteScope, 0, len(contribution.AllowedRoutes))
	for _, route := range contribution.AllowedRoutes {
		if authorization.Has(route.RequiredPermissions...) {
			routes = append(routes, modulesession.RouteScope{Methods: route.Methods, Prefix: route.Prefix})
		}
	}
	if len(routes) == 0 {
		return nil, web.Error(http.StatusForbidden, errors.New("no authorized module route"))
	}
	permissions := filterPrefix(authorization.Permissions, input.ModuleID+".")
	dataScopes := filterDataScopes(authorization.DataScopes, input.ModuleID+".")
	if l.deps.ModuleIssuer == nil {
		return nil, web.Error(http.StatusInternalServerError, errors.New("module session issuer is unavailable"))
	}
	token, err := l.deps.ModuleIssuer.Sign(modulesession.Claims{Subject: identity.Subject, Realm: identity.Realm, ModuleID: input.ModuleID, ModuleVersion: input.ModuleVersion, Surface: input.Surface, ContributionID: input.ContributionID, AllowedRoutes: routes, Permissions: permissions, DataScopes: dataScopes, AuthorizationRevision: authorization.Revision, AppID: identity.AppID, MerchantID: identity.MerchantID}, 5*time.Minute)
	if err != nil {
		return nil, web.Error(http.StatusInternalServerError, err)
	}
	return &apiruntime.ModuleSessionRes{Token: token, ExpiresIn: 300, AuthorizationRevision: authorization.Revision, Permissions: permissions, DataScopes: dataScopes, Tenant: apiruntime.Tenant{AppID: identity.AppID, MerchantID: identity.MerchantID}}, nil
}

func (l *Logic) Authorization(ctx context.Context) iam.Authorization {
	return requestctx.Authorization(ctx)
}

func (l *Logic) ModuleCatalog(ctx context.Context) (*apiruntime.ModuleCatalogRes, error) {
	revision, items, err := l.deps.Modules.CapabilityCatalogs(ctx)
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, errors.New("module capability catalog is unavailable"))
	}
	return &apiruntime.ModuleCatalogRes{Revision: revision, Items: items}, nil
}

func filterPrefix(values []string, prefix string) []string {
	output := make([]string, 0)
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			output = append(output, value)
		}
	}
	return output
}

func filterDataScopes(values []modulesession.DataScope, prefix string) []modulesession.DataScope {
	output := make([]modulesession.DataScope, 0)
	for _, value := range values {
		if strings.HasPrefix(value.Resource, prefix) {
			output = append(output, value)
		}
	}
	return output
}
