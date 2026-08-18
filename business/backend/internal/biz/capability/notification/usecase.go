package notification

import (
	"context"
	"strings"
	"time"

	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

type UseCase struct {
	repository Repository
	sender     ChannelSender
	now        func() time.Time
}

func New(repository Repository, sender ChannelSender) *UseCase {
	return &UseCase{repository: repository, sender: sender, now: time.Now}
}

func (u *UseCase) Project(ctx context.Context, revision uint64, declarations []notifymodel.Declaration) error {
	if u.repository == nil {
		return notifymodel.ErrInvalid
	}
	return u.repository.Project(ctx, revision, declarations)
}

func (u *UseCase) ProjectCapabilities(ctx context.Context, revision uint64, modules []model.ActiveModuleCapability) error {
	declarations := make([]notifymodel.Declaration, 0)
	for _, item := range modules {
		declarations = append(declarations, notifymodel.DeclarationsFromBackend(item.ModuleID, item.ModuleName, item.Backend)...)
	}
	return u.Project(ctx, revision, declarations)
}

func (u *UseCase) ListEvents(ctx context.Context, filter notifymodel.EventFilter) ([]notifymodel.Event, error) {
	if u.repository == nil {
		return nil, notifymodel.ErrInvalid
	}
	filter.Module = strings.TrimSpace(filter.Module)
	filter.Channel = notifymodel.Channel(strings.ToUpper(strings.TrimSpace(string(filter.Channel))))
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if filter.Channel != "" && filter.Channel != notifymodel.ChannelSMS && filter.Channel != notifymodel.ChannelEmail && filter.Channel != notifymodel.ChannelInApp {
		return nil, notifymodel.ErrInvalid
	}
	return u.repository.ListEvents(ctx, filter)
}

func (u *UseCase) GetEvent(ctx context.Context, eventKey string) (notifymodel.Event, error) {
	if u.repository == nil {
		return notifymodel.Event{}, notifymodel.ErrInvalid
	}
	return u.repository.GetEvent(ctx, strings.TrimSpace(eventKey))
}

func (u *UseCase) ReplacePolicy(ctx context.Context, scope notifymodel.Scope, input notifymodel.ReplacePolicy) (notifymodel.Policy, error) {
	if u.repository == nil {
		return notifymodel.Policy{}, notifymodel.ErrInvalid
	}
	input.EventKey = strings.TrimSpace(input.EventKey)
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.DispatchMode = notifymodel.DispatchMode(strings.ToUpper(strings.TrimSpace(string(input.DispatchMode))))
	normalized := map[notifymodel.Channel]notifymodel.ChannelPolicy{}
	for channel, item := range input.Channels {
		item.TemplateCode = strings.TrimSpace(item.TemplateCode)
		normalized[notifymodel.Channel(strings.ToUpper(strings.TrimSpace(string(channel))))] = item
	}
	input.Channels = normalized
	event, err := u.repository.GetEvent(ctx, input.EventKey)
	if err != nil {
		return notifymodel.Policy{}, err
	}
	if err := notifymodel.ValidateReplacePolicy(scope, event, input); err != nil {
		return notifymodel.Policy{}, err
	}
	for channel, item := range input.Channels {
		if !item.Enabled {
			continue
		}
		template, err := u.repository.GetLibraryTemplate(ctx, item.TemplateCode)
		if err != nil {
			return notifymodel.Policy{}, err
		}
		if template.Channel != channel || !notifymodel.TemplateCoversEvent(template, event.Variables) {
			return notifymodel.Policy{}, notifymodel.ErrInvalid
		}
	}
	return u.repository.ReplacePolicy(ctx, scope, input, notifymodel.CommandHash(input))
}

func (u *UseCase) ListLibraryTemplates(ctx context.Context, filter notifymodel.TemplateFilter) ([]notifymodel.LibraryTemplate, error) {
	if u.repository == nil {
		return nil, notifymodel.ErrInvalid
	}
	filter.Channel = notifymodel.Channel(strings.ToUpper(strings.TrimSpace(string(filter.Channel))))
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if filter.Channel != "" && filter.Channel != notifymodel.ChannelSMS && filter.Channel != notifymodel.ChannelEmail && filter.Channel != notifymodel.ChannelInApp {
		return nil, notifymodel.ErrInvalid
	}
	return u.repository.ListLibraryTemplates(ctx, filter)
}

func (u *UseCase) GetLibraryTemplate(ctx context.Context, code string) (notifymodel.LibraryTemplate, error) {
	if u.repository == nil {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrInvalid
	}
	return u.repository.GetLibraryTemplate(ctx, strings.TrimSpace(code))
}

func (u *UseCase) UpsertLibraryTemplate(ctx context.Context, scope notifymodel.Scope, input notifymodel.UpsertLibraryTemplate) (notifymodel.LibraryTemplate, error) {
	if u.repository == nil {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrInvalid
	}
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Channel = notifymodel.Channel(strings.ToUpper(strings.TrimSpace(string(input.Channel))))
	input.TextTemplate = strings.TrimSpace(input.TextTemplate)
	input.Subject = strings.TrimSpace(input.Subject)
	input.BodyHTML = strings.TrimSpace(input.BodyHTML)
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	if len(input.Variables) == 0 {
		input.Variables = notifymodel.ExtractTemplateVariables(input.TextTemplate, input.Subject, input.BodyHTML, input.Title, input.Body)
	}
	if err := notifymodel.ValidateUpsertLibraryTemplate(scope, input); err != nil {
		return notifymodel.LibraryTemplate{}, err
	}
	return u.repository.UpsertLibraryTemplate(ctx, scope, input, notifymodel.CommandHash(input))
}

func (u *UseCase) RetireLibraryTemplate(ctx context.Context, scope notifymodel.Scope, input notifymodel.RetireLibraryTemplate) (notifymodel.LibraryTemplate, error) {
	if u.repository == nil {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrInvalid
	}
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	if err := notifymodel.ValidateRetireLibraryTemplate(scope, input); err != nil {
		return notifymodel.LibraryTemplate{}, err
	}
	return u.repository.RetireLibraryTemplate(ctx, scope, input, notifymodel.CommandHash(input))
}

func (u *UseCase) GetInAppConfig(ctx context.Context) (notifymodel.InAppConfig, error) {
	if u.repository == nil {
		return notifymodel.InAppConfig{}, notifymodel.ErrInvalid
	}
	return u.repository.GetInAppConfig(ctx)
}

func (u *UseCase) ReplaceInAppConfig(ctx context.Context, scope notifymodel.Scope, input notifymodel.ReplaceInAppConfig) (notifymodel.InAppConfig, error) {
	if u.repository == nil {
		return notifymodel.InAppConfig{}, notifymodel.ErrInvalid
	}
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	if err := notifymodel.ValidateReplaceInAppConfig(scope, input); err != nil {
		return notifymodel.InAppConfig{}, err
	}
	return u.repository.ReplaceInAppConfig(ctx, scope, input, notifymodel.CommandHash(input))
}

func (u *UseCase) ListDeliveries(ctx context.Context, eventKey string) ([]notifymodel.Delivery, error) {
	if u.repository == nil {
		return nil, notifymodel.ErrInvalid
	}
	if _, err := u.repository.GetEvent(ctx, strings.TrimSpace(eventKey)); err != nil {
		return nil, err
	}
	return u.repository.ListDeliveries(ctx, strings.TrimSpace(eventKey))
}

func (u *UseCase) GetDelivery(ctx context.Context, caller notifymodel.Caller, deliveryID string) (notifymodel.Delivery, error) {
	if u.repository == nil || !caller.Valid() {
		return notifymodel.Delivery{}, notifymodel.ErrInvalid
	}
	item, err := u.repository.GetDelivery(ctx, strings.TrimSpace(deliveryID))
	if err != nil {
		return notifymodel.Delivery{}, err
	}
	if notifymodel.EventPrefix(item.EventKey) != caller.ModuleID {
		return notifymodel.Delivery{}, notifymodel.ErrForbidden
	}
	return item, nil
}

func (u *UseCase) Dispatch(ctx context.Context, caller notifymodel.Caller, input notifymodel.DispatchInput) (notifymodel.DispatchResult, error) {
	if u.repository == nil {
		return notifymodel.DispatchResult{}, notifymodel.ErrInvalid
	}
	input.EventKey = strings.TrimSpace(input.EventKey)
	input.DeliveryKey = strings.TrimSpace(input.DeliveryKey)
	input.Locale = strings.TrimSpace(input.Locale)
	event, err := u.repository.GetEvent(ctx, input.EventKey)
	if err != nil {
		return notifymodel.DispatchResult{}, err
	}
	if err := notifymodel.ValidateDispatch(caller, event, input); err != nil {
		return notifymodel.DispatchResult{}, err
	}
	if event.Policy.DispatchMode == notifymodel.ModeScheduled && input.NotBefore.IsZero() {
		delay := time.Duration(event.Policy.DelaySeconds) * time.Second
		input.NotBefore = u.clock().Add(delay)
	}
	if event.Policy.DispatchMode == notifymodel.ModeScheduled && input.NotBefore.Before(u.clock()) {
		return notifymodel.DispatchResult{}, notifymodel.ErrInvalid
	}
	channels := notifymodel.EnabledChannels(event.Policy, event.AllowedChannels)
	results, pending, err := u.repository.PrepareDeliveries(ctx, input, event, channels, notifymodel.RequestHash(input))
	if err != nil {
		return notifymodel.DispatchResult{}, err
	}
	if event.Policy.DispatchMode == notifymodel.ModeSync {
		for index, item := range pending {
			if results[index].Deduped && terminal(item.Status) {
				continue
			}
			results[index] = u.sendOne(ctx, event, input, item)
		}
	}
	return notifymodel.DispatchResult{Deliveries: results}, nil
}

func (u *UseCase) RecoverDue(ctx context.Context) error {
	if u.repository == nil {
		return notifymodel.ErrInvalid
	}
	items, err := u.repository.ListDue(ctx, u.clock(), 50)
	if err != nil {
		return err
	}
	for _, item := range items {
		event, err := u.repository.GetEvent(ctx, item.EventKey)
		if err != nil {
			continue
		}
		u.sendOne(ctx, event, recipientsInput(item), item)
	}
	return nil
}

func (u *UseCase) RunWorker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	_ = u.RecoverDue(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = u.RecoverDue(ctx)
		}
	}
}

