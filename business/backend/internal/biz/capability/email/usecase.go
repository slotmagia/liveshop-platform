package email

import (
	"context"
	"fmt"
	"strings"

	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
)

type UseCase struct {
	repository Repository
	sender     Sender
}

func New(repository Repository, sender Sender) *UseCase {
	return &UseCase{repository: repository, sender: sender}
}

func (u *UseCase) GetConfig(ctx context.Context, scope emailmodel.Scope) (emailmodel.Config, error) {
	if !scope.Valid() {
		return emailmodel.Config{}, emailmodel.ErrInvalid
	}
	return u.repository.GetConfig(ctx, scope)
}

func (u *UseCase) UpsertConfig(ctx context.Context, scope emailmodel.Scope, input emailmodel.UpsertConfig) (emailmodel.Config, error) {
	input = emailmodel.NormalizeUpsert(input)
	if err := emailmodel.ValidateUpsert(scope, input); err != nil {
		return emailmodel.Config{}, err
	}
	return u.repository.UpsertConfig(ctx, scope, input, emailmodel.RequestHash(input))
}

func (u *UseCase) SetEnabled(ctx context.Context, scope emailmodel.Scope, input emailmodel.SetEnabled) (emailmodel.Config, error) {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	if err := emailmodel.ValidateSetEnabled(scope, input); err != nil {
		return emailmodel.Config{}, err
	}
	return u.repository.SetEnabled(ctx, scope, input, emailmodel.RequestHash(input))
}

func (u *UseCase) TestSend(ctx context.Context, scope emailmodel.Scope, input emailmodel.TestSend) (emailmodel.TestSendResult, error) {
	input.To = strings.TrimSpace(input.To)
	input.Subject = strings.TrimSpace(input.Subject)
	if !scope.Valid() || !emailmodel.ValidateEmail(input.To) {
		return emailmodel.TestSendResult{}, emailmodel.ErrInvalid
	}
	selected, err := u.repository.LoadSecrets(ctx, scope)
	if err != nil {
		return emailmodel.TestSendResult{}, err
	}
	if !selected.Config.Configured() {
		return emailmodel.TestSendResult{}, emailmodel.ErrNotConfigured
	}
	if input.Subject == "" {
		input.Subject = "邮件配置测试 · Email config test"
	}
	detail, err := u.sender.Send(ctx, selected.Config.Driver, selected.Values, input)
	result := emailmodel.TestSendResult{OK: err == nil, Detail: detail, Driver: selected.Config.Driver, Mock: selected.Config.Driver == emailmodel.DriverMock}
	if err != nil {
		if result.Detail == "" {
			result.Detail = err.Error()
		}
		return result, nil
	}
	if result.Detail == "" {
		result.Detail = fmt.Sprintf("sent via %s", selected.Config.Driver)
	}
	return result, nil
}
