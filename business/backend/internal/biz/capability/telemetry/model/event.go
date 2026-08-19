package model

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lvtuopen-ai/kernel-go/apperror"
)

const (
	SurfaceShop  = "shop"
	SurfaceMerch = "merch"
	SurfaceAdmin = "admin"
	SurfaceLive  = "live"

	MaxEvents       = 100
	MaxBodyBytes    = 512 << 10
	MaxJSONBytes    = 16 << 10
	DefaultPageSize = 20
	MaxPageSize     = 100

	timeSkewFuture = 10 * time.Minute
	timeSkewPast   = 7 * 24 * time.Hour
)

var (
	ErrInvalid   = apperror.New("platform.telemetry.invalid", "telemetry input is invalid")
	ErrForbidden = apperror.New("platform.telemetry.forbidden", "shop context is required")

	coreTrackEvents = map[string]bool{
		"session_enter": true, "session_ping": true, "session_exit": true,
		"page_enter": true, "page_meta": true, "page_exit": true, "page_view": true,
		"product_view": true, "product_card_exposure": true, "product_card_click": true,
		"add_to_cart": true, "payment_attempt": true, "payment_succeeded": true, "order_create": true,
		"ad_touch":   true,
		"live_enter": true, "live_ping": true, "live_exit": true, "live_play_result": true,
	}
)

type Scope struct {
	MerchantID  int64
	ShopID      int64
	Surface     string
	Subject     string
	UserAgent   string
	IP          string
	Referer     string
	AdTouchID   int64
	ClickIDType string
	BodyBytes   int64
	Now         time.Time
}

func (s Scope) Valid() bool {
	return s.MerchantID > 0 && s.ShopID > 0 && ValidSurface(s.Surface)
}

type EventInput struct {
	EventID       string
	EventName     string
	EventType     string
	Page          string
	Component     string
	Action        string
	BizType       string
	BizID         string
	SessionID     string
	AnonymousID   string
	OccurredAtMs  int64
	ClientTs      int64
	SchemaVersion int
	LiveContext   map[string]any
	Props         map[string]any
	State         map[string]any
	Extra         map[string]any
	MerchantID    int64
	ShopID        int64
	App           string
	AppID         int64
	CommercialID  int64
	UID           int64
}

type Event struct {
	MerchantID    int64
	ShopID        int64
	Surface       string
	EventID       string
	EventType     string
	EventName     string
	Page          string
	Component     string
	Action        string
	BizType       string
	BizID         string
	SessionID     string
	AnonymousID   string
	Subject       string
	ClientTs      int64
	OccurredAt    time.Time
	ReceivedAt    time.Time
	SchemaVersion int
	LiveContext   json.RawMessage
	Props         json.RawMessage
	State         json.RawMessage
	Extra         json.RawMessage
	UserAgent     string
	IP            string
	Referer       string
	AdTouchID     int64
	ClickIDType   string
	CreatedAt     time.Time
}

type Filter struct {
	MerchantID  int64
	ShopID      int64
	Surface     string
	EventName   string
	EventType   string
	Subject     string
	AnonymousID string
	StartMs     int64
	EndMs       int64
	Page        int
	PageSize    int
}

type Page struct {
	Items []Event
	Total int
}

type IngestResult struct {
	Accepted   int
	Duplicates int
	Rejected   int
	Errors     []ItemError
	Stored     []Event
}

type ItemError struct {
	Index   int
	EventID string
	Code    string
	Message string
}

type RowError struct {
	Code    string
	Message string
}

func (e *RowError) Error() string { return e.Message }

func NormalizeFilter(filter Filter) Filter {
	filter.Surface = strings.TrimSpace(filter.Surface)
	filter.EventName = strings.TrimSpace(filter.EventName)
	filter.EventType = strings.TrimSpace(filter.EventType)
	filter.Subject = strings.TrimSpace(filter.Subject)
	filter.AnonymousID = strings.TrimSpace(filter.AnonymousID)
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > MaxPageSize {
		filter.PageSize = DefaultPageSize
	}
	return filter
}

func (f Filter) Offset() int { return (f.Page - 1) * f.PageSize }

func ValidateScope(scope Scope) error {
	if scope.MerchantID <= 0 || scope.ShopID <= 0 {
		return ErrForbidden
	}
	if !ValidSurface(scope.Surface) {
		return ErrInvalid
	}
	if scope.BodyBytes > MaxBodyBytes {
		return apperror.Wrap(ErrInvalid, ErrInvalid.Reason, "request body cannot exceed 512 KiB")
	}
	return nil
}