func (u *UseCase) sendOne(ctx context.Context, event notifymodel.Event, input notifymodel.DispatchInput, item notifymodel.Delivery) notifymodel.DeliveryResult {
	result := notifymodel.DeliveryResult{DeliveryID: item.DeliveryID, Channel: item.Channel, Status: item.Status}
	if terminal(item.Status) {
		return result
	}
	if err := u.repository.MarkSending(ctx, item.DeliveryID); err != nil {
		result.Status = notifymodel.StatusUnknown
		return result
	}
	status, detail, inbox := u.deliver(ctx, event, input, item)
	if err := u.repository.CompleteDelivery(ctx, item.DeliveryID, status, detail, inbox); err != nil {
		result.Status = notifymodel.StatusUnknown
		return result
	}
	result.Status = status
	return result
}

func (u *UseCase) deliver(ctx context.Context, event notifymodel.Event, input notifymodel.DispatchInput, item notifymodel.Delivery) (notifymodel.DeliveryStatus, string, *notifymodel.InboxMessage) {
	recipients := input.Recipients
	if strings.TrimSpace(recipients.Phone+recipients.Email+recipients.Subject) == "" {
		recipients = recipientsFrom(item)
	}
	recipient, ok := notifymodel.RecipientFor(item.Channel, recipients)
	if !ok {
		return notifymodel.StatusFailedPermanent, "recipient is missing or invalid", nil
	}
	code := notifymodel.TemplateCodeFor(event.Policy, item.Channel)
	template, err := u.repository.GetLibraryTemplate(ctx, code)
	if err != nil || !notifymodel.TemplateCoversEvent(template, event.Variables) {
		return notifymodel.StatusFailedPermanent, "template is missing", nil
	}
	if item.Channel == notifymodel.ChannelInApp {
		config, cfgErr := u.repository.GetInAppConfig(ctx)
		if cfgErr != nil || !config.Enabled {
			return notifymodel.StatusFailedPermanent, "in-app channel is disabled", nil
		}
	}
	switch item.Channel {
	case notifymodel.ChannelSMS:
		if u.sender == nil {
			return notifymodel.StatusFailedPermanent, "sms sender is unavailable", nil
		}
		sent, sendErr := u.sender.SendSMS(ctx, recipient, notifymodel.Render(template.TextTemplate, input.Variables))
		return sendStatus(sent, sendErr), sent.Detail, nil
	case notifymodel.ChannelEmail:
		if u.sender == nil {
			return notifymodel.StatusFailedPermanent, "email sender is unavailable", nil
		}
		sent, sendErr := u.sender.SendEmail(ctx, recipient, notifymodel.Render(template.Subject, input.Variables), notifymodel.Render(template.BodyHTML, input.Variables))
		return sendStatus(sent, sendErr), sent.Detail, nil
	case notifymodel.ChannelInApp:
		return notifymodel.StatusSent, "inbox written", &notifymodel.InboxMessage{
			MerchantID: item.MerchantID, ShopID: item.ShopID, Subject: recipient, DeliveryID: item.DeliveryID,
			Title: notifymodel.Render(template.Title, input.Variables), Body: notifymodel.Render(template.Body, input.Variables),
		}
	default:
		return notifymodel.StatusFailedPermanent, "unsupported channel", nil
	}
}

