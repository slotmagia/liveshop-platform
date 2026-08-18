package http

import (
	"context"
	"time"

	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
	"github.com/liveshop-platform/module-platform/internal/common/web"
	apinotify "github.com/liveshop-platform/module-platform/internal/controlplane/notification/api/http/v1/notify"
	"github.com/liveshop-platform/module-platform/internal/controlplane/notification/service"
)

type Controller struct{ service service.Notification }

func New(application service.Notification) *Controller { return &Controller{service: application} }

func (c *Controller) Dispatch(ctx context.Context, req *apinotify.DispatchReq) (*apinotify.DispatchRes, error) {
	var notBefore time.Time
	if req.NotBefore != "" {
		parsed, err := time.Parse(time.RFC3339Nano, req.NotBefore)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, req.NotBefore)
			if err != nil {
				return nil, web.Failure(notifymodel.ErrInvalid)
			}
		}
		notBefore = parsed
	}
	result, err := c.service.Dispatch(ctx, caller(ctx), notifymodel.DispatchInput{
		EventKey: req.EventKey, DeliveryKey: req.DeliveryKey, MerchantID: req.MerchantID, ShopID: req.ShopID,
		Recipients: notifymodel.Recipients{Phone: req.Recipients.Phone, Email: req.Recipients.Email, Subject: req.Recipients.Subject},
		Variables:  req.Variables, NotBefore: notBefore, Locale: req.Locale,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := &apinotify.DispatchRes{Deliveries: make([]apinotify.DeliveryResult, 0, len(result.Deliveries))}
	for _, item := range result.Deliveries {
		response.Deliveries = append(response.Deliveries, apinotify.DeliveryResult{
			DeliveryID: item.DeliveryID, Channel: string(item.Channel), Status: string(item.Status), Deduped: item.Deduped,
		})
	}
	return response, nil
}

func (c *Controller) Get(ctx context.Context, req *apinotify.GetReq) (*apinotify.GetRes, error) {
	item, err := c.service.GetDelivery(ctx, caller(ctx), req.DeliveryID)
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apinotify.GetRes{
		DeliveryID: item.DeliveryID, DeliveryKey: item.DeliveryKey, EventKey: item.EventKey, Channel: string(item.Channel),
		Status: string(item.Status), MerchantID: item.MerchantID, ShopID: item.ShopID, LastError: item.LastError, AttemptCount: item.AttemptCount,
	}, nil
}

func caller(ctx context.Context) notifymodel.Caller {
	subject := authctx.WorkloadSubject(ctx)
	return notifymodel.Caller{ModuleID: notifymodel.ModuleIDFromWorkload(subject, ""), Subject: subject}
}
