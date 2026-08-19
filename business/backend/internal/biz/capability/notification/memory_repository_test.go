package notification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
)

type memoryRepository struct {
	mu         sync.Mutex
	revision   uint64
	events     map[string]notifymodel.Event
	templates  map[string]notifymodel.LibraryTemplate
	inApp      notifymodel.InAppConfig
	deliveries map[string]notifymodel.Delivery
	byKey      map[string]string
	commands   map[string]commandReceipt
	attempts   map[string]int
}

type commandReceipt struct {
	hash    string
	kind    string
	payload any
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		events:     map[string]notifymodel.Event{},
		templates:  map[string]notifymodel.LibraryTemplate{},
		inApp:      notifymodel.InAppConfig{Driver: notifymodel.InAppDriver, Enabled: true, Version: 1, UpdatedAt: time.Now().UTC()},
		deliveries: map[string]notifymodel.Delivery{},
		byKey:      map[string]string{},
		commands:   map[string]commandReceipt{},
		attempts:   map[string]int{},
	}
}

func (r *memoryRepository) Project(_ context.Context, revision uint64, declarations []notifymodel.Declaration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revision = revision
	seen := map[string]bool{}
	now := time.Now().UTC()
	for _, declaration := range declarations {
		seen[declaration.EventKey] = true
		current, exists := r.events[declaration.EventKey]
		policy := current.Policy
		if !exists || policy.Version == 0 {
			policy = notifymodel.DefaultPolicy(declaration)
			notifymodel.BindEmptyPolicyTemplates(&policy, declaration.AllowedChannels, func(code string) (notifymodel.LibraryTemplate, bool) {
				item, ok := r.templates[code]
				return item, ok
			})
			policy.UpdatedAt = now
		}
		r.events[declaration.EventKey] = notifymodel.Event{
			EventKey: declaration.EventKey, ModuleID: declaration.ModuleID, ModuleName: declaration.ModuleName,
			OperationID: declaration.OperationID, Title: declaration.Title, Variables: append([]string{}, declaration.Variables...),
			AllowedChannels: append([]notifymodel.Channel{}, declaration.AllowedChannels...), DefaultDispatch: declaration.DefaultDispatch,
			Dispatchable: true, RegistryRevision: revision, Policy: policy, UpdatedAt: now,
		}
	}
	for key, event := range r.events {
		if !seen[key] {
			event.Dispatchable = false
			event.RegistryRevision = revision
			r.events[key] = event
		}
	}
	return nil
}

func (r *memoryRepository) ListEvents(_ context.Context, filter notifymodel.EventFilter) ([]notifymodel.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	output := make([]notifymodel.Event, 0)
	keyword := strings.ToLower(filter.Keyword)
	for _, item := range r.events {
		if !item.Dispatchable {
			continue
		}
		if filter.Module != "" && item.ModuleID != filter.Module {
			continue
		}
		if filter.Channel != "" && !item.Policy.Channels[filter.Channel].Enabled {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.Title+item.EventKey+item.OperationID), keyword) {
			continue
		}
		output = append(output, item)
	}
	return output, nil
}

func (r *memoryRepository) GetEvent(_ context.Context, eventKey string) (notifymodel.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.events[eventKey]
	if !ok {
		return notifymodel.Event{}, notifymodel.ErrNotFound
	}
	return item, nil
}

func (r *memoryRepository) ReplacePolicy(_ context.Context, _ notifymodel.Scope, input notifymodel.ReplacePolicy, requestHash string) (notifymodel.Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if replay, ok := r.commands[input.CommandKey]; ok {
		if replay.hash != requestHash {
			return notifymodel.Policy{}, notifymodel.ErrConflict
		}
		return replay.payload.(notifymodel.Policy), nil
	}
	event, ok := r.events[input.EventKey]
	if !ok {
		return notifymodel.Policy{}, notifymodel.ErrNotFound
	}
	if event.Policy.Version != input.ExpectedVersion {
		return notifymodel.Policy{}, notifymodel.ErrConflict
	}
	policy := notifymodel.Policy{EventKey: input.EventKey, DispatchMode: input.DispatchMode, DelaySeconds: input.DelaySeconds, Channels: cloneChannelPolicy(input.Channels), Version: event.Policy.Version + 1, UpdatedAt: time.Now().UTC()}
	event.Policy = policy
	event.UpdatedAt = policy.UpdatedAt
	r.events[input.EventKey] = event
	r.commands[input.CommandKey] = commandReceipt{hash: requestHash, kind: "POLICY", payload: policy}
	return policy, nil
}