func (u *UseCase) clock() time.Time {
	if u.now == nil {
		return time.Now()
	}
	return u.now()
}

func sendStatus(result notifymodel.ChannelSendResult, err error) notifymodel.DeliveryStatus {
	if result.Unknown || (err != nil && isUnknown(err)) {
		return notifymodel.StatusUnknown
	}
	if err != nil {
		return notifymodel.StatusFailedPermanent
	}
	return notifymodel.StatusSent
}

func isUnknown(err error) bool {
	return err != nil && (strings.Contains(strings.ToLower(err.Error()), "timeout") || strings.Contains(strings.ToLower(err.Error()), "deadline"))
}

func terminal(status notifymodel.DeliveryStatus) bool {
	return status == notifymodel.StatusSent || status == notifymodel.StatusFailedPermanent
}

func recipientsInput(item notifymodel.Delivery) notifymodel.DispatchInput {
	return notifymodel.DispatchInput{EventKey: item.EventKey, DeliveryKey: item.DeliveryKey, MerchantID: item.MerchantID, ShopID: item.ShopID, Recipients: recipientsFrom(item), Variables: item.Variables}
}

func recipientsFrom(item notifymodel.Delivery) notifymodel.Recipients {
	switch item.Channel {
	case notifymodel.ChannelSMS:
		return notifymodel.Recipients{Phone: item.Recipient}
	case notifymodel.ChannelEmail:
		return notifymodel.Recipients{Email: item.Recipient}
	default:
		return notifymodel.Recipients{Subject: item.Recipient}
	}
}
