package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/lvtuopen-ai/kernel-go/apperror"
)

type Channel string
type DispatchMode string
type DeliveryStatus string

const (
	ChannelSMS   Channel = "SMS"
	ChannelEmail Channel = "EMAIL"
	ChannelInApp Channel = "IN_APP"

	ModeSync      DispatchMode = "SYNC"
	ModeAsync     DispatchMode = "ASYNC"
	ModeScheduled DispatchMode = "SCHEDULED"

	StatusPending         DeliveryStatus = "PENDING"
	StatusScheduled       DeliveryStatus = "SCHEDULED"
	StatusSending         DeliveryStatus = "SENDING"
	StatusSent            DeliveryStatus = "SENT"
	StatusFailedPermanent DeliveryStatus = "FAILED_PERMANENT"
	StatusUnknown         DeliveryStatus = "UNKNOWN"

	MaxDelaySeconds = 2592000
	MaxAttempts     = 5
)

var (
	ErrInvalid   = apperror.New("platform.notify.invalid", "notification input is invalid")
	ErrNotFound  = apperror.New("platform.notify.not_found", "notification resource was not found")
	ErrConflict  = apperror.New("platform.notify.conflict", "notification version, command or delivery conflicts")
	ErrForbidden = apperror.New("platform.notify.forbidden", "caller cannot dispatch this event")
	ErrUnknown   = apperror.New("platform.notify.unknown", "provider result is unknown")

	eventKeyPattern     = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-zA-Z0-9]*){2,}$`)
	variablePattern     = regexp.MustCompile(`^[a-z][a-zA-Z0-9]{0,31}$`)
	templateVarPattern  = regexp.MustCompile(`\{\{([a-z][a-zA-Z0-9]{0,31})\}\}`)
	deliveryKeyPattern  = regexp.MustCompile(`^[A-Za-z0-9._:/-]{8,128}$`)
	templateCodePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,63}$`)
)

type Scope struct {
	Realm      string
	MerchantID int64
	Subject    string
}

func (s Scope) Valid() bool {
	return strings.TrimSpace(s.Realm) != "" && s.MerchantID >= 0 && strings.TrimSpace(s.Subject) != ""
}

type Caller struct {
	ModuleID string
	Subject  string
}

func (c Caller) Valid() bool {
	return strings.TrimSpace(c.ModuleID) != "" && strings.TrimSpace(c.Subject) != ""
}

type Declaration struct {
	EventKey        string
	ModuleID        string
	ModuleName      string
	OperationID     string
	Title           string
	Variables       []string
	AllowedChannels []Channel
	DefaultDispatch DispatchMode
}

type Event struct {
	EventKey         string
	ModuleID         string
	ModuleName       string
	OperationID      string
	Title            string
	Variables        []string
	AllowedChannels  []Channel
	DefaultDispatch  DispatchMode
	Dispatchable     bool
	RegistryRevision uint64
	Policy           Policy
	UpdatedAt        time.Time
}

type ChannelPolicy struct {
	Enabled      bool   `json:"enabled"`
	TemplateCode string `json:"templateCode,omitempty"`
}

type Policy struct {
	EventKey     string
	DispatchMode DispatchMode
	DelaySeconds int
	Channels     map[Channel]ChannelPolicy
	Version      int64
	UpdatedAt    time.Time
}

type LibraryTemplate struct {
	Code         string
	Channel      Channel
	TextTemplate string
	Subject      string
	BodyHTML     string
	Title        string
	Body         string
	Variables    []string
	Lifecycle    string
	Version      int64
	UpdatedAt    time.Time
}

type InAppConfig struct {
	Driver    string
	Enabled   bool
	Version   int64
	UpdatedAt time.Time
}

const (
	TemplateActive  = "ACTIVE"
	TemplateRetired = "RETIRED"
	InAppDriver     = "inbox"
)

type Recipients struct {
	Phone   string
	Email   string
	Subject string
}

type DispatchInput struct {
	EventKey    string
	DeliveryKey string
	MerchantID  int64
	ShopID      int64
	Recipients  Recipients
	Variables   map[string]string
	NotBefore   time.Time
	Locale      string
}

type DeliveryResult struct {
	DeliveryID string
	Channel    Channel
	Status     DeliveryStatus
	Deduped    bool
}

type DispatchResult struct {
	Deliveries []DeliveryResult
}

