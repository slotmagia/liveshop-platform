// Package logic implements the provisioning surface boundary.
package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning/appmodel"
	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning/service"
)

type Logic struct{ release *biz.Release }

var _ service.Provisioning = (*Logic)(nil)

func New(release *biz.Release) *Logic { return &Logic{release: release} }

func (l *Logic) RegisterRelease(ctx context.Context, document []byte) (appmodel.RegisteredRelease, error) {
	if l.release == nil {
		return appmodel.RegisteredRelease{}, model.ErrUnavailable
	}
	digest, err := l.release.RegisterManifest(ctx, document)
	if err != nil {
		return appmodel.RegisteredRelease{}, err
	}
	return appmodel.RegisteredRelease{Digest: digest}, nil
}

func (l *Logic) Activate(ctx context.Context, activation appmodel.Activation) error {
	if l.release == nil {
		return model.ErrUnavailable
	}
	return l.release.Activate(ctx, activation.ModuleID, activation.Version)
}

func (l *Logic) Routes(ctx context.Context) (appmodel.RouteSnapshot, error) {
	if l.release == nil {
		return appmodel.RouteSnapshot{}, model.ErrUnavailable
	}
	revision, routes, err := l.release.Routes(ctx)
	if err != nil {
		return appmodel.RouteSnapshot{}, err
	}
	return appmodel.RouteSnapshot{Revision: revision, Routes: routes}, nil
}

func (l *Logic) ActiveCapabilities(ctx context.Context) (appmodel.ActiveCapabilitySnapshot, error) {
	if l.release == nil {
		return appmodel.ActiveCapabilitySnapshot{}, model.ErrUnavailable
	}
	revision, items, err := l.release.ActiveCapabilities(ctx)
	if err != nil {
		return appmodel.ActiveCapabilitySnapshot{}, err
	}
	return appmodel.ActiveCapabilitySnapshot{RegistryRevision: revision, Modules: items}, nil
}
