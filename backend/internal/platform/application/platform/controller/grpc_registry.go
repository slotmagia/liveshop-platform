package controller

import (
	"context"
	"encoding/json"
	"fmt"

	platformv1 "github.com/liveshop-platform/contracts/gen/go/platform/v1"
	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCRegistryController 将公开 Proto 转换到现有 Registry 应用边界。
type GRPCRegistryController struct {
	platformv1.UnimplementedPlatformRegistryServiceServer
	service service.Registry
}

func RegisterGRPCRegistry(server grpc.ServiceRegistrar, application service.Registry) {
	platformv1.RegisterPlatformRegistryServiceServer(server, &GRPCRegistryController{service: application})
}

func (c *GRPCRegistryController) GetRouteSnapshot(ctx context.Context, _ *platformv1.GetRouteSnapshotRequest) (*platformv1.GetRouteSnapshotResponse, error) {
	result, err := c.service.Routes(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	response := &platformv1.GetRouteSnapshotResponse{
		Revision: result.Revision,
		Routes:   make([]*platformv1.ActiveRoute, 0, len(result.Routes)),
	}
	for _, route := range result.Routes {
		response.Routes = append(response.Routes, &platformv1.ActiveRoute{
			ModuleId: route.ModuleID,
			Surface:  route.Surface,
			Prefix:   route.Prefix,
			Service:  route.Service,
			Origin:   route.Origin,
		})
	}
	return response, nil
}

func (c *GRPCRegistryController) GetCapabilityCatalog(ctx context.Context, _ *platformv1.GetCapabilityCatalogRequest) (*platformv1.GetCapabilityCatalogResponse, error) {
	result, err := c.service.Capabilities(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	response := &platformv1.GetCapabilityCatalogResponse{
		Revision: result.Revision,
		Items:    make([]*platformv1.ModuleCapabilityCatalog, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		catalog := &platformv1.ModuleCapabilityCatalog{
			ModuleId:      item.ID,
			Name:          item.Name,
			ActiveVersion: item.ActiveVersion,
			Releases:      make([]*platformv1.CapabilityRelease, 0, len(item.Releases)),
		}
		for _, release := range item.Releases {
			manifestJSON, err := capabilityManifestJSON(item.ID, item.Name, release.Version, release.Backend, release.Permissions, release.Contributions)
			if err != nil {
				return nil, status.Error(codes.Internal, "platform capability manifest serialization failed")
			}
			catalog.Releases = append(catalog.Releases, &platformv1.CapabilityRelease{
				Version:      release.Version,
				Digest:       release.Digest,
				Active:       release.Active,
				ManifestJson: manifestJSON,
			})
		}
		response.Items = append(response.Items, catalog)
	}
	return response, nil
}

func capabilityManifestJSON(moduleID, name, version string, backend modulemanifest.Backend, permissions []modulemanifest.PermissionDefinition, contributions []modulemanifest.Contribution) ([]byte, error) {
	manifest := modulemanifest.Manifest{
		APIVersion: modulemanifest.APIVersion,
		Kind:       "ModuleRelease",
		Metadata: modulemanifest.Metadata{
			ID:      moduleID,
			Name:    name,
			Version: version,
		},
		Spec: modulemanifest.Spec{
			Backend:       backend,
			Permissions:   permissions,
			Contributions: contributions,
		},
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal capability manifest: %w", err)
	}
	return encoded, nil
}
