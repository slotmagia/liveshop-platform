package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

func smsScope(ctx context.Context) smsmodel.Scope {
	claims := authctx.Capability(ctx)
	scope := smsmodel.Scope{Realm: claims.Realm.String(), MerchantID: claims.MerchantID, Subject: claims.Subject}
	if scope.Valid() {
		return scope
	}
	return smsmodel.Scope{Realm: "PLATFORM", Subject: "identity-compose"}
}

func (l *Logic) SMSDrivers(context.Context) []smsmodel.DriverDefinition {
	return smsmodel.DriverDefinitions()
}

func (l *Logic) ListSMSChannels(ctx context.Context, filter smsmodel.ChannelFilter) ([]smsmodel.Channel, error) {
	if l.deps.SMS == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.SMS.ListChannels(ctx, smsScope(ctx), filter)
}

func (l *Logic) PutSMSChannel(ctx context.Context, input appmodel.PutSMSChannel) (smsmodel.Channel, error) {
	if l.deps.SMS == nil {
		return smsmodel.Channel{}, model.ErrUnavailable
	}
	return l.deps.SMS.UpsertChannel(ctx, smsScope(ctx), smsmodel.UpsertChannel{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code, Name: input.Name,
		Driver: input.Driver, Region: input.Region, Priority: input.Priority, PublicConfig: input.PublicConfig, Secrets: input.Secrets,
	})
}

func (l *Logic) SetSMSChannelEnabled(ctx context.Context, input appmodel.SetSMSEnabled) (smsmodel.Channel, error) {
	if l.deps.SMS == nil {
		return smsmodel.Channel{}, model.ErrUnavailable
	}
	return l.deps.SMS.SetChannelEnabled(ctx, smsScope(ctx), smsmodel.SetEnabled{CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code, Enabled: input.Enabled})
}

func (l *Logic) RetireSMSChannel(ctx context.Context, input appmodel.RetireSMS) (smsmodel.Channel, error) {
	if l.deps.SMS == nil {
		return smsmodel.Channel{}, model.ErrUnavailable
	}
	return l.deps.SMS.RetireChannel(ctx, smsScope(ctx), smsmodel.Retire{CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code})
}

func (l *Logic) ListSMSRegions(ctx context.Context, filter smsmodel.RegionFilter) ([]smsmodel.Region, error) {
	if l.deps.SMS == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.SMS.ListRegions(ctx, smsScope(ctx), filter)
}

func (l *Logic) PutSMSRegion(ctx context.Context, input appmodel.PutSMSRegion) (smsmodel.Region, error) {
	if l.deps.SMS == nil {
		return smsmodel.Region{}, model.ErrUnavailable
	}
	return l.deps.SMS.UpsertRegion(ctx, smsScope(ctx), smsmodel.UpsertRegion{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code, DialCode: input.DialCode,
		Name: input.Name, ISO2: input.ISO2, Emoji: input.Emoji, Sort: input.Sort,
	})
}

func (l *Logic) SetSMSRegionEnabled(ctx context.Context, input appmodel.SetSMSEnabled) (smsmodel.Region, error) {
	if l.deps.SMS == nil {
		return smsmodel.Region{}, model.ErrUnavailable
	}
	return l.deps.SMS.SetRegionEnabled(ctx, smsScope(ctx), smsmodel.SetEnabled{CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code, Enabled: input.Enabled})
}

func (l *Logic) RetireSMSRegion(ctx context.Context, input appmodel.RetireSMS) (smsmodel.Region, error) {
	if l.deps.SMS == nil {
		return smsmodel.Region{}, model.ErrUnavailable
	}
	return l.deps.SMS.RetireRegion(ctx, smsScope(ctx), smsmodel.Retire{CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code})
}

func (l *Logic) GetSMSMerchantGrant(ctx context.Context, merchantID, shopID int64) (smsmodel.MerchantGrant, error) {
	if l.deps.SMS == nil {
		return smsmodel.MerchantGrant{}, model.ErrUnavailable
	}
	return l.deps.SMS.GetMerchantGrant(ctx, smsScope(ctx), merchantID, shopID)
}

func (l *Logic) PutSMSMerchantGrant(ctx context.Context, input appmodel.PutSMSMerchantGrant) (smsmodel.MerchantGrant, error) {
	if l.deps.SMS == nil {
		return smsmodel.MerchantGrant{}, model.ErrUnavailable
	}
	return l.deps.SMS.PutMerchantGrant(ctx, smsScope(ctx), smsmodel.PutMerchantGrant{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, MerchantID: input.MerchantID, ShopID: input.ShopID, DialCodes: input.DialCodes,
	})
}

func (l *Logic) TestSMSSend(ctx context.Context, input appmodel.TestSMSSend) (smsmodel.TestSendResult, error) {
	if l.deps.SMS == nil {
		return smsmodel.TestSendResult{}, model.ErrUnavailable
	}
	return l.deps.SMS.TestSend(ctx, smsScope(ctx), smsmodel.TestSend{ChannelCode: input.ChannelCode, Phone: input.Phone})
}
