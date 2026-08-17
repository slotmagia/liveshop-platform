package sms

import (
	"context"

	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
)

// Repository performs each SMS command, immutable version snapshot and audit
// write in one database transaction. CommandKey reuse with a different hash is
// a conflict.
type Repository interface {
	ListChannels(context.Context, smsmodel.Scope, smsmodel.ChannelFilter) ([]smsmodel.Channel, error)
	UpsertChannel(context.Context, smsmodel.Scope, smsmodel.UpsertChannel, string) (smsmodel.Channel, error)
	SetChannelEnabled(context.Context, smsmodel.Scope, smsmodel.SetEnabled, string) (smsmodel.Channel, error)
	RetireChannel(context.Context, smsmodel.Scope, smsmodel.Retire, string) (smsmodel.Channel, error)
	LoadChannelSecrets(context.Context, smsmodel.Scope, string) (smsmodel.ChannelSecrets, error)

	ListRegions(context.Context, smsmodel.Scope, smsmodel.RegionFilter) ([]smsmodel.Region, error)
	UpsertRegion(context.Context, smsmodel.Scope, smsmodel.UpsertRegion, string) (smsmodel.Region, error)
	SetRegionEnabled(context.Context, smsmodel.Scope, smsmodel.SetEnabled, string) (smsmodel.Region, error)
	RetireRegion(context.Context, smsmodel.Scope, smsmodel.Retire, string) (smsmodel.Region, error)

	GetMerchantGrant(context.Context, smsmodel.Scope, int64, int64) (smsmodel.MerchantGrant, error)
	PutMerchantGrant(context.Context, smsmodel.Scope, smsmodel.PutMerchantGrant, string) (smsmodel.MerchantGrant, error)
}

// Sender delivers one test message through a concrete driver.
type Sender interface {
	Send(context.Context, smsmodel.Driver, map[string]string, string, string) (string, error)
}
