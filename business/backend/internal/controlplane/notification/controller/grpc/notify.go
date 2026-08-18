package grpc

import (
	"context"
	"errors"
	"time"

	platformv1 "github.com/liveshop-platform/contracts/gen/go/platform/v1"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	"github.com/liveshop-platform/module-platform/internal/common/grpcauth"
	"github.com/liveshop-platform/module-platform/internal/controlplane/notification/service"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Controller struct {
	platformv1.UnimplementedPlatformNotificationServiceServer
	service service.Notification
}

func Register(server grpclib.ServiceRegistrar, application service.Notification) {
	platformv1.RegisterPlatformNotificationServiceServer(server, &Controller{service: application})
}

func (c *Controller) Dispatch(ctx context.Context, req *platformv1.DispatchRequest) (*platformv1.DispatchResponse, error) {
	var notBefore time.Time
	if req.GetNotBeforeUnixMs() > 0 {
		notBefore = time.UnixMilli(req.GetNotBeforeUnixMs()).UTC()
	}
	recipients := notifymodel.Recipients{}
	if req.GetRecipients() != nil {
		recipients = notifymodel.Recipients{Phone: req.GetRecipients().GetPhone(), Email: req.GetRecipients().GetEmail(), Subject: req.GetRecipients().GetSubject()}
	}
	result, err := c.service.Dispatch(ctx, caller(ctx), notifymodel.DispatchInput{
		EventKey: req.GetEventKey(), DeliveryKey: req.GetDeliveryKey(), MerchantID: req.GetMerchantId(), ShopID: req.GetShopId(),
		Recipients: recipients, Variables: req.GetVariables(), NotBefore: notBefore, Locale: req.GetLocale(),
	})
	if err != nil {
		return nil, failure(err)
	}
	response := &platformv1.DispatchResponse{}
	for _, item := range result.Deliveries {
		response.Deliveries = append(response.Deliveries, &platformv1.NotificationDeliveryResult{
			DeliveryId: item.DeliveryID, Channel: string(item.Channel), Status: string(item.Status), Deduped: item.Deduped,
		})
	}
	return response, nil
}

func (c *Controller) GetDelivery(ctx context.Context, req *platformv1.GetDeliveryRequest) (*platformv1.GetDeliveryResponse, error) {
	item, err := c.service.GetDelivery(ctx, caller(ctx), req.GetDeliveryId())
	if err != nil {
		return nil, failure(err)
	}
	response := &platformv1.GetDeliveryResponse{
		DeliveryId: item.DeliveryID, DeliveryKey: item.DeliveryKey, EventKey: item.EventKey, Channel: string(item.Channel),
		Status: string(item.Status), MerchantId: item.MerchantID, ShopId: item.ShopID, LastError: item.LastError, AttemptCount: int32(item.AttemptCount),
	}
	if !item.NotBefore.IsZero() {
		response.NotBeforeUnixMs = item.NotBefore.UnixMilli()
	}
	return response, nil
}

func caller(ctx context.Context) notifymodel.Caller {
	subject := grpcauth.Subject(ctx)
	return notifymodel.Caller{ModuleID: notifymodel.ModuleIDFromWorkload(subject, ""), Subject: subject}
}

func failure(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, notifymodel.ErrInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, notifymodel.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, notifymodel.ErrConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, notifymodel.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "platform notification request failed")
	}
}
