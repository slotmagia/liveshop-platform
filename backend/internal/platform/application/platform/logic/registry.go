package logic

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/liveshop-platform/contracts/modulemanifest"
	apiregistry "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/registry"
	"github.com/liveshop-platform/module-platform/internal/platform/common/requestctx"
	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry/module"
)

func (l *Logic) RegisterRelease(ctx context.Context, body []byte) (*apiregistry.RegisterReleaseRes, error) {
	manifest, err := modulemanifest.Decode(body)
	if err != nil {
		return nil, web.Error(http.StatusBadRequest, err)
	}
	digest, err := l.deps.Modules.Register(ctx, manifest)
	if err != nil {
		return nil, web.Error(http.StatusConflict, err)
	}
	return &apiregistry.RegisterReleaseRes{Digest: digest}, nil
}

func (l *Logic) Activate(ctx context.Context, req *apiregistry.ActivateReq) error {
	if err := l.deps.Modules.Activate(ctx, req.ModuleID, req.Version); err != nil {
		return web.Error(http.StatusConflict, err)
	}
	return nil
}

func (l *Logic) Routes(ctx context.Context) (*apiregistry.RoutesRes, error) {
	revision, routes, err := l.deps.Modules.Routes(ctx)
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, errors.New("module registry is unavailable"))
	}
	return &apiregistry.RoutesRes{Revision: revision, Routes: routes}, nil
}

func (l *Logic) Capabilities(ctx context.Context) (*apiregistry.CapabilitiesRes, error) {
	revision, items, err := l.deps.Modules.CapabilityCatalogs(ctx)
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, errors.New("module capability catalog is unavailable"))
	}
	return &apiregistry.CapabilitiesRes{Revision: revision, Items: items}, nil
}

func (l *Logic) Modules(ctx context.Context) (*apiregistry.ModulesRes, error) {
	items, err := l.deps.Modules.Modules(ctx)
	if err != nil {
		return nil, web.Error(http.StatusServiceUnavailable, err)
	}
	result := apiregistry.ModulesRes(items)
	return &result, nil
}

func (l *Logic) AdminActivate(ctx context.Context, req *apiregistry.AdminActivateReq) error {
	if strings.TrimSpace(req.Version) == "" {
		return web.Error(http.StatusBadRequest, errors.New("module version is required"))
	}
	current := requestctx.Identity(ctx)
	actor := platformregistry.AuditActor{Realm: current.Realm, AppID: current.AppID, MerchantID: current.MerchantID, Subject: current.Subject}
	if err := l.deps.Modules.ActivateAudited(ctx, actor, req.ModuleID, req.Version); err != nil {
		return web.Error(http.StatusConflict, err)
	}
	return nil
}

func (l *Logic) AdminDeactivate(ctx context.Context, moduleID string) error {
	if moduleID == "platform" {
		return web.Error(http.StatusForbidden, errors.New("the platform control-plane module cannot deactivate itself"))
	}
	current := requestctx.Identity(ctx)
	actor := platformregistry.AuditActor{Realm: current.Realm, AppID: current.AppID, MerchantID: current.MerchantID, Subject: current.Subject}
	if err := l.deps.Modules.DeactivateAudited(ctx, actor, moduleID); err != nil {
		if errors.Is(err, platformregistry.ErrNotFound) {
			return web.Error(http.StatusNotFound, err)
		}
		return web.Error(http.StatusConflict, err)
	}
	return nil
}
