package service

import (
	"context"

	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
)

type Notification interface {
	Dispatch(context.Context, notifymodel.Caller, notifymodel.DispatchInput) (notifymodel.DispatchResult, error)
	GetDelivery(context.Context, notifymodel.Caller, string) (notifymodel.Delivery, error)
}
