package controller

import (
	"context"

	apiruntime "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/runtime"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/service"
)

type RuntimeController struct{ service service.Runtime }

func NewRuntime(application service.Runtime) *RuntimeController {
	return &RuntimeController{service: application}
}

func (c *RuntimeController) Contributions(ctx context.Context, req *apiruntime.ContributionsReq) (*apiruntime.ContributionsRes, error) {
	return c.service.Contributions(ctx, req.Surface)
}

func (c *RuntimeController) ModuleSession(ctx context.Context, req *apiruntime.ModuleSessionReq) (*apiruntime.ModuleSessionRes, error) {
	return c.service.ModuleSession(ctx, req)
}

func (c *RuntimeController) MyAuthorization(ctx context.Context, _ *apiruntime.MyAuthorizationReq) (*apiruntime.MyAuthorizationRes, error) {
	result := apiruntime.MyAuthorizationRes(c.service.Authorization(ctx))
	return &result, nil
}

type CatalogController struct{ service service.Runtime }

func NewCatalog(application service.Runtime) *CatalogController {
	return &CatalogController{service: application}
}

func (c *CatalogController) ModuleCatalog(ctx context.Context, _ *apiruntime.ModuleCatalogReq) (*apiruntime.ModuleCatalogRes, error) {
	return c.service.ModuleCatalog(ctx)
}
