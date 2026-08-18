package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
)

type stubSender struct {
	smsErr, emailErr error
	unknown          bool
}

func (s stubSender) SendSMS(context.Context, string, string) (notifymodel.ChannelSendResult, error) {
	return notifymodel.ChannelSendResult{Detail: "mock sms", Unknown: s.unknown}, s.smsErr
}
func (s stubSender) SendEmail(context.Context, string, string, string) (notifymodel.ChannelSendResult, error) {
	return notifymodel.ChannelSendResult{Detail: "mock email", Unknown: s.unknown}, s.emailErr
}

func seedSMSTemplate(t *testing.T, usecase *UseCase, commandSuffix string) {
	t.Helper()
	scope := notifymodel.Scope{Realm: "PLATFORM", Subject: "op"}
	if _, err := usecase.UpsertLibraryTemplate(context.Background(), scope, notifymodel.UpsertLibraryTemplate{
		Code: "identity.auth.otp.requested.sms", Channel: notifymodel.ChannelSMS, CommandKey: "lib-" + commandSuffix, ExpectedVersion: 0,
		TextTemplate: "code {{code}} ttl {{ttlSeconds}}", Variables: []string{"code", "ttlSeconds"},
	}); err != nil {
		t.Fatal(err)
	}
	event, err := usecase.GetEvent(context.Background(), "identity.auth.otp.requested")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecase.ReplacePolicy(context.Background(), scope, notifymodel.ReplacePolicy{
		EventKey: event.EventKey, CommandKey: "pol-" + commandSuffix, ExpectedVersion: event.Policy.Version, DispatchMode: event.Policy.DispatchMode,
		Channels: map[notifymodel.Channel]notifymodel.ChannelPolicy{
			notifymodel.ChannelSMS:   {Enabled: true, TemplateCode: "identity.auth.otp.requested.sms"},
			notifymodel.ChannelEmail: {Enabled: false},
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func testEvent() notifymodel.Declaration {
	return notifymodel.Declaration{
		EventKey: "identity.auth.otp.requested", ModuleID: "identity", ModuleName: "Identity",
		OperationID: "identity.shop.login.otp.create", Title: "登录验证码", Variables: []string{"code", "ttlSeconds"},
		AllowedChannels: []notifymodel.Channel{notifymodel.ChannelSMS, notifymodel.ChannelEmail}, DefaultDispatch: notifymodel.ModeSync,
	}
}

func setupUseCase(t *testing.T, sender ChannelSender) (*UseCase, *memoryRepository) {
	t.Helper()
	repo := newMemoryRepository()
	if err := repo.Project(context.Background(), 2, []notifymodel.Declaration{testEvent()}); err != nil {
		t.Fatal(err)
	}
	return New(repo, sender), repo
}

func TestDispatchRejectsCrossModuleEventKey(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{})
	_, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "trade", Subject: "trade"}, notifymodel.DispatchInput{
		EventKey: "identity.auth.otp.requested", DeliveryKey: "identity.auth.otp.requested:c1", MerchantID: 1, ShopID: 1,
		Recipients: notifymodel.Recipients{Phone: "+8613800138000"}, Variables: map[string]string{"code": "123456", "ttlSeconds": "60"},
	})
	if !errors.Is(err, notifymodel.ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
}

func TestDispatchRejectsMissingVariables(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{})
	_, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, notifymodel.DispatchInput{
		EventKey: "identity.auth.otp.requested", DeliveryKey: "identity.auth.otp.requested:c2", MerchantID: 1, ShopID: 1,
		Recipients: notifymodel.Recipients{Phone: "+8613800138000"}, Variables: map[string]string{"code": "123456"},
	})
	if !errors.Is(err, notifymodel.ErrInvalid) {
		t.Fatalf("err=%v", err)
	}
}

