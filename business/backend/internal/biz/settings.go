package biz

import (
	"context"
	"encoding/json"

	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

// SettingsRepository stores versioned, non-secret configuration documents. It
// receives a value that the use case already canonicalised and screened.
type SettingsRepository interface {
	List(ctx context.Context, scope model.SettingScope) ([]model.SettingDocument, error)
	Get(ctx context.Context, scope model.SettingScope, namespace string) (model.SettingDocument, error)
	Put(ctx context.Context, scope model.SettingScope, namespace string, expectedVersion int64, canonical []byte) (model.SettingDocument, error)
}

type Settings struct{ repository SettingsRepository }

func NewSettings(repository SettingsRepository) *Settings { return &Settings{repository: repository} }

func (a *Settings) List(ctx context.Context, scope model.SettingScope) ([]model.SettingDocument, error) {
	if !scope.Valid() {
		return nil, model.ErrSettingsInvalid
	}
	return a.repository.List(ctx, scope)
}

func (a *Settings) Get(ctx context.Context, scope model.SettingScope, namespace string) (model.SettingDocument, error) {
	namespace = model.NormalizeNamespace(namespace)
	if !scope.Valid() || !model.ValidNamespace(namespace) {
		return model.SettingDocument{}, model.ErrSettingsInvalid
	}
	return a.repository.Get(ctx, scope, namespace)
}

func (a *Settings) Catalog() []model.SettingGroup {
	return model.SettingCatalog()
}

func (a *Settings) Put(ctx context.Context, scope model.SettingScope, namespace string, expectedVersion int64, value json.RawMessage) (model.SettingDocument, error) {
	namespace = model.NormalizeNamespace(namespace)
	canonical, err := model.CanonicalCatalogValue(namespace, value)
	if !scope.Valid() || !model.ValidNamespace(namespace) || expectedVersion < 0 || err != nil {
		return model.SettingDocument{}, model.ErrSettingsInvalid
	}
	return a.repository.Put(ctx, scope, namespace, expectedVersion, canonical)
}
