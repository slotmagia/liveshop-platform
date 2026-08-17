package http

import (
	"context"

	apiregistry "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/registry"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type RegistryController struct{ service service.Registry }

func NewRegistry(application service.Registry) *RegistryController {
	return &RegistryController{service: application}
}

func (c *RegistryController) Modules(ctx context.Context, _ *apiregistry.ModulesReq) (*apiregistry.ModulesRes, error) {
	items, err := c.service.Modules(ctx)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apiregistry.ModulesRes, 0, len(items))
	for _, item := range items {
		releases := make([]apiregistry.ReleaseInfo, 0, len(item.Releases))
		for _, release := range item.Releases {
			releases = append(releases, apiregistry.ReleaseInfo{Version: release.Version, Digest: release.Digest})
		}
		response = append(response, apiregistry.ModuleInfo{
			ID:            item.ID,
			Name:          item.Name,
			ActiveVersion: item.ActiveVersion,
			Releases:      releases,
		})
	}
	return &response, nil
}

func (c *RegistryController) Capabilities(ctx context.Context, _ *apiregistry.CapabilitiesReq) (*apiregistry.CapabilitiesRes, error) {
	catalog, err := c.service.Capabilities(ctx)
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apiregistry.CapabilitiesRes{Revision: catalog.Revision, Items: catalogItems(catalog.Items)}, nil
}

func (c *RegistryController) Activate(ctx context.Context, req *apiregistry.ActivateReq) (*apiregistry.ActivateRes, error) {
	if err := c.service.Activate(ctx, appmodel.Activation{ModuleID: req.ModuleID, Version: req.Version}); err != nil {
		return nil, web.Failure(err)
	}
	return &apiregistry.ActivateRes{}, nil
}

func (c *RegistryController) Deactivate(ctx context.Context, req *apiregistry.DeactivateReq) (*apiregistry.DeactivateRes, error) {
	if err := c.service.Deactivate(ctx, req.ModuleID); err != nil {
		return nil, web.Failure(err)
	}
	return &apiregistry.DeactivateRes{}, nil
}

func catalogItems(items []model.ModuleCapabilityCatalog) []apiregistry.ModuleCapabilityCatalog {
	output := make([]apiregistry.ModuleCapabilityCatalog, 0, len(items))
	for _, item := range items {
		releases := make([]apiregistry.CapabilityRelease, 0, len(item.Releases))
		for _, release := range item.Releases {
			releases = append(releases, apiregistry.CapabilityRelease{
				Version:       release.Version,
				Digest:        release.Digest,
				Active:        release.Active,
				Backend:       release.Backend,
				Permissions:   release.Permissions,
				Contributions: release.Contributions,
			})
		}
		output = append(output, apiregistry.ModuleCapabilityCatalog{
			ID:            item.ID,
			Name:          item.Name,
			ActiveVersion: item.ActiveVersion,
			Releases:      releases,
		})
	}
	return output
}
