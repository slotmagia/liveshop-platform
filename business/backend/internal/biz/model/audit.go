package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

func NewEventID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(value)
}

const (
	AuditDefaultLimit = 100
	AuditMaxLimit     = 200
)

type AuditScope struct {
	Realm      string
	MerchantID int64
}

type AuditEvent struct {
	ID           string
	OccurredAt   time.Time
	ActorSubject string
	Action       string
	ResourceType string
	ResourceID   string
	Result       string
	Details      json.RawMessage
}

func NormalizeAuditLimit(limit int) int {
	if limit <= 0 || limit > AuditMaxLimit {
		return AuditDefaultLimit
	}
	return limit
}
