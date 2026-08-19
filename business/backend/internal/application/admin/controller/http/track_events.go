package http

import (
	"context"

	apitrack "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/trackevents"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	telemmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type TrackEventsReaderController struct{ service service.TrackEvents }

func NewTrackEventsReader(application service.TrackEvents) *TrackEventsReaderController {
	return &TrackEventsReaderController{service: application}
}

func (c *TrackEventsReaderController) List(ctx context.Context, req *apitrack.ListReq) (*apitrack.ListRes, error) {
	page, err := c.service.ListTrackEvents(ctx, telemmodel.Filter{
		MerchantID: req.MerchantID, ShopID: req.ShopID, Surface: req.Surface, EventName: req.EventName,
		EventType: req.EventType, Subject: req.Subject, AnonymousID: req.AnonymousID,
		StartMs: req.StartMs, EndMs: req.EndMs, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	items := make([]apitrack.Event, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, trackEventView(item))
	}
	return &apitrack.ListRes{Items: items, Total: page.Total}, nil
}

func trackEventView(item telemmodel.Event) apitrack.Event {
	return apitrack.Event{
		MerchantID: item.MerchantID, ShopID: item.ShopID, Surface: item.Surface, EventID: item.EventID,
		EventType: item.EventType, EventName: item.EventName, Page: item.Page, Component: item.Component,
		Action: item.Action, BizType: item.BizType, BizID: item.BizID, SessionID: item.SessionID,
		AnonymousID: item.AnonymousID, Subject: item.Subject, ClientTs: item.ClientTs,
		OccurredAt: item.OccurredAt, ReceivedAt: item.ReceivedAt, SchemaVersion: item.SchemaVersion,
		LiveContext: item.LiveContext, Props: item.Props, State: item.State, Extra: item.Extra,
		UserAgent: item.UserAgent, IP: item.IP, Referer: item.Referer, AdTouchID: item.AdTouchID,
		ClickIDType: item.ClickIDType, CreatedAt: item.CreatedAt,
	}
}
