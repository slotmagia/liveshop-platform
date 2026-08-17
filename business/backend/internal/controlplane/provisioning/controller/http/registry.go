// Package http adapts the provisioning HTTP contract to its application
// boundary. Each controller is bound under its own workload permission.
package http

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/common/web"
	apiregistry "github.com/liveshop-platform/module-platform/internal/controlplane/provisioning/api/http/v1/registry"
	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning/appmodel"
	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning/service"
)

type ReleaseController struct{ service service.Provisioning }

func NewRelease(application service.Provisioning) *ReleaseController {
	return &ReleaseController{service: application}
}

func (c *ReleaseController) RegisterRelease(ctx context.Context, _ *apiregistry.RegisterReleaseReq) (*apiregistry.RegisterReleaseRes, error) {
	registered, err := c.service.RegisterRelease(ctx, web.RawBody(ctx))
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apiregistry.RegisterReleaseRes{Digest: registered.Digest}, nil
}

type ActivationController struct{ service service.Provisioning }

func NewActivation(application service.Provisioning) *ActivationController {
	return &ActivationController{service: application}
}

func (c *ActivationController) Activate(ctx context.Context, req *apiregistry.ActivateReq) (*apiregistry.ActivateRes, error) {
	if err := c.service.Activate(ctx, appmodel.Activation{ModuleID: req.ModuleID, Version: req.Version}); err != nil {
		return nil, web.Failure(err)
	}
	return &apiregistry.ActivateRes{}, nil
}

type RoutesController struct{ service service.Provisioning }

func NewRoutes(application service.Provisioning) *RoutesController {
	return &RoutesController{service: application}
}

func (c *RoutesController) Routes(ctx context.Context, _ *apiregistry.RoutesReq) (*apiregistry.RoutesRes, error) {
	snapshot, err := c.service.Routes(ctx)
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apiregistry.RoutesRes{Revision: snapshot.Revision, Routes: snapshot.Routes}, nil
}
