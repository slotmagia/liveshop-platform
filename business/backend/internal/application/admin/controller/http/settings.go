// Package http adapts the admin surface HTTP contract to its application
// boundary. Read and write capabilities use separate controllers so the router
// can gate them with different permissions.
package http

import (
	"context"

	apisettings "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/settings"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type SettingsReaderController struct{ service service.Settings }

func NewSettingsReader(application service.Settings) *SettingsReaderController {
	return &SettingsReaderController{service: application}
}

func (c *SettingsReaderController) Catalog(_ context.Context, _ *apisettings.CatalogReq) (*apisettings.CatalogRes, error) {
	groups := c.service.SettingCatalog()
	response := make(apisettings.CatalogRes, 0, len(groups))
	for _, group := range groups {
		categories := make([]apisettings.CatalogCategory, 0, len(group.Categories))
		for _, category := range group.Categories {
			fields := make([]apisettings.CatalogField, 0, len(category.Fields))
			for _, field := range category.Fields {
				options := make([]apisettings.CatalogFieldOption, 0, len(field.Options))
				for _, option := range field.Options {
					options = append(options, apisettings.CatalogFieldOption{Value: option.Value, Label: option.Label})
				}
				fields = append(fields, apisettings.CatalogField{Key: field.Key, Label: field.Label, Type: field.Type, Help: field.Help, Placeholder: field.Placeholder, Options: options})
			}
			categories = append(categories, apisettings.CatalogCategory{Key: category.Key, Label: category.Label, GroupKey: category.GroupKey, GroupLabel: category.GroupLabel, Fields: fields})
		}
		response = append(response, apisettings.CatalogGroup{Key: group.Key, Label: group.Label, Categories: categories})
	}
	return &response, nil
}

func (c *SettingsReaderController) List(ctx context.Context, _ *apisettings.ListReq) (*apisettings.ListRes, error) {
	items, err := c.service.ListSettings(ctx)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apisettings.ListRes, 0, len(items))
	for _, item := range items {
		response = append(response, settingDocument(item))
	}
	return &response, nil
}

func (c *SettingsReaderController) Get(ctx context.Context, req *apisettings.GetReq) (*apisettings.GetRes, error) {
	item, err := c.service.GetSetting(ctx, req.Namespace)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisettings.GetRes(settingDocument(item))
	return &response, nil
}

type SettingsWriterController struct{ service service.Settings }

func NewSettingsWriter(application service.Settings) *SettingsWriterController {
	return &SettingsWriterController{service: application}
}

func (c *SettingsWriterController) Put(ctx context.Context, req *apisettings.PutReq) (*apisettings.PutRes, error) {
	item, err := c.service.PutSetting(ctx, appmodel.PutSetting{
		Namespace:       req.Namespace,
		ExpectedVersion: req.ExpectedVersion,
		Value:           req.Value,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisettings.PutRes(settingDocument(item))
	return &response, nil
}

func settingDocument(item model.SettingDocument) apisettings.Document {
	return apisettings.Document{
		Namespace: item.Namespace,
		Value:     item.Value,
		Version:   item.Version,
		UpdatedBy: item.UpdatedBy,
		UpdatedAt: item.UpdatedAt,
	}
}
