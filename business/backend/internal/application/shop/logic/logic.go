package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/shop/service"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry"
	telemmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

type Deps struct {
	Telemetry *telemetry.UseCase
}

type Logic struct{ deps Deps }

var _ service.TrackEvents = (*Logic)(nil)

func New(deps Deps) *Logic { return &Logic{deps: deps} }

func (l *Logic) CreateTrackEvents(ctx context.Context, extra telemmodel.Scope, events []telemmodel.EventInput) (telemmodel.IngestResult, error) {
	if l.deps.Telemetry == nil {
		return telemmodel.IngestResult{}, model.ErrUnavailable
	}
	claims := authctx.Capability(ctx)
	scope := extra
	scope.MerchantID = claims.MerchantID
	scope.ShopID = claims.ShopID
	scope.Surface = telemmodel.SurfaceShop
	scope.Subject = claims.Subject
	return l.deps.Telemetry.Ingest(ctx, scope, events)
}
