package controller

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"
	apiregistry "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/registry"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/service"
)

type ReleaseController struct{ service service.Registry }

func NewRelease(application service.Registry) *ReleaseController {
	return &ReleaseController{service: application}
}
func (c *ReleaseController) RegisterRelease(ctx context.Context, _ *apiregistry.RegisterReleaseReq) (*apiregistry.RegisterReleaseRes, error) {
	return c.service.RegisterRelease(ctx, ghttp.RequestFromCtx(ctx).GetBody())
}

type ActivationController struct{ service service.Registry }

func NewActivation(application service.Registry) *ActivationController {
	return &ActivationController{service: application}
}
func (c *ActivationController) Activate(ctx context.Context, req *apiregistry.ActivateReq) (*apiregistry.ActivateRes, error) {
	if err := c.service.Activate(ctx, req); err != nil {
		return nil, err
	}
	return &apiregistry.ActivateRes{}, nil
}

type RoutesController struct{ service service.Registry }

func NewRoutes(application service.Registry) *RoutesController {
	return &RoutesController{service: application}
}
func (c *RoutesController) Routes(ctx context.Context, _ *apiregistry.RoutesReq) (*apiregistry.RoutesRes, error) {
	return c.service.Routes(ctx)
}

type CapabilitiesController struct{ service service.Registry }

func NewCapabilities(application service.Registry) *CapabilitiesController {
	return &CapabilitiesController{service: application}
}
func (c *CapabilitiesController) Capabilities(ctx context.Context, _ *apiregistry.CapabilitiesReq) (*apiregistry.CapabilitiesRes, error) {
	return c.service.Capabilities(ctx)
}

type AdminRegistryController struct{ service service.Registry }

func NewAdminRegistry(application service.Registry) *AdminRegistryController {
	return &AdminRegistryController{service: application}
}
func (c *AdminRegistryController) Modules(ctx context.Context, _ *apiregistry.ModulesReq) (*apiregistry.ModulesRes, error) {
	return c.service.Modules(ctx)
}
func (c *AdminRegistryController) AdminCapabilities(ctx context.Context, _ *apiregistry.AdminCapabilitiesReq) (*apiregistry.CapabilitiesRes, error) {
	return c.service.Capabilities(ctx)
}
func (c *AdminRegistryController) AdminActivate(ctx context.Context, req *apiregistry.AdminActivateReq) (*apiregistry.AdminActivateRes, error) {
	if err := c.service.AdminActivate(ctx, req); err != nil {
		return nil, err
	}
	return &apiregistry.AdminActivateRes{}, nil
}
func (c *AdminRegistryController) AdminDeactivate(ctx context.Context, req *apiregistry.AdminDeactivateReq) (*apiregistry.AdminDeactivateRes, error) {
	if err := c.service.AdminDeactivate(ctx, req.ModuleID); err != nil {
		return nil, err
	}
	return &apiregistry.AdminDeactivateRes{}, nil
}
