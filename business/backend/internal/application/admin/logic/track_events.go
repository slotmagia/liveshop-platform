package logic

import (
	"context"

	telemmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

func (l *Logic) ListTrackEvents(ctx context.Context, filter telemmodel.Filter) (telemmodel.Page, error) {
	if l.deps.Telemetry == nil {
		return telemmodel.Page{}, model.ErrUnavailable
	}
	return l.deps.Telemetry.List(ctx, filter)
}
