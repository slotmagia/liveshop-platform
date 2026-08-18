package http

import (
	"context"

	apichannels "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/notifychannels"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type NotifyChannelsReaderController struct{ service service.NotifyChannels }

func NewNotifyChannelsReader(application service.NotifyChannels) *NotifyChannelsReaderController {
	return &NotifyChannelsReaderController{service: application}
}

func (c *NotifyChannelsReaderController) GetInApp(ctx context.Context, _ *apichannels.GetInAppReq) (*apichannels.GetInAppRes, error) {
	item, err := c.service.GetNotifyInApp(ctx)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apichannels.GetInAppRes(inAppView(item))
	return &response, nil
}

type NotifyChannelsWriterController struct{ service service.NotifyChannels }

func NewNotifyChannelsWriter(application service.NotifyChannels) *NotifyChannelsWriterController {
	return &NotifyChannelsWriterController{service: application}
}

func (c *NotifyChannelsWriterController) UpdateInApp(ctx context.Context, req *apichannels.UpdateInAppReq) (*apichannels.UpdateInAppRes, error) {
	item, err := c.service.ReplaceNotifyInApp(ctx, appmodel.ReplaceNotifyInApp{
		CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: req.Enabled,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apichannels.UpdateInAppRes(inAppView(item))
	return &response, nil
}

func inAppView(item notifymodel.InAppConfig) apichannels.InAppConfig {
	return apichannels.InAppConfig{Driver: item.Driver, Enabled: item.Enabled, Version: item.Version, UpdatedAt: item.UpdatedAt}
}
