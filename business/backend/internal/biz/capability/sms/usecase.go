package sms

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
)

type UseCase struct {
	repository Repository
	sender     Sender
}

func New(repository Repository, sender Sender) *UseCase {
	return &UseCase{repository: repository, sender: sender}
}

func (u *UseCase) ListChannels(ctx context.Context, scope smsmodel.Scope, filter smsmodel.ChannelFilter) ([]smsmodel.Channel, error) {
	if !scope.Valid() {
		return nil, smsmodel.ErrInvalid
	}
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Driver = smsmodel.Driver(strings.ToLower(strings.TrimSpace(string(filter.Driver))))
	filter.Lifecycle = smsmodel.Lifecycle(strings.ToUpper(strings.TrimSpace(string(filter.Lifecycle))))
	if filter.Driver != "" {
		if _, ok := smsmodel.DefinitionFor(filter.Driver); !ok {
			return nil, smsmodel.ErrInvalid
		}
	}
	if filter.Lifecycle != "" && filter.Lifecycle != smsmodel.LifecycleActive && filter.Lifecycle != smsmodel.LifecycleRetired {
		return nil, smsmodel.ErrInvalid
	}
	return u.repository.ListChannels(ctx, scope, filter)
}

func (u *UseCase) UpsertChannel(ctx context.Context, scope smsmodel.Scope, input smsmodel.UpsertChannel) (smsmodel.Channel, error) {
	input = smsmodel.NormalizeUpsertChannel(input)
	if err := smsmodel.ValidateUpsertChannel(scope, input); err != nil {
		return smsmodel.Channel{}, err
	}
	return u.repository.UpsertChannel(ctx, scope, input, smsmodel.RequestHash(input))
}

func (u *UseCase) SetChannelEnabled(ctx context.Context, scope smsmodel.Scope, input smsmodel.SetEnabled) (smsmodel.Channel, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if err := smsmodel.ValidateSetEnabled(scope, input); err != nil {
		return smsmodel.Channel{}, err
	}
	return u.repository.SetChannelEnabled(ctx, scope, input, smsmodel.RequestHash(input))
}

func (u *UseCase) RetireChannel(ctx context.Context, scope smsmodel.Scope, input smsmodel.Retire) (smsmodel.Channel, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if err := smsmodel.ValidateRetire(scope, input, false); err != nil {
		return smsmodel.Channel{}, err
	}
	return u.repository.RetireChannel(ctx, scope, input, smsmodel.RequestHash(input))
}

func (u *UseCase) ListRegions(ctx context.Context, scope smsmodel.Scope, filter smsmodel.RegionFilter) ([]smsmodel.Region, error) {
	if !scope.Valid() {
		return nil, smsmodel.ErrInvalid
	}
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Lifecycle = smsmodel.Lifecycle(strings.ToUpper(strings.TrimSpace(string(filter.Lifecycle))))
	if filter.Lifecycle != "" && filter.Lifecycle != smsmodel.LifecycleActive && filter.Lifecycle != smsmodel.LifecycleRetired {
		return nil, smsmodel.ErrInvalid
	}
	return u.repository.ListRegions(ctx, scope, filter)
}

func (u *UseCase) UpsertRegion(ctx context.Context, scope smsmodel.Scope, input smsmodel.UpsertRegion) (smsmodel.Region, error) {
	input = smsmodel.NormalizeUpsertRegion(input)
	if err := smsmodel.ValidateUpsertRegion(scope, input); err != nil {
		return smsmodel.Region{}, err
	}
	return u.repository.UpsertRegion(ctx, scope, input, smsmodel.RequestHash(input))
}

func (u *UseCase) SetRegionEnabled(ctx context.Context, scope smsmodel.Scope, input smsmodel.SetEnabled) (smsmodel.Region, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.TrimSpace(input.Code)
	if err := smsmodel.ValidateSetEnabled(scope, input); err != nil {
		return smsmodel.Region{}, err
	}
	return u.repository.SetRegionEnabled(ctx, scope, input, smsmodel.RequestHash(input))
}

