// Package service declares the admin surface application boundary. Each
// capability is a separate interface so a controller only receives the
// operations its route group is authorized to perform.
package service

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
	locmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

type Registry interface {
	Modules(ctx context.Context) ([]model.ModuleInfo, error)
	Capabilities(ctx context.Context) (appmodel.CapabilityCatalog, error)
	Activate(ctx context.Context, activation appmodel.Activation) error
	Deactivate(ctx context.Context, moduleID string) error
}

type Settings interface {
	SettingCatalog() []model.SettingGroup
	ListSettings(ctx context.Context) ([]model.SettingDocument, error)
	GetSetting(ctx context.Context, namespace string) (model.SettingDocument, error)
	PutSetting(ctx context.Context, input appmodel.PutSetting) (model.SettingDocument, error)
}

type Audit interface {
	ListAudit(ctx context.Context, limit int) ([]model.AuditEvent, error)
}

type LiveProvider interface {
	LiveProviderDrivers(context.Context) []providermodel.DriverDefinition
	ListLiveProviders(context.Context, providermodel.Filter) ([]providermodel.Provider, error)
	PutLiveProvider(context.Context, appmodel.PutLiveProvider) (providermodel.Provider, error)
	RetireLiveProvider(context.Context, appmodel.RetireLiveProvider) (providermodel.Provider, error)
	GetLiveProviderAssignments(context.Context, int64) (providermodel.AssignmentSet, error)
	PutLiveProviderAssignments(context.Context, appmodel.PutLiveProviderAssignments) (providermodel.AssignmentSet, error)
}

type SMS interface {
	SMSDrivers(context.Context) []smsmodel.DriverDefinition
	ListSMSChannels(context.Context, smsmodel.ChannelFilter) ([]smsmodel.Channel, error)
	PutSMSChannel(context.Context, appmodel.PutSMSChannel) (smsmodel.Channel, error)
	SetSMSChannelEnabled(context.Context, appmodel.SetSMSEnabled) (smsmodel.Channel, error)
	RetireSMSChannel(context.Context, appmodel.RetireSMS) (smsmodel.Channel, error)
	ListSMSRegions(context.Context, smsmodel.RegionFilter) ([]smsmodel.Region, error)
	PutSMSRegion(context.Context, appmodel.PutSMSRegion) (smsmodel.Region, error)
	SetSMSRegionEnabled(context.Context, appmodel.SetSMSEnabled) (smsmodel.Region, error)
	RetireSMSRegion(context.Context, appmodel.RetireSMS) (smsmodel.Region, error)
	GetSMSMerchantGrant(context.Context, int64, int64) (smsmodel.MerchantGrant, error)
	PutSMSMerchantGrant(context.Context, appmodel.PutSMSMerchantGrant) (smsmodel.MerchantGrant, error)
	TestSMSSend(context.Context, appmodel.TestSMSSend) (smsmodel.TestSendResult, error)
}

type Email interface {
	EmailDrivers(context.Context) []emailmodel.DriverDefinition
	GetEmailConfig(context.Context) (emailmodel.Config, error)
	PutEmailConfig(context.Context, appmodel.PutEmailConfig) (emailmodel.Config, error)
	SetEmailEnabled(context.Context, appmodel.SetEmailEnabled) (emailmodel.Config, error)
	TestEmailSend(context.Context, appmodel.TestEmailSend) (emailmodel.TestSendResult, error)
}

type Storage interface {
	StorageDrivers(context.Context) []storagemodel.DriverDefinition
	ListStorageChannels(context.Context, storagemodel.ChannelFilter) ([]storagemodel.Channel, error)
	PutStorageChannel(context.Context, appmodel.PutStorageChannel) (storagemodel.Channel, error)
	SetStorageChannelEnabled(context.Context, appmodel.SetStorageEnabled) (storagemodel.Channel, error)
	SetStorageDefault(context.Context, appmodel.SetStorageDefault) (storagemodel.Channel, error)
	RetireStorageChannel(context.Context, appmodel.RetireStorage) (storagemodel.Channel, error)
	TestStorageChannel(context.Context, appmodel.TestStorageChannel) (storagemodel.TestResult, error)
	GetStorageObject(context.Context, string) (storagemodel.Object, error)
}

type NotifyEvents interface {
	ListNotifyEvents(context.Context, notifymodel.EventFilter) ([]notifymodel.Event, error)
	GetNotifyEvent(context.Context, string) (notifymodel.Event, error)
	ReplaceNotifyPolicy(context.Context, appmodel.ReplaceNotifyPolicy) (notifymodel.Event, error)
	ListNotifyDeliveries(context.Context, string) ([]notifymodel.Delivery, error)
}

type NotifyTemplates interface {
	ListNotifyTemplates(context.Context, notifymodel.TemplateFilter) ([]notifymodel.LibraryTemplate, error)
	GetNotifyLibraryTemplate(context.Context, string) (notifymodel.LibraryTemplate, error)
	UpsertNotifyTemplate(context.Context, appmodel.UpsertNotifyTemplate) (notifymodel.LibraryTemplate, error)
	RetireNotifyTemplate(context.Context, appmodel.RetireNotifyTemplate) (notifymodel.LibraryTemplate, error)
}

type NotifyChannels interface {
	GetNotifyInApp(context.Context) (notifymodel.InAppConfig, error)
	ReplaceNotifyInApp(context.Context, appmodel.ReplaceNotifyInApp) (notifymodel.InAppConfig, error)
}

type I18n interface {
	I18nDrivers() []locmodel.DriverDefinition
	I18nLocales() []locmodel.Locale
	I18nEntities() []locmodel.Entity
	GetI18nConfig(context.Context) (locmodel.Config, error)
	PutI18nConfig(context.Context, appmodel.PutI18nConfig) (locmodel.Config, error)
	ListI18nTexts(context.Context, string, string) ([]locmodel.WorklistRow, error)
	PublishI18nText(context.Context, appmodel.PublishI18nText) (locmodel.PublishResult, error)
	FillI18nTexts(context.Context, appmodel.FillI18nTexts) (locmodel.FillResult, error)
}