func NormalizeAndValidate(scope Scope, input EventInput) (Event, *RowError) {
	name := strings.TrimSpace(input.EventName)
	typ := strings.TrimSpace(input.EventType)
	if name == "" || typ == "" {
		return Event{}, rowInvalid("required", "eventName and eventType are required")
	}
	if input.App != "" || input.AppID != 0 || input.CommercialID != 0 {
		return Event{}, rowInvalid("forbidden_field", "app, appId and commercialId are not accepted")
	}
	if input.UID != 0 {
		return Event{}, rowInvalid("uid_mismatch", "subject is assigned from the identity capability")
	}
	if input.MerchantID != 0 && input.MerchantID != scope.MerchantID {
		return Event{}, rowInvalid("tenant_mismatch", "merchantId does not match resolved tenant")
	}
	if input.ShopID != 0 && input.ShopID != scope.ShopID {
		return Event{}, rowInvalid("tenant_mismatch", "shopId does not match resolved tenant")
	}
	fields := []struct {
		value string
		max   int
		name  string
	}{
		{strings.TrimSpace(input.EventID), 64, "eventId"},
		{name, 128, "eventName"},
		{typ, 32, "eventType"},
		{input.Page, 255, "page"},
		{input.Component, 128, "component"},
		{input.Action, 64, "action"},
		{input.BizType, 64, "bizType"},
		{input.BizID, 64, "bizId"},
		{input.SessionID, 96, "sessionId"},
		{input.AnonymousID, 96, "anonymousId"},
	}
	for _, field := range fields {
		if len(field.value) > field.max {
			return Event{}, rowInvalid("too_long", fmt.Sprintf("%s exceeds %d bytes", field.name, field.max))
		}
	}
	now := scope.Now
	if now.IsZero() {
		now = time.Now()
	}
	occurred := input.OccurredAtMs
	if occurred == 0 {
		occurred = input.ClientTs
	}
	if occurred == 0 {
		occurred = now.UnixMilli()
	}
	if occurred > now.Add(timeSkewFuture).UnixMilli() || occurred < now.Add(-timeSkewPast).UnixMilli() {
		return Event{}, rowInvalid("invalid_time", "event time must be within the last 7 days and no more than 10 minutes in the future")
	}
	live, err := marshalJSON(input.LiveContext)
	if err != nil {
		return Event{}, err
	}
	props, err := marshalJSON(input.Props)
	if err != nil {
		return Event{}, err
	}
	state, err := marshalJSON(input.State)
	if err != nil {
		return Event{}, err
	}
	extra, err := marshalJSON(input.Extra)
	if err != nil {
		return Event{}, err
	}
	if coreTrackEvents[name] {
		if err := validateCoreProps(name, input); err != nil {
			return Event{}, err
		}
	}
	schema := input.SchemaVersion
	if schema <= 0 {
		schema = 1
	}
	return Event{
		MerchantID: scope.MerchantID, ShopID: scope.ShopID, Surface: scope.Surface,
		EventID: strings.TrimSpace(input.EventID), EventType: typ, EventName: name,
		Page: strings.TrimSpace(input.Page), Component: strings.TrimSpace(input.Component),
		Action: strings.TrimSpace(input.Action), BizType: strings.TrimSpace(input.BizType),
		BizID: strings.TrimSpace(input.BizID), SessionID: strings.TrimSpace(input.SessionID),
		AnonymousID: strings.TrimSpace(input.AnonymousID), Subject: strings.TrimSpace(scope.Subject),
		ClientTs: occurred, OccurredAt: time.UnixMilli(occurred).UTC(), ReceivedAt: now.UTC(),
		SchemaVersion: schema, LiveContext: live, Props: props, State: state, Extra: extra,
		UserAgent: truncate(scope.UserAgent, 512), IP: truncate(scope.IP, 64), Referer: truncate(scope.Referer, 512),
		AdTouchID: scope.AdTouchID, ClickIDType: truncate(scope.ClickIDType, 32),
	}, nil
}

func validateCoreProps(name string, input EventInput) *RowError {
	if strings.HasPrefix(name, "session_") && strings.TrimSpace(input.SessionID) == "" {
		return rowInvalid("required", "sessionId is required for session lifecycle events")
	}
	if strings.HasPrefix(name, "page_") && name != "page_view" {
		if strings.TrimSpace(input.SessionID) == "" {
			return rowInvalid("required", "sessionId is required for page lifecycle events")
		}
		if v, ok := input.Props["page_seq"]; !ok || !validPositiveNumber(v) {
			return rowInvalid("invalid_field", "props.page_seq must be a positive integer")
		}
	}
	if name == "add_to_cart" {
		if v, ok := input.Props["sku_id"]; ok && !validNonNegativeNumber(v) {
			return rowInvalid("invalid_field", "props.sku_id must be a non-negative integer")
		}
	}
	if name == "payment_succeeded" || name == "payment_attempt" {
		if v, ok := input.Props["amount"]; ok && !validNonNegativeNumber(v) {
			return rowInvalid("invalid_field", "props.amount must be a non-negative integer")
		}
	}
	if v, ok := input.LiveContext["room_id"]; ok && !validNonNegativeNumber(v) {
		return rowInvalid("invalid_field", "liveContext.room_id must be a non-negative integer")
	}
	return nil
}

func marshalJSON(value map[string]any) (json.RawMessage, *RowError) {
	if value == nil {
		value = map[string]any{}
	}
	payload, err := json.Marshal(value)
	if err != nil || len(payload) > MaxJSONBytes {
		return nil, rowInvalid("json_too_large", "JSON field exceeds 16 KiB")
	}
	return payload, nil
}

func ValidSurface(surface string) bool {
	switch surface {
	case SurfaceShop, SurfaceMerch, SurfaceAdmin, SurfaceLive:
		return true
	}
	return false
}

func validPositiveNumber(v any) bool {
	if !validNonNegativeNumber(v) {
		return false
	}
	switch n := v.(type) {
	case float64:
		return n > 0
	case int:
		return n > 0
	case int64:
		return n > 0
	case json.Number:
		i, _ := n.Int64()
		return i > 0
	}
	return false
}

func validNonNegativeNumber(v any) bool {
	switch n := v.(type) {
	case float64:
		return n >= 0 && n == math.Trunc(n)
	case int:
		return n >= 0
	case int64:
		return n >= 0
	case json.Number:
		i, err := n.Int64()
		return err == nil && i >= 0
	default:
		return false
	}
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func rowInvalid(code, message string) *RowError {
	return &RowError{Code: strings.TrimSpace(code), Message: message}
}