type Delivery struct {
	DeliveryID   string
	DeliveryKey  string
	EventKey     string
	Channel      Channel
	MerchantID   int64
	ShopID       int64
	Status       DeliveryStatus
	Recipient    string
	Variables    map[string]string
	RequestHash  string
	NotBefore    time.Time
	LastError    string
	AttemptCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type EventFilter struct {
	Module  string
	Channel Channel
	Keyword string
}

type ReplacePolicy struct {
	EventKey        string
	CommandKey      string
	ExpectedVersion int64
	DispatchMode    DispatchMode
	DelaySeconds    int
	Channels        map[Channel]ChannelPolicy
}

type UpsertLibraryTemplate struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
	Channel         Channel
	TextTemplate    string
	Subject         string
	BodyHTML        string
	Title           string
	Body            string
	Variables       []string
}

type RetireLibraryTemplate struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
}

type ReplaceInAppConfig struct {
	CommandKey      string
	ExpectedVersion int64
	Enabled         bool
}

type TemplateFilter struct {
	Channel Channel
	Keyword string
}

type ChannelSendResult struct {
	Detail  string
	Unknown bool
}

type InboxMessage struct {
	MerchantID int64
	ShopID     int64
	Subject    string
	DeliveryID string
	Title      string
	Body       string
}

func DeclarationsFromBackend(moduleID, moduleName string, backend modulemanifest.Backend) []Declaration {
	output := make([]Declaration, 0)
	for _, route := range backend.HTTPRoutes {
		for _, operation := range route.Operations {
			for _, item := range operation.Notifications {
				channels := make([]Channel, 0, len(item.AllowedChannels))
				for _, channel := range item.AllowedChannels {
					channels = append(channels, Channel(channel))
				}
				output = append(output, Declaration{
					EventKey: item.EventKey, ModuleID: moduleID, ModuleName: moduleName,
					OperationID: operation.ID, Title: item.Title, Variables: append([]string{}, item.Variables...),
					AllowedChannels: channels, DefaultDispatch: DispatchMode(item.DefaultDispatch),
				})
			}
		}
	}
	sort.Slice(output, func(i, j int) bool { return output[i].EventKey < output[j].EventKey })
	return output
}

func DefaultPolicy(declaration Declaration) Policy {
	channels := map[Channel]ChannelPolicy{}
	for _, channel := range declaration.AllowedChannels {
		channels[channel] = ChannelPolicy{Enabled: true}
	}
	return Policy{EventKey: declaration.EventKey, DispatchMode: declaration.DefaultDispatch, Channels: channels, Version: 1}
}

func EnabledChannels(policy Policy, allowed []Channel) []Channel {
	allowedSet := map[Channel]bool{}
	for _, channel := range allowed {
		allowedSet[channel] = true
	}
	output := make([]Channel, 0)
	for _, channel := range []Channel{ChannelSMS, ChannelEmail, ChannelInApp} {
		if policy.Channels[channel].Enabled && allowedSet[channel] {
			output = append(output, channel)
		}
	}
	return output
}

func ChannelsWithRecipient(channels []Channel, recipients Recipients) []Channel {
	output := make([]Channel, 0, len(channels))
	for _, channel := range channels {
		if _, ok := RecipientFor(channel, recipients); ok {
			output = append(output, channel)
		}
	}
	return output
}

func ConventionalTemplateCode(eventKey string, channel Channel) string {
	return strings.ToLower(strings.TrimSpace(eventKey)) + "." + strings.ToLower(string(channel))
}

func BindEmptyPolicyTemplates(policy *Policy, allowed []Channel, lookup func(code string) (LibraryTemplate, bool)) {
	if policy == nil {
		return
	}
	if policy.Channels == nil {
		policy.Channels = map[Channel]ChannelPolicy{}
	}
	for _, channel := range allowed {
		current := policy.Channels[channel]
		if strings.TrimSpace(current.TemplateCode) != "" {
			continue
		}
		code := ConventionalTemplateCode(policy.EventKey, channel)
		item, ok := lookup(code)
		if !ok || item.Lifecycle != TemplateActive || item.Channel != channel {
			continue
		}
		current.TemplateCode = code
		policy.Channels[channel] = current
	}
}

func TemplateCodeFor(policy Policy, channel Channel) string {
	return strings.TrimSpace(policy.Channels[channel].TemplateCode)
}

func EventPrefix(eventKey string) string {
	eventKey = strings.TrimSpace(eventKey)
	dot := strings.IndexByte(eventKey, '.')
	if dot <= 0 {
		return ""
	}
	return eventKey[:dot]
}