func (r *memoryRepository) ListLibraryTemplates(_ context.Context, filter notifymodel.TemplateFilter) ([]notifymodel.LibraryTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	output := make([]notifymodel.LibraryTemplate, 0)
	keyword := strings.ToLower(filter.Keyword)
	for _, item := range r.templates {
		if filter.Channel != "" && item.Channel != filter.Channel {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.Code+item.Title+item.Subject), keyword) {
			continue
		}
		output = append(output, item)
	}
	return output, nil
}

func (r *memoryRepository) GetLibraryTemplate(_ context.Context, code string) (notifymodel.LibraryTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.templates[code]
	if !ok {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrNotFound
	}
	return item, nil
}

func (r *memoryRepository) UpsertLibraryTemplate(_ context.Context, _ notifymodel.Scope, input notifymodel.UpsertLibraryTemplate, requestHash string) (notifymodel.LibraryTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if replay, ok := r.commands[input.CommandKey]; ok {
		if replay.hash != requestHash {
			return notifymodel.LibraryTemplate{}, notifymodel.ErrConflict
		}
		return replay.payload.(notifymodel.LibraryTemplate), nil
	}
	current := r.templates[input.Code]
	if current.Version != input.ExpectedVersion {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrConflict
	}
	if current.Lifecycle == notifymodel.TemplateRetired {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrConflict
	}
	item := notifymodel.LibraryTemplate{
		Code: input.Code, Channel: input.Channel, TextTemplate: input.TextTemplate, Subject: input.Subject,
		BodyHTML: input.BodyHTML, Title: input.Title, Body: input.Body, Variables: append([]string{}, input.Variables...),
		Lifecycle: notifymodel.TemplateActive, Version: current.Version + 1, UpdatedAt: time.Now().UTC(),
	}
	r.templates[input.Code] = item
	r.commands[input.CommandKey] = commandReceipt{hash: requestHash, kind: "LIBRARY", payload: item}
	return item, nil
}

func (r *memoryRepository) RetireLibraryTemplate(_ context.Context, _ notifymodel.Scope, input notifymodel.RetireLibraryTemplate, requestHash string) (notifymodel.LibraryTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if replay, ok := r.commands[input.CommandKey]; ok {
		if replay.hash != requestHash {
			return notifymodel.LibraryTemplate{}, notifymodel.ErrConflict
		}
		return replay.payload.(notifymodel.LibraryTemplate), nil
	}
	current, ok := r.templates[input.Code]
	if !ok {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrNotFound
	}
	if current.Version != input.ExpectedVersion {
		return notifymodel.LibraryTemplate{}, notifymodel.ErrConflict
	}
	current.Lifecycle = notifymodel.TemplateRetired
	current.Version++
	current.UpdatedAt = time.Now().UTC()
	r.templates[input.Code] = current
	r.commands[input.CommandKey] = commandReceipt{hash: requestHash, kind: "RETIRE", payload: current}
	return current, nil
}

func (r *memoryRepository) GetInAppConfig(_ context.Context) (notifymodel.InAppConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inApp, nil
}

func (r *memoryRepository) ReplaceInAppConfig(_ context.Context, _ notifymodel.Scope, input notifymodel.ReplaceInAppConfig, requestHash string) (notifymodel.InAppConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if replay, ok := r.commands[input.CommandKey]; ok {
		if replay.hash != requestHash {
			return notifymodel.InAppConfig{}, notifymodel.ErrConflict
		}
		return replay.payload.(notifymodel.InAppConfig), nil
	}
	if r.inApp.Version != input.ExpectedVersion {
		return notifymodel.InAppConfig{}, notifymodel.ErrConflict
	}
	r.inApp.Enabled = input.Enabled
	r.inApp.Driver = notifymodel.InAppDriver
	r.inApp.Version++
	r.inApp.UpdatedAt = time.Now().UTC()
	r.commands[input.CommandKey] = commandReceipt{hash: requestHash, kind: "INAPP", payload: r.inApp}
	return r.inApp, nil
}

