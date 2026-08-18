package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

// settingScope derives the tenant of a settings document from the verified
// identity; a request body can never widen it.
func settingScope(ctx context.Context) model.SettingScope {
	current := authctx.Capability(ctx)
	return model.SettingScope{Realm: current.Realm.String(), MerchantID: current.MerchantID, Subject: current.Subject}
}

func (l *Logic) SettingCatalog() []model.SettingGroup {
	if l.deps.Settings == nil {
		return nil
	}
	return l.deps.Settings.Catalog()
}

func (l *Logic) ListSettings(ctx context.Context) ([]model.SettingDocument, error) {
	if l.deps.Settings == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.Settings.List(ctx, settingScope(ctx))
}

func (l *Logic) GetSetting(ctx context.Context, namespace string) (model.SettingDocument, error) {
	if l.deps.Settings == nil {
		return model.SettingDocument{}, model.ErrUnavailable
	}
	return l.deps.Settings.Get(ctx, settingScope(ctx), namespace)
}

func (l *Logic) PutSetting(ctx context.Context, input appmodel.PutSetting) (model.SettingDocument, error) {
	if l.deps.Settings == nil {
		return model.SettingDocument{}, model.ErrUnavailable
	}
	document, err := l.deps.Settings.Put(ctx, settingScope(ctx), input.Namespace, input.ExpectedVersion, input.Value)
	if err != nil {
		return model.SettingDocument{}, err
	}
	if model.NormalizeNamespace(input.Namespace) == "domain-base" && l.deps.Edge != nil {
		if applyErr := l.deps.Edge.Apply(ctx); applyErr != nil {
			return model.SettingDocument{}, applyErr
		}
	}
	return document, nil
}

func (l *Logic) ListAudit(ctx context.Context, limit int) ([]model.AuditEvent, error) {
	if l.deps.Audit == nil {
		return nil, model.ErrUnavailable
	}
	current := authctx.Capability(ctx)
	return l.deps.Audit.List(ctx, model.AuditScope{Realm: current.Realm.String(), MerchantID: current.MerchantID}, limit)
}
