package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

func emailScope(ctx context.Context) emailmodel.Scope {
	claims := authctx.Capability(ctx)
	return emailmodel.Scope{Realm: claims.Realm.String(), MerchantID: claims.MerchantID, Subject: claims.Subject}
}

func (l *Logic) EmailDrivers(context.Context) []emailmodel.DriverDefinition {
	return emailmodel.DriverDefinitions()
}

func (l *Logic) GetEmailConfig(ctx context.Context) (emailmodel.Config, error) {
	if l.deps.Email == nil {
		return emailmodel.Config{}, model.ErrUnavailable
	}
	return l.deps.Email.GetConfig(ctx, emailScope(ctx))
}

func (l *Logic) PutEmailConfig(ctx context.Context, input appmodel.PutEmailConfig) (emailmodel.Config, error) {
	if l.deps.Email == nil {
		return emailmodel.Config{}, model.ErrUnavailable
	}
	return l.deps.Email.UpsertConfig(ctx, emailScope(ctx), emailmodel.UpsertConfig{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Driver: input.Driver, PublicConfig: input.PublicConfig, Secrets: input.Secrets,
	})
}

func (l *Logic) SetEmailEnabled(ctx context.Context, input appmodel.SetEmailEnabled) (emailmodel.Config, error) {
	if l.deps.Email == nil {
		return emailmodel.Config{}, model.ErrUnavailable
	}
	return l.deps.Email.SetEnabled(ctx, emailScope(ctx), emailmodel.SetEnabled{CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Enabled: input.Enabled})
}

func (l *Logic) TestEmailSend(ctx context.Context, input appmodel.TestEmailSend) (emailmodel.TestSendResult, error) {
	if l.deps.Email == nil {
		return emailmodel.TestSendResult{}, model.ErrUnavailable
	}
	return l.deps.Email.TestSend(ctx, emailScope(ctx), emailmodel.TestSend{To: input.To, Subject: input.Subject})
}