func ModuleIDFromWorkload(subject, spiffeID string) string {
	subject = strings.TrimSpace(subject)
	spiffeID = strings.TrimSpace(spiffeID)
	for _, moduleID := range []string{"identity", "catalog", "trade", "live"} {
		if subject == moduleID || strings.HasSuffix(subject, "-"+moduleID) || strings.Contains(spiffeID, "/"+moduleID) {
			return moduleID
		}
	}
	if i := strings.LastIndex(spiffeID, "/"); i >= 0 && i+1 < len(spiffeID) {
		return spiffeID[i+1:]
	}
	return subject
}

func ValidateDispatch(caller Caller, event Event, input DispatchInput) error {
	if !caller.Valid() || !eventKeyPattern.MatchString(input.EventKey) || !deliveryKeyPattern.MatchString(strings.TrimSpace(input.DeliveryKey)) {
		return ErrInvalid
	}
	if !event.Dispatchable || event.EventKey != input.EventKey {
		return ErrNotFound
	}
	if caller.ModuleID != EventPrefix(event.EventKey) || caller.ModuleID != event.ModuleID {
		return ErrForbidden
	}
	if input.MerchantID < 0 || input.ShopID < 0 {
		return ErrInvalid
	}
	if !sameVariables(event.Variables, input.Variables) {
		return ErrInvalid
	}
	if event.Policy.DispatchMode != ModeScheduled && !input.NotBefore.IsZero() {
		return ErrInvalid
	}
	return nil
}

func ValidateReplacePolicy(scope Scope, event Event, input ReplacePolicy) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || input.ExpectedVersion < 0 || event.EventKey == "" {
		return ErrInvalid
	}
	if !validMode(input.DispatchMode) {
		return ErrInvalid
	}
	if input.DispatchMode == ModeScheduled {
		if input.DelaySeconds < 0 || input.DelaySeconds > MaxDelaySeconds {
			return ErrInvalid
		}
	} else if input.DelaySeconds != 0 {
		return ErrInvalid
	}
	allowed := map[Channel]bool{}
	for _, channel := range event.AllowedChannels {
		allowed[channel] = true
	}
	if len(input.Channels) == 0 {
		return ErrInvalid
	}
	for channel, item := range input.Channels {
		if !allowed[channel] {
			return ErrInvalid
		}
		code := strings.TrimSpace(item.TemplateCode)
		if item.Enabled {
			if !templateCodePattern.MatchString(code) {
				return ErrInvalid
			}
		} else if code != "" && !templateCodePattern.MatchString(code) {
			return ErrInvalid
		}
	}
	return nil
}

func ValidateUpsertLibraryTemplate(scope Scope, input UpsertLibraryTemplate) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || input.ExpectedVersion < 0 || !templateCodePattern.MatchString(strings.TrimSpace(input.Code)) {
		return ErrInvalid
	}
	declared := map[string]bool{}
	for _, variable := range input.Variables {
		if !variablePattern.MatchString(variable) {
			return ErrInvalid
		}
		declared[variable] = true
	}
	extracted := ExtractTemplateVariables(input.TextTemplate, input.Subject, input.BodyHTML, input.Title, input.Body)
	for _, name := range extracted {
		if len(declared) > 0 && !declared[name] {
			return ErrInvalid
		}
	}
	switch input.Channel {
	case ChannelSMS:
		if strings.TrimSpace(input.TextTemplate) == "" {
			return ErrInvalid
		}
	case ChannelEmail:
		if strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.BodyHTML) == "" {
			return ErrInvalid
		}
	case ChannelInApp:
		if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Body) == "" {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

func ValidateRetireLibraryTemplate(scope Scope, input RetireLibraryTemplate) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || input.ExpectedVersion < 1 || !templateCodePattern.MatchString(strings.TrimSpace(input.Code)) {
		return ErrInvalid
	}
	return nil
}

func ValidateReplaceInAppConfig(scope Scope, input ReplaceInAppConfig) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || input.ExpectedVersion < 0 {
		return ErrInvalid
	}
	return nil
}

func TemplateCoversEvent(item LibraryTemplate, eventVariables []string) bool {
	if item.Lifecycle != TemplateActive || !LibraryTemplateReady(item, item.Channel) {
		return false
	}
	declared := map[string]bool{}
	for _, name := range eventVariables {
		declared[name] = true
	}
	for _, name := range item.Variables {
		if !declared[name] {
			return false
		}
	}
	return templateUsesDeclared(strings.Join([]string{item.TextTemplate, item.Subject, item.BodyHTML, item.Title, item.Body}, " "), declared)
}

