package http

import (
	"context"
	"strconv"

	"github.com/gogf/gf/v2/net/ghttp"
	apitrack "github.com/liveshop-platform/module-platform/internal/application/shop/api/http/v1/trackevents"
	"github.com/liveshop-platform/module-platform/internal/application/shop/service"
	telemmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type TrackEventsWriterController struct{ service service.TrackEvents }

func NewTrackEventsWriter(application service.TrackEvents) *TrackEventsWriterController {
	return &TrackEventsWriterController{service: application}
}

func (c *TrackEventsWriterController) Create(ctx context.Context, req *apitrack.CreateReq) (*apitrack.CreateRes, error) {
	scope := telemmodel.Scope{BodyBytes: int64(len(web.RawBody(ctx)))}
	if request := ghttp.RequestFromCtx(ctx); request != nil {
		scope.UserAgent = request.UserAgent()
		scope.IP = request.GetClientIp()
		scope.Referer = request.Header.Get("Referer")
		scope.ClickIDType = request.Header.Get("X-Ad-Touch-Type")
		if raw := request.Header.Get("X-Ad-Touch-Id"); raw != "" {
			if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
				scope.AdTouchID = id
			}
		}
	}
	events := make([]telemmodel.EventInput, 0, len(req.Events))
	for _, item := range req.Events {
		events = append(events, telemmodel.EventInput{
			EventID: item.EventID, EventName: item.EventName, EventType: item.EventType,
			Page: item.Page, Component: item.Component, Action: item.Action, BizType: item.BizType, BizID: item.BizID,
			SessionID: item.SessionID, AnonymousID: item.AnonymousID, OccurredAtMs: item.OccurredAtMs, ClientTs: item.ClientTs,
			SchemaVersion: item.SchemaVersion, LiveContext: item.LiveContext, Props: item.Props, State: item.State, Extra: item.Extra,
			MerchantID: item.MerchantID, ShopID: item.ShopID, App: item.App, AppID: item.AppID, CommercialID: item.CommercialID, UID: item.UID,
		})
	}
	result, err := c.service.CreateTrackEvents(ctx, scope, events)
	if err != nil {
		return nil, web.Failure(err)
	}
	errors := make([]apitrack.ItemError, 0, len(result.Errors))
	for _, item := range result.Errors {
		errors = append(errors, apitrack.ItemError{Index: item.Index, EventID: item.EventID, Code: item.Code, Message: item.Message})
	}
	return &apitrack.CreateRes{Accepted: result.Accepted, Duplicates: result.Duplicates, Rejected: result.Rejected, Errors: errors}, nil
}
