package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

func notifyScope(ctx context.Context) notifymodel.Scope {
	claims := authctx.Capability(ctx)
	return notifymodel.Scope{Realm: claims.Realm.String(), MerchantID: claims.MerchantID, Subject: claims.Subject}
}

func (l *Logic) ListNotifyEvents(ctx context.Context, filter notifymodel.EventFilter) ([]notifymodel.Event, error) {
	if l.deps.Notification == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.Notification.ListEvents(ctx, filter)
}

func (l *Logic) GetNotifyEvent(ctx context.Context, eventKey string) (notifymodel.Event, error) {
	if l.deps.Notification == nil {
		return notifymodel.Event{}, model.ErrUnavailable
	}
	item, err := l.deps.Notification.GetEvent(ctx, eventKey)
	if err != nil {
		return notifymodel.Event{}, err
	}
	if !item.Dispatchable {
		return notifymodel.Event{}, notifymodel.ErrNotFound
	}
	return item, nil
}

func (l *Logic) ReplaceNotifyPolicy(ctx context.Context, input appmodel.ReplaceNotifyPolicy) (notifymodel.Event, error) {
	if l.deps.Notification == nil {
		return notifymodel.Event{}, model.ErrUnavailable
	}
	channels := map[notifymodel.Channel]notifymodel.ChannelPolicy{}
	for key, item := range input.Channels {
		channels[notifymodel.Channel(key)] = notifymodel.ChannelPolicy{Enabled: item.Enabled, TemplateCode: item.TemplateCode}
	}
	if _, err := l.deps.Notification.ReplacePolicy(ctx, notifyScope(ctx), notifymodel.ReplacePolicy{
		EventKey: input.EventKey, CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion,
		DispatchMode: notifymodel.DispatchMode(input.DispatchMode), DelaySeconds: input.DelaySeconds, Channels: channels,
	}); err != nil {
		return notifymodel.Event{}, err
	}
	return l.GetNotifyEvent(ctx, input.EventKey)
}

func (l *Logic) ListNotifyDeliveries(ctx context.Context, eventKey string) ([]notifymodel.Delivery, error) {
	if l.deps.Notification == nil {
		return nil, model.ErrUnavailable
	}
	if _, err := l.GetNotifyEvent(ctx, eventKey); err != nil {
		return nil, err
	}
	return l.deps.Notification.ListDeliveries(ctx, eventKey)
}

func (l *Logic) ListNotifyTemplates(ctx context.Context, filter notifymodel.TemplateFilter) ([]notifymodel.LibraryTemplate, error) {
	if l.deps.Notification == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.Notification.ListLibraryTemplates(ctx, filter)
}

func (l *Logic) GetNotifyLibraryTemplate(ctx context.Context, code string) (notifymodel.LibraryTemplate, error) {
	if l.deps.Notification == nil {
		return notifymodel.LibraryTemplate{}, model.ErrUnavailable
	}
	return l.deps.Notification.GetLibraryTemplate(ctx, code)
}

func (l *Logic) UpsertNotifyTemplate(ctx context.Context, input appmodel.UpsertNotifyTemplate) (notifymodel.LibraryTemplate, error) {
	if l.deps.Notification == nil {
		return notifymodel.LibraryTemplate{}, model.ErrUnavailable
	}
	return l.deps.Notification.UpsertLibraryTemplate(ctx, notifyScope(ctx), notifymodel.UpsertLibraryTemplate{
		Code: input.Code, CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Channel: notifymodel.Channel(input.Channel),
		TextTemplate: input.TextTemplate, Subject: input.Subject, BodyHTML: input.BodyHTML, Title: input.Title, Body: input.Body, Variables: input.Variables,
	})
}

func (l *Logic) RetireNotifyTemplate(ctx context.Context, input appmodel.RetireNotifyTemplate) (notifymodel.LibraryTemplate, error) {
	if l.deps.Notification == nil {
		return notifymodel.LibraryTemplate{}, model.ErrUnavailable
	}
	return l.deps.Notification.RetireLibraryTemplate(ctx, notifyScope(ctx), notifymodel.RetireLibraryTemplate{
		Code: input.Code, CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion,
	})
}

func (l *Logic) GetNotifyInApp(ctx context.Context) (notifymodel.InAppConfig, error) {
	if l.deps.Notification == nil {
		return notifymodel.InAppConfig{}, model.ErrUnavailable
	}
	return l.deps.Notification.GetInAppConfig(ctx)
}

func (l *Logic) ReplaceNotifyInApp(ctx context.Context, input appmodel.ReplaceNotifyInApp) (notifymodel.InAppConfig, error) {
	if l.deps.Notification == nil {
		return notifymodel.InAppConfig{}, model.ErrUnavailable
	}
	return l.deps.Notification.ReplaceInAppConfig(ctx, notifyScope(ctx), notifymodel.ReplaceInAppConfig{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Enabled: input.Enabled,
	})
}

func (l *Logic) ProjectNotifications(ctx context.Context) error {
	if l.deps.Notification == nil || l.deps.Release == nil {
		return nil
	}
	revision, items, err := l.deps.Release.ActiveCapabilities(ctx)
	if err != nil {
		return err
	}
	return l.deps.Notification.ProjectCapabilities(ctx, revision, items)
}
