package http

import (
	"context"

	apinotify "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/notifyevents"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type NotifyEventsReaderController struct{ service service.NotifyEvents }

func NewNotifyEventsReader(application service.NotifyEvents) *NotifyEventsReaderController {
	return &NotifyEventsReaderController{service: application}
}

func (c *NotifyEventsReaderController) List(ctx context.Context, req *apinotify.ListReq) (*apinotify.ListRes, error) {
	items, err := c.service.ListNotifyEvents(ctx, notifymodel.EventFilter{Module: req.Module, Channel: notifymodel.Channel(req.Channel), Keyword: req.Keyword})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apinotify.ListRes, 0, len(items))
	for _, item := range items {
		response = append(response, eventView(item))
	}
	return &response, nil
}

func (c *NotifyEventsReaderController) Get(ctx context.Context, req *apinotify.GetReq) (*apinotify.GetRes, error) {
	item, err := c.service.GetNotifyEvent(ctx, req.EventKey)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apinotify.GetRes(eventView(item))
	return &response, nil
}

func (c *NotifyEventsReaderController) ListDeliveries(ctx context.Context, req *apinotify.ListDeliveriesReq) (*apinotify.ListDeliveriesRes, error) {
	items, err := c.service.ListNotifyDeliveries(ctx, req.EventKey)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apinotify.ListDeliveriesRes, 0, len(items))
	for _, item := range items {
		response = append(response, apinotify.Delivery{
			DeliveryID: item.DeliveryID, DeliveryKey: item.DeliveryKey, Channel: string(item.Channel), Status: string(item.Status),
			Recipient: item.Recipient, LastError: item.LastError, AttemptCount: item.AttemptCount, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return &response, nil
}

type NotifyEventsWriterController struct{ service service.NotifyEvents }

func NewNotifyEventsWriter(application service.NotifyEvents) *NotifyEventsWriterController {
	return &NotifyEventsWriterController{service: application}
}

func (c *NotifyEventsWriterController) ReplacePolicy(ctx context.Context, req *apinotify.ReplacePolicyReq) (*apinotify.ReplacePolicyRes, error) {
	channels := map[string]appmodel.NotifyChannelPolicy{}
	for key, item := range req.Channels {
		channels[key] = appmodel.NotifyChannelPolicy{Enabled: item.Enabled, TemplateCode: item.TemplateCode}
	}
	item, err := c.service.ReplaceNotifyPolicy(ctx, appmodel.ReplaceNotifyPolicy{
		EventKey: req.EventKey, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion,
		DispatchMode: req.DispatchMode, DelaySeconds: req.DelaySeconds, Channels: channels,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apinotify.ReplacePolicyRes(eventView(item))
	return &response, nil
}

func eventView(item notifymodel.Event) apinotify.Event {
	channels := map[string]apinotify.ChannelPolicy{}
	allowed := make([]string, 0, len(item.AllowedChannels))
	for _, channel := range item.AllowedChannels {
		allowed = append(allowed, string(channel))
		policy := item.Policy.Channels[channel]
		channels[string(channel)] = apinotify.ChannelPolicy{Enabled: policy.Enabled, TemplateCode: policy.TemplateCode}
	}
	return apinotify.Event{
		EventKey: item.EventKey, ModuleID: item.ModuleID, ModuleName: item.ModuleName, OperationID: item.OperationID, Title: item.Title,
		Variables: item.Variables, AllowedChannels: allowed, DefaultDispatch: string(item.DefaultDispatch),
		DispatchMode: string(item.Policy.DispatchMode), DelaySeconds: item.Policy.DelaySeconds, Channels: channels,
		PolicyVersion: item.Policy.Version, UpdatedAt: item.UpdatedAt,
	}
}