func cloneChannelPolicy(input map[notifymodel.Channel]notifymodel.ChannelPolicy) map[notifymodel.Channel]notifymodel.ChannelPolicy {
	output := make(map[notifymodel.Channel]notifymodel.ChannelPolicy, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func (r *memoryRepository) ListDeliveries(_ context.Context, eventKey string) ([]notifymodel.Delivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	output := make([]notifymodel.Delivery, 0)
	for _, item := range r.deliveries {
		if item.EventKey == eventKey {
			output = append(output, item)
		}
	}
	return output, nil
}

func (r *memoryRepository) GetDelivery(_ context.Context, deliveryID string) (notifymodel.Delivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.deliveries[deliveryID]
	if !ok {
		return notifymodel.Delivery{}, notifymodel.ErrNotFound
	}
	return item, nil
}

func (r *memoryRepository) PrepareDeliveries(_ context.Context, input notifymodel.DispatchInput, event notifymodel.Event, channels []notifymodel.Channel, requestHash string) ([]notifymodel.DeliveryResult, []notifymodel.Delivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	results := make([]notifymodel.DeliveryResult, 0, len(channels))
	pending := make([]notifymodel.Delivery, 0, len(channels))
	now := time.Now().UTC()
	for _, channel := range channels {
		key := input.DeliveryKey + "/" + string(channel)
		if existingID, ok := r.byKey[key]; ok {
			current := r.deliveries[existingID]
			if current.RequestHash != requestHash {
				return nil, nil, notifymodel.ErrConflict
			}
			results = append(results, notifymodel.DeliveryResult{DeliveryID: current.DeliveryID, Channel: channel, Status: current.Status, Deduped: true})
			pending = append(pending, current)
			continue
		}
		status := notifymodel.StatusPending
		if event.Policy.DispatchMode == notifymodel.ModeScheduled {
			status = notifymodel.StatusScheduled
		}
		if event.Policy.DispatchMode == notifymodel.ModeSync {
			status = notifymodel.StatusSending
		}
		recipient, _ := notifymodel.RecipientFor(channel, input.Recipients)
		item := notifymodel.Delivery{
			DeliveryID: newID(), DeliveryKey: input.DeliveryKey, EventKey: event.EventKey, Channel: channel,
			MerchantID: input.MerchantID, ShopID: input.ShopID, Status: status, Recipient: recipient,
			Variables: cloneVars(input.Variables), RequestHash: requestHash, NotBefore: input.NotBefore, CreatedAt: now, UpdatedAt: now,
		}
		r.deliveries[item.DeliveryID] = item
		r.byKey[key] = item.DeliveryID
		results = append(results, notifymodel.DeliveryResult{DeliveryID: item.DeliveryID, Channel: channel, Status: status})
		pending = append(pending, item)
	}
	return results, pending, nil
}

func (r *memoryRepository) MarkSending(_ context.Context, deliveryID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.deliveries[deliveryID]
	if !ok {
		return notifymodel.ErrNotFound
	}
	if terminalMemory(item.Status) {
		return nil
	}
	item.Status = notifymodel.StatusSending
	item.UpdatedAt = time.Now().UTC()
	r.deliveries[deliveryID] = item
	return nil
}

func (r *memoryRepository) CompleteDelivery(_ context.Context, deliveryID string, status notifymodel.DeliveryStatus, detail string, _ *notifymodel.InboxMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.deliveries[deliveryID]
	if !ok {
		return notifymodel.ErrNotFound
	}
	item.Status = status
	item.LastError = detail
	item.AttemptCount++
	item.UpdatedAt = time.Now().UTC()
	r.deliveries[deliveryID] = item
	r.attempts[deliveryID] = item.AttemptCount
	return nil
}

func (r *memoryRepository) ListDue(_ context.Context, now time.Time, limit int) ([]notifymodel.Delivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	output := make([]notifymodel.Delivery, 0)
	for _, item := range r.deliveries {
		due := item.Status == notifymodel.StatusPending || item.Status == notifymodel.StatusUnknown || (item.Status == notifymodel.StatusScheduled && (item.NotBefore.IsZero() || !item.NotBefore.After(now)))
		if due && item.AttemptCount < notifymodel.MaxAttempts {
			output = append(output, item)
			if len(output) >= limit {
				break
			}
		}
	}
	return output, nil
}

func newID() string {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}

func cloneVars(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func terminalMemory(status notifymodel.DeliveryStatus) bool {
	return status == notifymodel.StatusSent || status == notifymodel.StatusFailedPermanent
}

var _ Repository = (*memoryRepository)(nil)
