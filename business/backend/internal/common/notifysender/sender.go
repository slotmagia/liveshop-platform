package notifysender

import (
	"context"
	"errors"
	"strings"

	emailuc "github.com/liveshop-platform/module-platform/internal/biz/capability/email"
	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/notification"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	smsuc "github.com/liveshop-platform/module-platform/internal/biz/capability/sms"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
)

type Adapter struct {
	SMS   *smsuc.UseCase
	Email *emailuc.UseCase
}

var _ notification.ChannelSender = Adapter{}

func (a Adapter) SendSMS(ctx context.Context, phone, text string) (notifymodel.ChannelSendResult, error) {
	if a.SMS == nil {
		return notifymodel.ChannelSendResult{}, errors.New("sms capability is unavailable")
	}
	result, err := a.SMS.SendMessage(ctx, smsmodel.Scope{Realm: "PLATFORM", Subject: "notify-dispatch"}, phone, text)
	return sendResult(result.OK, result.Detail, err)
}

func (a Adapter) SendEmail(ctx context.Context, to, subject, html string) (notifymodel.ChannelSendResult, error) {
	if a.Email == nil {
		return notifymodel.ChannelSendResult{}, errors.New("email capability is unavailable")
	}
	result, err := a.Email.TestSend(ctx, emailmodel.Scope{Realm: "PLATFORM", Subject: "notify-dispatch"}, emailmodel.TestSend{To: to, Subject: subject, BodyHTML: html})
	return sendResult(result.OK, result.Detail, err)
}

func sendResult(ok bool, detail string, err error) (notifymodel.ChannelSendResult, error) {
	if err != nil {
		return notifymodel.ChannelSendResult{Detail: detail, Unknown: isUnknown(err)}, err
	}
	if !ok {
		return notifymodel.ChannelSendResult{Detail: detail}, errors.New(strings.TrimSpace(detail))
	}
	return notifymodel.ChannelSendResult{Detail: detail}, nil
}

func isUnknown(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") || strings.Contains(text, "deadline")
}
