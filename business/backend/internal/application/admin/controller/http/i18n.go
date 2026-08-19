package http

import (
	"context"

	apii18n "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/i18n"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type I18nReaderController struct{ service service.I18n }

func NewI18nReader(application service.I18n) *I18nReaderController {
	return &I18nReaderController{service: application}
}

func (c *I18nReaderController) Drivers(_ context.Context, _ *apii18n.DriversReq) (*apii18n.DriversRes, error) {
	return &apii18n.DriversRes{Items: driverViews(c.service.I18nDrivers())}, nil
}

func (c *I18nReaderController) Config(ctx context.Context, _ *apii18n.ConfigGetReq) (*apii18n.ConfigGetRes, error) {
	item, err := c.service.GetI18nConfig(ctx)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apii18n.ConfigGetRes(configView(item))
	return &response, nil
}

func (c *I18nReaderController) Locales(_ context.Context, _ *apii18n.LocalesReq) (*apii18n.LocalesRes, error) {
	items := c.service.I18nLocales()
	response := &apii18n.LocalesRes{Items: make([]apii18n.LocaleView, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, apii18n.LocaleView{Code: item.Code, Label: item.Label})
	}
	return response, nil
}

func (c *I18nReaderController) Entities(_ context.Context, _ *apii18n.EntitiesReq) (*apii18n.EntitiesRes, error) {
	items := c.service.I18nEntities()
	locales := c.service.I18nLocales()
	response := &apii18n.EntitiesRes{Items: make([]apii18n.EntityView, 0, len(items)), Locales: make([]string, 0, len(locales))}
	for _, item := range items {
		response.Items = append(response.Items, apii18n.EntityView{EntityType: item.EntityType, Label: item.Label, OwnerModule: item.OwnerModule, Field: item.Field})
	}
	for _, item := range locales {
		response.Locales = append(response.Locales, item.Code)
	}
	return response, nil
}

func (c *I18nReaderController) Texts(ctx context.Context, req *apii18n.TextsListReq) (*apii18n.TextsListRes, error) {
	items, err := c.service.ListI18nTexts(ctx, req.EntityType, req.Locale)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := &apii18n.TextsListRes{Items: make([]apii18n.TextRow, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, apii18n.TextRow{
			EntityID: item.EntityID, MerchantID: item.MerchantID, ShopID: item.ShopID, Source: item.Source,
			Value: item.Value, Status: item.Status, TextSource: item.TextSource, Stale: item.Stale, Version: item.Version,
		})
	}
	return response, nil
}

type I18nWriterController struct{ service service.I18n }

func NewI18nWriter(application service.I18n) *I18nWriterController {
	return &I18nWriterController{service: application}
}

func (c *I18nWriterController) UpdateConfig(ctx context.Context, req *apii18n.ConfigUpdateReq) (*apii18n.ConfigUpdateRes, error) {
	item, err := c.service.PutI18nConfig(ctx, appmodel.PutI18nConfig{
		CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Provider: req.Provider, APIKey: req.APIKey, APIKeyClear: req.APIKeyClear,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apii18n.ConfigUpdateRes(configView(item))
	return &response, nil
}

func (c *I18nWriterController) Publish(ctx context.Context, req *apii18n.TextsPublishReq) (*apii18n.TextsPublishRes, error) {
	item, err := c.service.PublishI18nText(ctx, appmodel.PublishI18nText{
		CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, EntityType: req.EntityType, EntityID: req.EntityID,
		Locale: req.Locale, Value: req.Value, MerchantID: req.MerchantID, ShopID: req.ShopID,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apii18n.TextsPublishRes{OK: item.OK, Version: item.Version}, nil
}

func (c *I18nWriterController) Fill(ctx context.Context, req *apii18n.TextsFillReq) (*apii18n.TextsFillRes, error) {
	item, err := c.service.FillI18nTexts(ctx, appmodel.FillI18nTexts{CommandKey: req.CommandKey, EntityType: req.EntityType, Locale: req.Locale})
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apii18n.TextsFillRes{Provider: string(item.Provider), Filled: item.Filled, Skipped: item.Skipped}, nil
}

func configView(item model.Config) apii18n.ConfigView {
	return apii18n.ConfigView{Provider: string(item.Provider), APIKeySet: item.APIKeySet, Version: item.Version}
}

func driverViews(definitions []model.DriverDefinition) []apii18n.DriverDefinition {
	response := make([]apii18n.DriverDefinition, 0, len(definitions))
	for _, definition := range definitions {
		fields := make([]apii18n.DriverField, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			fields = append(fields, apii18n.DriverField{Key: field.Key, Label: field.Label, Type: field.Type, Required: field.Required, Secret: field.Secret, Placeholder: field.Placeholder, Help: field.Help})
		}
		response = append(response, apii18n.DriverDefinition{Code: definition.Code, Name: definition.Name, Description: definition.Description, Fields: fields})
	}
	return response
}
