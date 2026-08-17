package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
)

type UseCase struct {
	repository Repository
	sender     Sender
}

func New(repository Repository, sender Sender) *UseCase {
	return &UseCase{repository: repository, sender: sender}
}

func (u *UseCase) ListChannels(ctx context.Context, scope storagemodel.Scope, filter storagemodel.ChannelFilter) ([]storagemodel.Channel, error) {
	if !scope.Valid() {
		return nil, storagemodel.ErrInvalid
	}
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Driver = storagemodel.Driver(strings.ToLower(strings.TrimSpace(string(filter.Driver))))
	filter.Lifecycle = storagemodel.Lifecycle(strings.ToUpper(strings.TrimSpace(string(filter.Lifecycle))))
	if filter.Driver != "" {
		if _, ok := storagemodel.DefinitionFor(filter.Driver); !ok {
			return nil, storagemodel.ErrInvalid
		}
	}
	if filter.Lifecycle != "" && filter.Lifecycle != storagemodel.LifecycleActive && filter.Lifecycle != storagemodel.LifecycleRetired {
		return nil, storagemodel.ErrInvalid
	}
	return u.repository.ListChannels(ctx, scope, filter)
}

func (u *UseCase) UpsertChannel(ctx context.Context, scope storagemodel.Scope, input storagemodel.UpsertChannel) (storagemodel.Channel, error) {
	input = storagemodel.NormalizeUpsert(input)
	if err := storagemodel.ValidateUpsert(scope, input); err != nil {
		return storagemodel.Channel{}, err
	}
	return u.repository.UpsertChannel(ctx, scope, input, storagemodel.RequestHash(input))
}

func (u *UseCase) SetEnabled(ctx context.Context, scope storagemodel.Scope, input storagemodel.SetEnabled) (storagemodel.Channel, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if err := storagemodel.ValidateSetEnabled(scope, input); err != nil {
		return storagemodel.Channel{}, err
	}
	return u.repository.SetEnabled(ctx, scope, input, storagemodel.RequestHash(input))
}

func (u *UseCase) SetDefault(ctx context.Context, scope storagemodel.Scope, input storagemodel.SetDefault) (storagemodel.Channel, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if err := storagemodel.ValidateSetDefault(scope, input); err != nil {
		return storagemodel.Channel{}, err
	}
	return u.repository.SetDefault(ctx, scope, input, storagemodel.RequestHash(input))
}

func (u *UseCase) Retire(ctx context.Context, scope storagemodel.Scope, input storagemodel.Retire) (storagemodel.Channel, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if err := storagemodel.ValidateRetire(scope, input); err != nil {
		return storagemodel.Channel{}, err
	}
	return u.repository.Retire(ctx, scope, input, storagemodel.RequestHash(input))
}

func (u *UseCase) Test(ctx context.Context, scope storagemodel.Scope, input storagemodel.TestChannel) (storagemodel.TestResult, error) {
	if !scope.Valid() {
		return storagemodel.TestResult{}, storagemodel.ErrInvalid
	}
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if input.Code == "" {
		return storagemodel.TestResult{}, storagemodel.ErrInvalid
	}
	selected, err := u.repository.LoadSecrets(ctx, scope, input.Code)
	if err != nil {
		return storagemodel.TestResult{}, err
	}
	if selected.Channel.Lifecycle != storagemodel.LifecycleActive {
		return storagemodel.TestResult{}, storagemodel.ErrInvalid
	}
	key := fmt.Sprintf("_storage_test/ping-%d.txt", time.Now().UnixMilli())
	url, err := u.sender.Put(ctx, selected.Channel.Driver, selected.Config, key, []byte("liveshop storage connectivity test\n"))
	result := storagemodel.TestResult{OK: err == nil, Detail: "", URL: url, Driver: selected.Channel.Driver}
	if err != nil {
		result.Detail = err.Error()
		return result, nil
	}
	result.Detail = "写入成功，打开返回的 URL 验证是否可访问"
	return result, nil
}

func (u *UseCase) ReadLocal(_ context.Context, key string) (storagemodel.Object, error) {
	return u.sender.GetLocal(key)
}
