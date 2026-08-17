package liveprovider

import (
	"context"
	"strings"

	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
)

type UseCase struct{ repository Repository }

func New(repository Repository) *UseCase { return &UseCase{repository: repository} }

func (u *UseCase) List(ctx context.Context, scope providermodel.Scope, filter providermodel.Filter) ([]providermodel.Provider, error) {
	if !scope.Valid() {
		return nil, providermodel.ErrInvalid
	}
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Kind = providermodel.Kind(strings.ToUpper(strings.TrimSpace(string(filter.Kind))))
	filter.Driver = providermodel.Driver(strings.ToUpper(strings.TrimSpace(string(filter.Driver))))
	filter.Lifecycle = providermodel.Lifecycle(strings.ToUpper(strings.TrimSpace(string(filter.Lifecycle))))
	if filter.Kind != "" && filter.Kind != providermodel.KindRTMP && filter.Kind != providermodel.KindRTC {
		return nil, providermodel.ErrInvalid
	}
	if filter.Driver != "" {
		if _, ok := providermodel.KindFor(filter.Driver); !ok {
			return nil, providermodel.ErrInvalid
		}
	}
	if filter.Lifecycle != "" && filter.Lifecycle != providermodel.LifecycleActive && filter.Lifecycle != providermodel.LifecycleRetired {
		return nil, providermodel.ErrInvalid
	}
	return u.repository.List(ctx, scope, filter)
}

func (u *UseCase) Upsert(ctx context.Context, scope providermodel.Scope, input providermodel.Upsert) (providermodel.Provider, error) {
	input = providermodel.NormalizeUpsert(input)
	if err := providermodel.ValidateUpsert(scope, input); err != nil {
		return providermodel.Provider{}, err
	}
	return u.repository.Upsert(ctx, scope, input, providermodel.RequestHash(input))
}

func (u *UseCase) Retire(ctx context.Context, scope providermodel.Scope, input providermodel.Retire) (providermodel.Provider, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if err := providermodel.ValidateRetire(scope, input); err != nil {
		return providermodel.Provider{}, err
	}
	return u.repository.Retire(ctx, scope, input, providermodel.RequestHash(input))
}

func (u *UseCase) GetAssignments(ctx context.Context, merchantID int64) (providermodel.AssignmentSet, error) {
	if merchantID <= 0 {
		return providermodel.AssignmentSet{}, providermodel.ErrInvalid
	}
	repo, ok := u.repository.(AssignmentRepository)
	if !ok {
		return providermodel.AssignmentSet{}, providermodel.ErrInvalid
	}
	value, err := repo.GetAssignments(ctx, merchantID)
	if err != nil {
		return providermodel.AssignmentSet{}, err
	}
	catalog, err := u.repository.List(ctx, providermodel.Scope{Realm: "PLATFORM", Subject: "identity-compose"}, providermodel.Filter{Lifecycle: providermodel.LifecycleActive})
	if err != nil {
		return value, nil
	}
	granted := map[string]providermodel.Assignment{}
	for _, item := range value.Providers {
		granted[item.ProviderCode] = item
	}
	merged := make([]providermodel.Assignment, 0, len(catalog))
	for _, provider := range catalog {
		item := providermodel.Assignment{ProviderCode: provider.Code, Name: provider.Name, Enabled: false, Default: false}
		if current, ok := granted[provider.Code]; ok {
			item.Enabled = current.Enabled
			item.Default = current.Default
		}
		merged = append(merged, item)
	}
	value.Providers = merged
	return value, nil
}

func (u *UseCase) PutAssignments(ctx context.Context, input providermodel.PutAssignments) (providermodel.AssignmentSet, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	if input.MerchantID <= 0 || len(input.CommandKey) < 8 {
		return providermodel.AssignmentSet{}, providermodel.ErrInvalid
	}
	repo, ok := u.repository.(AssignmentRepository)
	if !ok {
		return providermodel.AssignmentSet{}, providermodel.ErrInvalid
	}
	return repo.PutAssignments(ctx, input)
}
