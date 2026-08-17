package logic

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
)

func liveProviderScope(ctx context.Context) providermodel.Scope {
	claims := authctx.Capability(ctx)
	return providermodel.Scope{Realm: claims.Realm.String(), MerchantID: claims.MerchantID, Subject: claims.Subject}
}

func (l *Logic) LiveProviderDrivers(context.Context) []providermodel.DriverDefinition {
	return providermodel.DriverDefinitions()
}

func (l *Logic) ListLiveProviders(ctx context.Context, filter providermodel.Filter) ([]providermodel.Provider, error) {
	if l.deps.LiveProvider == nil {
		return nil, model.ErrUnavailable
	}
	return l.deps.LiveProvider.List(ctx, liveProviderScope(ctx), filter)
}

func (l *Logic) PutLiveProvider(ctx context.Context, input appmodel.PutLiveProvider) (providermodel.Provider, error) {
	if l.deps.LiveProvider == nil {
		return providermodel.Provider{}, model.ErrUnavailable
	}
	return l.deps.LiveProvider.Upsert(ctx, liveProviderScope(ctx), providermodel.Upsert{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, Code: input.Code, Name: input.Name,
		Driver: input.Driver, App: input.App, PushDomain: input.PushDomain, PullDomain: input.PullDomain,
		AgoraAppID: input.AgoraAppID, Codec: input.Codec, IngestDomain: input.IngestDomain, Region: input.Region,
		TTLSeconds: input.TTLSeconds, IsDefault: input.IsDefault,
		Secret: input.Secret, AppCertificate: input.AppCertificate, CustomerCredential: input.CustomerCredential,
	})
}

func (l *Logic) RetireLiveProvider(ctx context.Context, input appmodel.RetireLiveProvider) (providermodel.Provider, error) {
	if l.deps.LiveProvider == nil {
		return providermodel.Provider{}, model.ErrUnavailable
	}
	return l.deps.LiveProvider.Retire(ctx, liveProviderScope(ctx), providermodel.Retire{Code: input.Code, CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion})
}

func (l *Logic) GetLiveProviderAssignments(ctx context.Context, merchantID int64) (providermodel.AssignmentSet, error) {
	if l.deps.LiveProvider == nil {
		return providermodel.AssignmentSet{}, model.ErrUnavailable
	}
	return l.deps.LiveProvider.GetAssignments(ctx, merchantID)
}

func (l *Logic) PutLiveProviderAssignments(ctx context.Context, input appmodel.PutLiveProviderAssignments) (providermodel.AssignmentSet, error) {
	if l.deps.LiveProvider == nil {
		return providermodel.AssignmentSet{}, model.ErrUnavailable
	}
	providers := make([]providermodel.Assignment, 0, len(input.Providers))
	for _, item := range input.Providers {
		providers = append(providers, providermodel.Assignment{ProviderCode: item.ProviderCode, Name: item.Name, Enabled: item.Enabled, Default: item.Default})
	}
	return l.deps.LiveProvider.PutAssignments(ctx, providermodel.PutAssignments{
		CommandKey: input.CommandKey, ExpectedVersion: input.ExpectedVersion, MerchantID: input.MerchantID, Providers: providers,
	})
}
