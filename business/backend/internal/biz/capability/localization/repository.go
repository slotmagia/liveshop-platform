package localization

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
)

type Repository interface {
	GetConfig(context.Context) (model.Config, error)
	LoadAPIKey(context.Context) (string, error)
	UpsertConfig(context.Context, model.UpsertConfig, string) (model.Config, error)
	ListWorklist(context.Context, string, string) ([]model.WorklistRow, error)
	Publish(context.Context, model.PublishInput, string) (model.PublishResult, error)
	Fill(context.Context, model.FillInput, model.Provider, string, Translator, string) (model.FillResult, error)
	UpsertSource(context.Context, model.SourceSnapshot) error
	ListPublished(context.Context, string, string, int64, int64) ([]model.PublishedText, error)
}

type Translator interface {
	Translate(ctx context.Context, provider model.Provider, apiKey, text, targetLocale string) (string, error)
}