func PolicyTemplateError(item LibraryTemplate, channel Channel, event Event) error {
	if item.Channel != channel {
		return apperror.Wrap(ErrInvalid, ErrInvalid.Reason, "template "+item.Code+" is channel "+string(item.Channel)+", not "+string(channel), "templateCode", item.Code, "channel", string(channel))
	}
	if !TemplateCoversEvent(item, event.Variables) {
		return apperror.Wrap(ErrInvalid, ErrInvalid.Reason, "template "+item.Code+" placeholders must be a subset of "+event.EventKey+" variables ("+strings.Join(event.Variables, ", ")+")", "templateCode", item.Code, "eventKey", event.EventKey)
	}
	return nil
}

func Render(template string, variables map[string]string) string {
	return templateVarPattern.ReplaceAllStringFunc(template, func(match string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}")
		return variables[name]
	})
}

func ExtractTemplateVariables(parts ...string) []string {
	seen := map[string]bool{}
	output := make([]string, 0)
	for _, part := range parts {
		for _, match := range templateVarPattern.FindAllStringSubmatch(part, -1) {
			if seen[match[1]] {
				continue
			}
			seen[match[1]] = true
			output = append(output, match[1])
		}
	}
	return output
}

func LibraryTemplateReady(item LibraryTemplate, channel Channel) bool {
	if item.Channel != channel {
		return false
	}
	switch channel {
	case ChannelSMS:
		return strings.TrimSpace(item.TextTemplate) != ""
	case ChannelEmail:
		return strings.TrimSpace(item.Subject) != "" && strings.TrimSpace(item.BodyHTML) != ""
	case ChannelInApp:
		return strings.TrimSpace(item.Title) != "" && strings.TrimSpace(item.Body) != ""
	default:
		return false
	}
}

func RecipientFor(channel Channel, recipients Recipients) (string, bool) {
	switch channel {
	case ChannelSMS:
		phone := strings.TrimSpace(recipients.Phone)
		return phone, validPhone(phone)
	case ChannelEmail:
		email := strings.TrimSpace(recipients.Email)
		return email, validEmail(email)
	case ChannelInApp:
		subject := strings.TrimSpace(recipients.Subject)
		return subject, subject != ""
	default:
		return "", false
	}
}

func RequestHash(input DispatchInput) string {
	payload := struct {
		EventKey    string            `json:"eventKey"`
		DeliveryKey string            `json:"deliveryKey"`
		MerchantID  int64             `json:"merchantId"`
		ShopID      int64             `json:"shopId"`
		Recipients  Recipients        `json:"recipients"`
		Variables   map[string]string `json:"variables"`
		NotBefore   string            `json:"notBefore,omitempty"`
		Locale      string            `json:"locale,omitempty"`
	}{
		EventKey: strings.TrimSpace(input.EventKey), DeliveryKey: strings.TrimSpace(input.DeliveryKey),
		MerchantID: input.MerchantID, ShopID: input.ShopID,
		Recipients: Recipients{Phone: strings.TrimSpace(input.Recipients.Phone), Email: strings.TrimSpace(input.Recipients.Email), Subject: strings.TrimSpace(input.Recipients.Subject)},
		Variables:  input.Variables, Locale: strings.TrimSpace(input.Locale),
	}
	if !input.NotBefore.IsZero() {
		payload.NotBefore = input.NotBefore.UTC().Format(time.RFC3339Nano)
	}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func CommandHash(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func sameVariables(declared []string, provided map[string]string) bool {
	if len(declared) != len(provided) {
		return false
	}
	for _, name := range declared {
		value, ok := provided[name]
		if !ok || strings.TrimSpace(value) == "" || !variablePattern.MatchString(name) {
			return false
		}
	}
	return true
}

func templateUsesDeclared(template string, declared map[string]bool) bool {
	matches := templateVarPattern.FindAllStringSubmatch(template, -1)
	for _, match := range matches {
		if !declared[match[1]] {
			return false
		}
	}
	return true
}

func validMode(mode DispatchMode) bool {
	return mode == ModeSync || mode == ModeAsync || mode == ModeScheduled
}

func validCommandKey(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 64
}

func validPhone(phone string) bool {
	if !strings.HasPrefix(phone, "+") || len(phone) < 8 {
		return false
	}
	for _, item := range phone[1:] {
		if item < '0' || item > '9' {
			return false
		}
	}
	return true
}

func validEmail(address string) bool {
	at := strings.LastIndex(address, "@")
	if at < 1 || at >= len(address)-3 {
		return false
	}
	return strings.Contains(address[at+1:], ".") && !strings.ContainsAny(address, " \t\r\n,;")
}
