package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

func (l *Logic) Modules(ctx context.Context) ([]model.ModuleInfo, error) {
	if l.deps.Release == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.Release.Modules(ctx)
}

func (l *Logic) Capabilities(ctx context.Context) (appmodel.CapabilityCatalog, error) {
	if l.deps.Release == nil {
		return appmodel.CapabilityCatalog{}, model.ErrUnavailable
	}
	revision, items, err := l.deps.Release.Capabilities(ctx)
	if err != nil {
		return appmodel.CapabilityCatalog{}, err
	}
	return appmodel.CapabilityCatalog{Revision: revision, Items: items}, nil
}

func (l *Logic) Activate(ctx context.Context, activation appmodel.Activation) error {
	if l.deps.Release == nil {
		return model.ErrUnavailable
	}
	if err := l.deps.Release.ActivateAudited(ctx, authctx.RegistryActor(ctx), activation.ModuleID, activation.Version); err != nil {
		return err
	}
	_ = l.ProjectNotifications(ctx)
	return nil
}

func (l *Logic) Deactivate(ctx context.Context, moduleID string) error {
	if l.deps.Release == nil {
		return model.ErrUnavailable
	}
	if err := l.deps.Release.DeactivateAudited(ctx, authctx.RegistryActor(ctx), moduleID); err != nil {
		return err
	}
	_ = l.ProjectNotifications(ctx)
	return nil
}
