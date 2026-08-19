package localization

import (
	"context"
	"strings"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
)

type UseCase struct {
	repository Repository
	translator Translator
}

func New(repository Repository, translator Translator) *UseCase {
	if translator == nil {
		translator = NoopTranslator{}
	}
	return &UseCase{repository: repository, translator: translator}
}

func (u *UseCase) Drivers() []model.DriverDefinition { return model.DriverDefinitions() }

func (u *UseCase) Locales() []model.Locale { return model.PlatformLocales() }

func (u *UseCase) Entities() []model.Entity { return model.Entities() }

func (u *UseCase) GetConfig(ctx context.Context, scope model.Scope) (model.Config, error) {
	if !scope.Valid() {
		return model.Config{}, model.ErrInvalid
	}
	if u.repository == nil {
		return model.Config{}, model.ErrUnavailable
	}
	return u.repository.GetConfig(ctx)
}

func (u *UseCase) UpsertConfig(ctx context.Context, scope model.Scope, input model.UpsertConfig) (model.Config, error) {
	if !scope.Valid() || !model.ValidCommandKey(input.CommandKey) || !model.ValidProvider(input.Provider) {
		return model.Config{}, model.ErrInvalid
	}
	if input.APIKeyClear && strings.TrimSpace(input.APIKey) != "" {
		return model.Config{}, model.ErrInvalid
	}
	if u.repository == nil {
		return model.Config{}, model.ErrUnavailable
	}
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.APIKey = strings.TrimSpace(input.APIKey)
	return u.repository.UpsertConfig(ctx, input, model.RequestHash(input))
}

func (u *UseCase) ListWorklist(ctx context.Context, scope model.Scope, entityType, locale string) ([]model.WorklistRow, error) {
	if !scope.Valid() {
		return nil, model.ErrInvalid
	}
	entityType = strings.TrimSpace(entityType)
	locale = strings.TrimSpace(locale)
	if !model.KnownEntity(entityType) {
		return nil, model.ErrEntityUnknown
	}
	if !model.IsTargetLocale(locale) {
		return nil, model.ErrLocaleUnknown
	}
	if u.repository == nil {
		return nil, model.ErrUnavailable
	}
	return u.repository.ListWorklist(ctx, entityType, locale)
}

func (u *UseCase) Publish(ctx context.Context, scope model.Scope, input model.PublishInput) (model.PublishResult, error) {
	if !scope.Valid() || !model.ValidCommandKey(input.CommandKey) {
		return model.PublishResult{}, model.ErrInvalid
	}
	input.EntityType = strings.TrimSpace(input.EntityType)
	input.EntityID = strings.TrimSpace(input.EntityID)
	input.Locale = strings.TrimSpace(input.Locale)
	input.Value = strings.TrimSpace(input.Value)
	if !model.KnownEntity(input.EntityType) {
		return model.PublishResult{}, model.ErrEntityUnknown
	}
	if !model.IsTargetLocale(input.Locale) || input.EntityID == "" || input.Value == "" {
		return model.PublishResult{}, model.ErrInvalid
	}
	if u.repository == nil {
		return model.PublishResult{}, model.ErrUnavailable
	}
	return u.repository.Publish(ctx, input, model.RequestHash(input))
}

func (u *UseCase) Fill(ctx context.Context, scope model.Scope, input model.FillInput) (model.FillResult, error) {
	if !scope.Valid() || !model.ValidCommandKey(input.CommandKey) {
		return model.FillResult{}, model.ErrInvalid
	}
	input.EntityType = strings.TrimSpace(input.EntityType)
	input.Locale = strings.TrimSpace(input.Locale)
	if !model.KnownEntity(input.EntityType) {
		return model.FillResult{}, model.ErrEntityUnknown
	}
	if !model.IsTargetLocale(input.Locale) {
		return model.FillResult{}, model.ErrLocaleUnknown
	}
	if u.repository == nil {
		return model.FillResult{}, model.ErrUnavailable
	}
	config, err := u.repository.GetConfig(ctx)
	if err != nil {
		return model.FillResult{}, err
	}
	if config.Provider != model.ProviderNoop && !config.APIKeySet {
		return model.FillResult{}, model.ErrProviderKey
	}
	apiKey, err := u.repository.LoadAPIKey(ctx)
	if err != nil {
		return model.FillResult{}, err
	}
	return u.repository.Fill(ctx, input, config.Provider, apiKey, u.translator, model.RequestHash(input))
}

func (u *UseCase) UpsertSource(ctx context.Context, snapshot model.SourceSnapshot) error {
	snapshot.EntityType = strings.TrimSpace(snapshot.EntityType)
	snapshot.EntityID = strings.TrimSpace(snapshot.EntityID)
	snapshot.Source = strings.TrimSpace(snapshot.Source)
	if !model.KnownEntity(snapshot.EntityType) || snapshot.EntityID == "" || snapshot.Source == "" || snapshot.MerchantID < 0 || snapshot.ShopID < 0 {
		return model.ErrInvalid
	}
	if u.repository == nil {
		return model.ErrUnavailable
	}
	return u.repository.UpsertSource(ctx, snapshot)
}

func (u *UseCase) ListPublished(ctx context.Context, entityType, locale string, merchantID, shopID int64) ([]model.PublishedText, error) {
	entityType = strings.TrimSpace(entityType)
	locale = strings.TrimSpace(locale)
	if !model.KnownEntity(entityType) {
		return nil, model.ErrEntityUnknown
	}
	if !model.IsTargetLocale(locale) {
		return nil, model.ErrLocaleUnknown
	}
	if merchantID < 0 || shopID < 0 {
		return nil, model.ErrInvalid
	}
	if u.repository == nil {
		return nil, model.ErrUnavailable
	}
	return u.repository.ListPublished(ctx, entityType, locale, merchantID, shopID)
}
