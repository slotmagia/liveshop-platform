package telemetry

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
)

// Repository is implemented by the Platform data layer.
type Repository interface {
	InsertIgnore(ctx context.Context, items []model.Event) ([]bool, error)
	List(ctx context.Context, filter model.Filter) (model.Page, error)
}