func TestDispatchAllChannelsOffSucceedsEmpty(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{})
	scope := notifymodel.Scope{Realm: "PLATFORM", Subject: "op"}
	event, _ := usecase.GetEvent(context.Background(), "identity.auth.otp.requested")
	if _, err := usecase.ReplacePolicy(context.Background(), scope, notifymodel.ReplacePolicy{
		EventKey: event.EventKey, CommandKey: "cmd-off", ExpectedVersion: event.Policy.Version, DispatchMode: notifymodel.ModeSync,
		Channels: map[notifymodel.Channel]notifymodel.ChannelPolicy{notifymodel.ChannelSMS: {Enabled: false}, notifymodel.ChannelEmail: {Enabled: false}},
	}); err != nil {
		t.Fatal(err)
	}
	result, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, notifymodel.DispatchInput{
		EventKey: "identity.auth.otp.requested", DeliveryKey: "identity.auth.otp.requested:c3", MerchantID: 1, ShopID: 1,
		Variables: map[string]string{"code": "123456", "ttlSeconds": "60"},
	})
	if err != nil || len(result.Deliveries) != 0 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestDispatchMissingTemplateIsPermanentFailure(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{})
	result, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, notifymodel.DispatchInput{
		EventKey: "identity.auth.otp.requested", DeliveryKey: "identity.auth.otp.requested:c4", MerchantID: 1, ShopID: 1,
		Recipients: notifymodel.Recipients{Phone: "+8613800138000", Email: "a@example.com"}, Variables: map[string]string{"code": "123456", "ttlSeconds": "60"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Deliveries) != 2 {
		t.Fatalf("deliveries=%d", len(result.Deliveries))
	}
	for _, item := range result.Deliveries {
		if item.Status != notifymodel.StatusFailedPermanent {
			t.Fatalf("status=%s", item.Status)
		}
	}
}

func TestDispatchIdempotentReplay(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{})
	seedSMSTemplate(t, usecase, "c5")
	input := notifymodel.DispatchInput{
		EventKey: "identity.auth.otp.requested", DeliveryKey: "identity.auth.otp.requested:c5", MerchantID: 1, ShopID: 1,
		Recipients: notifymodel.Recipients{Phone: "+8613800138000", Email: "a@example.com"}, Variables: map[string]string{"code": "123456", "ttlSeconds": "60"},
	}
	first, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Deliveries) == 0 || !second.Deliveries[0].Deduped || first.Deliveries[0].DeliveryID != second.Deliveries[0].DeliveryID {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
	input.Variables["code"] = "999999"
	if _, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, input); !errors.Is(err, notifymodel.ErrConflict) {
		t.Fatalf("err=%v", err)
	}
}

func TestDispatchUnknownDoesNotLetCallerChangeKey(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{unknown: true})
	seedSMSTemplate(t, usecase, "c6")
	input := notifymodel.DispatchInput{
		EventKey: "identity.auth.otp.requested", DeliveryKey: "identity.auth.otp.requested:c6", MerchantID: 1, ShopID: 1,
		Recipients: notifymodel.Recipients{Phone: "+8613800138000", Email: "a@example.com"}, Variables: map[string]string{"code": "123456", "ttlSeconds": "60"},
	}
	result, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, input)
	if err != nil {
		t.Fatal(err)
	}
	var sms notifymodel.DeliveryResult
	for _, item := range result.Deliveries {
		if item.Channel == notifymodel.ChannelSMS {
			sms = item
		}
	}
	if sms.Status != notifymodel.StatusUnknown {
		t.Fatalf("status=%s", sms.Status)
	}
	replay, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, input)
	if err != nil {
		t.Fatal(err)
	}
	var replaySMS notifymodel.DeliveryResult
	for _, item := range replay.Deliveries {
		if item.Channel == notifymodel.ChannelSMS {
			replaySMS = item
		}
	}
	if replaySMS.DeliveryID != sms.DeliveryID {
		t.Fatalf("replay changed delivery id: first=%s replay=%s", sms.DeliveryID, replaySMS.DeliveryID)
	}
	input.Variables["code"] = "000000"
	if _, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, input); !errors.Is(err, notifymodel.ErrConflict) {
		t.Fatalf("err=%v", err)
	}
}