func (u *UseCase) RetireRegion(ctx context.Context, scope smsmodel.Scope, input smsmodel.Retire) (smsmodel.Region, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.TrimSpace(input.Code)
	if err := smsmodel.ValidateRetire(scope, input, true); err != nil {
		return smsmodel.Region{}, err
	}
	return u.repository.RetireRegion(ctx, scope, input, smsmodel.RequestHash(input))
}

func (u *UseCase) GetMerchantGrant(ctx context.Context, scope smsmodel.Scope, merchantID, shopID int64) (smsmodel.MerchantGrant, error) {
	if !scope.Valid() || merchantID <= 0 || shopID <= 0 {
		return smsmodel.MerchantGrant{}, smsmodel.ErrInvalid
	}
	return u.repository.GetMerchantGrant(ctx, scope, merchantID, shopID)
}

func (u *UseCase) PutMerchantGrant(ctx context.Context, scope smsmodel.Scope, input smsmodel.PutMerchantGrant) (smsmodel.MerchantGrant, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	codes := make([]string, 0, len(input.DialCodes))
	for _, dial := range input.DialCodes {
		if trimmed := strings.TrimSpace(dial); trimmed != "" {
			codes = append(codes, trimmed)
		}
	}
	input.DialCodes = codes
	if err := smsmodel.ValidatePutGrant(scope, input); err != nil {
		return smsmodel.MerchantGrant{}, err
	}
	return u.repository.PutMerchantGrant(ctx, scope, input, smsmodel.RequestHash(input))
}

func (u *UseCase) TestSend(ctx context.Context, scope smsmodel.Scope, input smsmodel.TestSend) (smsmodel.TestSendResult, error) {
	return u.send(ctx, scope, input.ChannelCode, input.Phone, "")
}

func (u *UseCase) SendMessage(ctx context.Context, scope smsmodel.Scope, phone, text string) (smsmodel.TestSendResult, error) {
	return u.send(ctx, scope, "", phone, text)
}

func (u *UseCase) send(ctx context.Context, scope smsmodel.Scope, channelCode, phone, text string) (smsmodel.TestSendResult, error) {
	if !scope.Valid() || !smsmodel.ValidatePhone(phone) {
		return smsmodel.TestSendResult{}, smsmodel.ErrInvalid
	}
	channelCode = strings.ToLower(strings.TrimSpace(channelCode))
	phone = strings.TrimSpace(phone)
	var selected smsmodel.ChannelSecrets
	if channelCode != "" {
		item, err := u.repository.LoadChannelSecrets(ctx, scope, channelCode)
		if err != nil {
			return smsmodel.TestSendResult{}, err
		}
		if item.Channel.Lifecycle != smsmodel.LifecycleActive || !item.Channel.Enabled {
			return smsmodel.TestSendResult{}, smsmodel.ErrInvalid
		}
		selected = item
	} else {
		channels, err := u.repository.ListChannels(ctx, scope, smsmodel.ChannelFilter{Lifecycle: smsmodel.LifecycleActive})
		if err != nil {
			return smsmodel.TestSendResult{}, err
		}
		routed := smsmodel.RouteChannels(phone, channels)
		if len(routed) == 0 {
			return smsmodel.TestSendResult{}, smsmodel.ErrNoChannel
		}
		item, err := u.repository.LoadChannelSecrets(ctx, scope, routed[0].Code)
		if err != nil {
			return smsmodel.TestSendResult{}, err
		}
		selected = item
	}
	code := strings.TrimSpace(text)
	if code == "" {
		code = generateCode(6)
	}
	detail, err := u.sender.Send(ctx, selected.Channel.Driver, selected.Config, phone, code)
	result := smsmodel.TestSendResult{OK: err == nil, Detail: detail, ChannelCode: selected.Channel.Code, Driver: selected.Channel.Driver, Mock: selected.Channel.Driver == smsmodel.DriverMock}
	if err != nil {
		if result.Detail == "" {
			result.Detail = err.Error()
		}
		return result, nil
	}
	if result.Mock {
		result.Code = code
	}
	if result.Detail == "" {
		result.Detail = fmt.Sprintf("sent via %s", selected.Channel.Code)
	}
	return result, nil
}

func generateCode(length int) string {
	const digits = "0123456789"
	out := make([]byte, length)
	for index := range out {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		out[index] = digits[n.Int64()]
	}
	return string(out)
}
