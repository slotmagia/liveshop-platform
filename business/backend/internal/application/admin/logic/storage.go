package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

func storageScope(ctx context.Context) storagemodel.Scope {
	claims := authctx.Capability(ctx)
	return storagemodel.Scope{Realm: claims.Realm.String(), MerchantID: claims.MerchantID, Subject: claims.Subject}
}

func (l *Logic) StorageDrivers(context.Context) []storagemodel.DriverDefinition {
	return storagemodel.DriverDefinitions()
}

func (l *Logic) ListStorageChannels(ctx context.Context, filter storagemodel.ChannelFilter) ([]storagemodel.Channel, error) {
	if l.deps.Storage == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.Storage.ListChannels(ctx, storageScope(ctx), filter)
}

func (l *Logic) PutStorageChannel(ctx context.Context, input appmodel.PutStorageChannel) (storagemodel.Channel, error) {
	if l.deps.Storage == nil {
		return storagemodel.Channel{}, model.ErrUnavailable
	}
	return l.deps.Storage.UpsertChannel(ctx, storageScope(ctx), storagemodel.UpsertChannel{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code, Name: input.Name,
		Driver: input.Driver, PublicConfig: input.PublicConfig, Secrets: input.Secrets,
	})
}

func (l *Logic) SetStorageChannelEnabled(ctx context.Context, input appmodel.SetStorageEnabled) (storagemodel.Channel, error) {
	if l.deps.Storage == nil {
		return storagemodel.Channel{}, model.ErrUnavailable
	}
	return l.deps.Storage.SetEnabled(ctx, storageScope(ctx), storagemodel.SetEnabled{CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code, Enabled: input.Enabled})
}

func (l *Logic) SetStorageDefault(ctx context.Context, input appmodel.SetStorageDefault) (storagemodel.Channel, error) {
	if l.deps.Storage == nil {
		return storagemodel.Channel{}, model.ErrUnavailable
	}
	return l.deps.Storage.SetDefault(ctx, storageScope(ctx), storagemodel.SetDefault{CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code})
}

func (l *Logic) RetireStorageChannel(ctx context.Context, input appmodel.RetireStorage) (storagemodel.Channel, error) {
	if l.deps.Storage == nil {
		return storagemodel.Channel{}, model.ErrUnavailable
	}
	return l.deps.Storage.Retire(ctx, storageScope(ctx), storagemodel.Retire{CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code})
}

func (l *Logic) TestStorageChannel(ctx context.Context, input appmodel.TestStorageChannel) (storagemodel.TestResult, error) {
	if l.deps.Storage == nil {
		return storagemodel.TestResult{}, model.ErrUnavailable
	}
	return l.deps.Storage.Test(ctx, storageScope(ctx), storagemodel.TestChannel{Code: input.Code})
}

func (l *Logic) GetStorageObject(ctx context.Context, key string) (storagemodel.Object, error) {
	if l.deps.Storage == nil {
		return storagemodel.Object{}, model.ErrUnavailable
	}
	return l.deps.Storage.ReadLocal(ctx, key)
}
