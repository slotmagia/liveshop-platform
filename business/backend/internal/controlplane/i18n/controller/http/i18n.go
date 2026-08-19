package http

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
	apii18n "github.com/liveshop-platform/module-platform/internal/controlplane/i18n/api/http/v1/i18n"
)

type Reader struct{ use *localization.UseCase }

func NewReader(use *localization.UseCase) *Reader { return &Reader{use: use} }

func (c *Reader) Texts(ctx context.Context, req *apii18n.TextsListReq) (*apii18n.TextsListRes, error) {
	items, err := c.use.ListPublished(ctx, req.EntityType, req.Locale, req.MerchantID, req.ShopID)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := &apii18n.TextsListRes{Items: make([]apii18n.PublishedText, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, apii18n.PublishedText{EntityID: item.EntityID, Value: item.Value, Version: item.Version})
	}
	return response, nil
}

func (c *Reader) Locales(_ context.Context, _ *apii18n.LocalesReq) (*apii18n.LocalesRes, error) {
	items := c.use.Locales()
	response := &apii18n.LocalesRes{Items: make([]apii18n.LocaleView, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, apii18n.LocaleView{Code: item.Code, Label: item.Label})
	}
	return response, nil
}

func (c *Reader) Entities(_ context.Context, _ *apii18n.EntitiesReq) (*apii18n.EntitiesRes, error) {
	items := c.use.Entities()
	response := &apii18n.EntitiesRes{Items: make([]apii18n.EntityView, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, apii18n.EntityView{EntityType: item.EntityType, Label: item.Label, OwnerModule: item.OwnerModule, Field: item.Field})
	}
	return response, nil
}

type Writer struct{ use *localization.UseCase }

func NewWriter(use *localization.UseCase) *Writer { return &Writer{use: use} }

func (c *Writer) Sources(ctx context.Context, req *apii18n.SourcesUpdateReq) (*apii18n.SourcesUpdateRes, error) {
	if err := c.use.UpsertSource(ctx, model.SourceSnapshot{
		EntityType: req.EntityType, EntityID: req.EntityID, MerchantID: req.MerchantID, ShopID: req.ShopID,
		Source: req.Source, SourceVersion: req.SourceVersion,
	}); err != nil {
		return nil, web.Failure(err)
	}
	return &apii18n.SourcesUpdateRes{OK: true}, nil
}
