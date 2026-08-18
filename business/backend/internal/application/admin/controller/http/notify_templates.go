package http

import (
	"context"

	apitemplates "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/notifytemplates"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type NotifyTemplatesReaderController struct{ service service.NotifyTemplates }

func NewNotifyTemplatesReader(application service.NotifyTemplates) *NotifyTemplatesReaderController {
	return &NotifyTemplatesReaderController{service: application}
}

func (c *NotifyTemplatesReaderController) List(ctx context.Context, req *apitemplates.ListReq) (*apitemplates.ListRes, error) {
	items, err := c.service.ListNotifyTemplates(ctx, notifymodel.TemplateFilter{Channel: notifymodel.Channel(req.Channel), Keyword: req.Keyword})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apitemplates.ListRes, 0, len(items))
	for _, item := range items {
		response = append(response, templateView(item))
	}
	return &response, nil
}

func (c *NotifyTemplatesReaderController) Get(ctx context.Context, req *apitemplates.GetReq) (*apitemplates.GetRes, error) {
	item, err := c.service.GetNotifyLibraryTemplate(ctx, req.Code)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apitemplates.GetRes(templateView(item))
	return &response, nil
}

type NotifyTemplatesWriterController struct{ service service.NotifyTemplates }

func NewNotifyTemplatesWriter(application service.NotifyTemplates) *NotifyTemplatesWriterController {
	return &NotifyTemplatesWriterController{service: application}
}

func (c *NotifyTemplatesWriterController) Update(ctx context.Context, req *apitemplates.UpdateReq) (*apitemplates.UpdateRes, error) {
	item, err := c.service.UpsertNotifyTemplate(ctx, appmodel.UpsertNotifyTemplate{
		Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Channel: req.Channel,
		TextTemplate: req.TextTemplate, Subject: req.Subject, BodyHTML: req.BodyHTML, Title: req.Title, Body: req.Body, Variables: req.Variables,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apitemplates.UpdateRes(templateView(item))
	return &response, nil
}

func (c *NotifyTemplatesWriterController) Retire(ctx context.Context, req *apitemplates.RetireReq) (*apitemplates.RetireRes, error) {
	item, err := c.service.RetireNotifyTemplate(ctx, appmodel.RetireNotifyTemplate{
		Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apitemplates.RetireRes(templateView(item))
	return &response, nil
}

func templateView(item notifymodel.LibraryTemplate) apitemplates.Template {
	return apitemplates.Template{
		Code: item.Code, Channel: string(item.Channel), TextTemplate: item.TextTemplate, Subject: item.Subject,
		BodyHTML: item.BodyHTML, Title: item.Title, Body: item.Body, Variables: item.Variables, Lifecycle: item.Lifecycle,
		Version: item.Version, UpdatedAt: item.UpdatedAt,
	}
}