func TestScheduledDispatchWaitsForDue(t *testing.T) {
	usecase, repo := setupUseCase(t, stubSender{})
	seedSMSTemplate(t, usecase, "sched")
	scope := notifymodel.Scope{Realm: "PLATFORM", Subject: "op"}
	event, _ := usecase.GetEvent(context.Background(), "identity.auth.otp.requested")
	if _, err := usecase.ReplacePolicy(context.Background(), scope, notifymodel.ReplacePolicy{
		EventKey: event.EventKey, CommandKey: "cmd-sched", ExpectedVersion: event.Policy.Version, DispatchMode: notifymodel.ModeScheduled, DelaySeconds: 60,
		Channels: map[notifymodel.Channel]notifymodel.ChannelPolicy{
			notifymodel.ChannelSMS:   {Enabled: true, TemplateCode: "identity.auth.otp.requested.sms"},
			notifymodel.ChannelEmail: {Enabled: false},
		},
	}); err != nil {
		t.Fatal(err)
	}
	result, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, notifymodel.DispatchInput{
		EventKey: event.EventKey, DeliveryKey: "identity.auth.otp.requested:c7", MerchantID: 1, ShopID: 1,
		Recipients: notifymodel.Recipients{Phone: "+8613800138000"}, Variables: map[string]string{"code": "123456", "ttlSeconds": "60"},
		NotBefore: time.Now().UTC().Add(time.Hour),
	})
	if err != nil || result.Deliveries[0].Status != notifymodel.StatusScheduled {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if err := usecase.RecoverDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	item, _ := repo.GetDelivery(context.Background(), result.Deliveries[0].DeliveryID)
	if item.Status != notifymodel.StatusScheduled {
		t.Fatalf("status=%s", item.Status)
	}
}

func TestUnknownSenderTimeoutMapsToUnknown(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{smsErr: errors.New("context deadline exceeded")})
	seedSMSTemplate(t, usecase, "to")
	result, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, notifymodel.DispatchInput{
		EventKey: "identity.auth.otp.requested", DeliveryKey: "identity.auth.otp.requested:c8", MerchantID: 1, ShopID: 1,
		Recipients: notifymodel.Recipients{Phone: "+8613800138000", Email: "a@example.com"}, Variables: map[string]string{"code": "1", "ttlSeconds": "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range result.Deliveries {
		if item.Channel == notifymodel.ChannelSMS && item.Status != notifymodel.StatusUnknown {
			t.Fatalf("sms status=%s", item.Status)
		}
	}
}

func TestReplacePolicyRejectsTemplateVariableSuperset(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{})
	scope := notifymodel.Scope{Realm: "PLATFORM", Subject: "op"}
	if _, err := usecase.UpsertLibraryTemplate(context.Background(), scope, notifymodel.UpsertLibraryTemplate{
		Code: "otp.sms.extra", Channel: notifymodel.ChannelSMS, CommandKey: "lib-extra", ExpectedVersion: 0,
		TextTemplate: "code {{code}} extra {{orderNo}}", Variables: []string{"code", "orderNo"},
	}); err != nil {
		t.Fatal(err)
	}
	event, _ := usecase.GetEvent(context.Background(), "identity.auth.otp.requested")
	if _, err := usecase.ReplacePolicy(context.Background(), scope, notifymodel.ReplacePolicy{
		EventKey: event.EventKey, CommandKey: "pol-extra", ExpectedVersion: event.Policy.Version, DispatchMode: notifymodel.ModeSync,
		Channels: map[notifymodel.Channel]notifymodel.ChannelPolicy{
			notifymodel.ChannelSMS:   {Enabled: true, TemplateCode: "otp.sms.extra"},
			notifymodel.ChannelEmail: {Enabled: false},
		},
	}); !errors.Is(err, notifymodel.ErrInvalid) {
		t.Fatalf("err=%v", err)
	}
}

func TestLibraryTemplateReuseSendsSMS(t *testing.T) {
	usecase, _ := setupUseCase(t, stubSender{})
	seedSMSTemplate(t, usecase, "reuse")
	result, err := usecase.Dispatch(context.Background(), notifymodel.Caller{ModuleID: "identity", Subject: "identity"}, notifymodel.DispatchInput{
		EventKey: "identity.auth.otp.requested", DeliveryKey: "identity.auth.otp.requested:reuse", MerchantID: 1, ShopID: 1,
		Recipients: notifymodel.Recipients{Phone: "+8613800138000"}, Variables: map[string]string{"code": "123456", "ttlSeconds": "60"},
	})
	if err != nil || len(result.Deliveries) != 1 || result.Deliveries[0].Status != notifymodel.StatusSent {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
