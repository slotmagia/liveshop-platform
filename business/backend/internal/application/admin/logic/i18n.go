package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	locmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

func i18nScope(ctx context.Context) locmodel.Scope {
	claims := authctx.Capability(ctx)
	return locmodel.Scope{Realm: claims.Realm.String(), MerchantID: claims.MerchantID, Subject: claims.Subject}
}

func (l *Logic) I18nDrivers() []locmodel.DriverDefinition {
	if l.deps.Localization == nil {
		return locmodel.DriverDefinitions()
	}
	return l.deps.Localization.Drivers()
}

func (l *Logic) I18nLocales() []locmodel.Locale {
	if l.deps.Localization == nil {
		return locmodel.PlatformLocales()
	}
	return l.deps.Localization.Locales()
}

func (l *Logic) I18nEntities() []locmodel.Entity {
	if l.deps.Localization == nil {
		return locmodel.Entities()
	}
	return l.deps.Localization.Entities()
}

func (l *Logic) GetI18nConfig(ctx context.Context) (locmodel.Config, error) {
	if l.deps.Localization == nil {
		return locmodel.Config{}, model.ErrUnavailable
	}
	return l.deps.Localization.GetConfig(ctx, i18nScope(ctx))
}

func (l *Logic) PutI18nConfig(ctx context.Context, input appmodel.PutI18nConfig) (locmodel.Config, error) {
	if l.deps.Localization == nil {
		return locmodel.Config{}, model.ErrUnavailable
	}
	return l.deps.Localization.UpsertConfig(ctx, i18nScope(ctx), locmodel.UpsertConfig{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Provider: locmodel.Provider(input.Provider),
		APIKey: input.APIKey, APIKeyClear: input.APIKeyClear,
	})
}

func (l *Logic) ListI18nTexts(ctx context.Context, entityType, locale string) ([]locmodel.WorklistRow, error) {
	if l.deps.Localization == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.Localization.ListWorklist(ctx, i18nScope(ctx), entityType, locale)
}

func (l *Logic) PublishI18nText(ctx context.Context, input appmodel.PublishI18nText) (locmodel.PublishResult, error) {
	if l.deps.Localization == nil {
		return locmodel.PublishResult{}, model.ErrUnavailable
	}
	return l.deps.Localization.Publish(ctx, i18nScope(ctx), locmodel.PublishInput{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, EntityType: input.EntityType,
		EntityID: input.EntityID, Locale: input.Locale, Value: input.Value, MerchantID: input.MerchantID, ShopID: input.ShopID,
	})
}

func (l *Logic) FillI18nTexts(ctx context.Context, input appmodel.FillI18nTexts) (locmodel.FillResult, error) {
	if l.deps.Localization == nil {
		return locmodel.FillResult{}, model.ErrUnavailable
	}
	return l.deps.Localization.Fill(ctx, i18nScope(ctx), locmodel.FillInput{CommandKey: input.CommandKey, EntityType: input.EntityType, Locale: input.Locale})
}
