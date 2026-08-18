package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/notification"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	"github.com/liveshop-platform/module-platform/internal/controlplane/notification/service"
)

type Logic struct{ notify *notification.UseCase }

var _ service.Notification = (*Logic)(nil)

func New(notify *notification.UseCase) *Logic { return &Logic{notify: notify} }

func (l *Logic) Dispatch(ctx context.Context, caller notifymodel.Caller, input notifymodel.DispatchInput) (notifymodel.DispatchResult, error) {
	if l.notify == nil {
		return notifymodel.DispatchResult{}, notifymodel.ErrInvalid
	}
	return l.notify.Dispatch(ctx, caller, input)
}

func (l *Logic) GetDelivery(ctx context.Context, caller notifymodel.Caller, deliveryID string) (notifymodel.Delivery, error) {
	if l.notify == nil {
		return notifymodel.Delivery{}, notifymodel.ErrInvalid
	}
	return l.notify.GetDelivery(ctx, caller, deliveryID)
}
