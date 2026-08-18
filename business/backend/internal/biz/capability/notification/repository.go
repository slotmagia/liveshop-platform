package notification

import (
	"context"
	"time"

	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
)

// Repository performs projection, policy, template library and delivery writes.
type Repository interface {
	Project(context.Context, uint64, []notifymodel.Declaration) error
	ListEvents(context.Context, notifymodel.EventFilter) ([]notifymodel.Event, error)
	GetEvent(context.Context, string) (notifymodel.Event, error)
	ReplacePolicy(context.Context, notifymodel.Scope, notifymodel.ReplacePolicy, string) (notifymodel.Policy, error)

	ListLibraryTemplates(context.Context, notifymodel.TemplateFilter) ([]notifymodel.LibraryTemplate, error)
	GetLibraryTemplate(context.Context, string) (notifymodel.LibraryTemplate, error)
	UpsertLibraryTemplate(context.Context, notifymodel.Scope, notifymodel.UpsertLibraryTemplate, string) (notifymodel.LibraryTemplate, error)
	RetireLibraryTemplate(context.Context, notifymodel.Scope, notifymodel.RetireLibraryTemplate, string) (notifymodel.LibraryTemplate, error)

	GetInAppConfig(context.Context) (notifymodel.InAppConfig, error)
	ReplaceInAppConfig(context.Context, notifymodel.Scope, notifymodel.ReplaceInAppConfig, string) (notifymodel.InAppConfig, error)

	ListDeliveries(context.Context, string) ([]notifymodel.Delivery, error)
	GetDelivery(context.Context, string) (notifymodel.Delivery, error)

	PrepareDeliveries(context.Context, notifymodel.DispatchInput, notifymodel.Event, []notifymodel.Channel, string) ([]notifymodel.DeliveryResult, []notifymodel.Delivery, error)
	MarkSending(context.Context, string) error
	CompleteDelivery(context.Context, string, notifymodel.DeliveryStatus, string, *notifymodel.InboxMessage) error
	ListDue(context.Context, time.Time, int) ([]notifymodel.Delivery, error)
}

type ChannelSender interface {
	SendSMS(ctx context.Context, phone, text string) (notifymodel.ChannelSendResult, error)
	SendEmail(ctx context.Context, to, subject, html string) (notifymodel.ChannelSendResult, error)
}
