// Package audit exposes the append-only platform security and configuration log.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type Scope struct {
	Realm      string
	AppID      int64
	MerchantID int64
}

type Event struct {
	ID           string          `json:"id"`
	OccurredAt   time.Time       `json:"occurredAt"`
	ActorSubject string          `json:"actorSubject"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resourceType"`
	ResourceID   string          `json:"resourceId"`
	Result       string          `json:"result"`
	Details      json.RawMessage `json:"details"`
}

type Store struct{ db *sql.DB }

func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("audit database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) List(ctx context.Context, scope Scope, limit int) ([]Event, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT event_id,occurred_at,actor_subject,action,resource_type,resource_id,result,details
        FROM platform_audit_event WHERE realm=$1 AND app_id=$2 AND merchant_id=$3
        ORDER BY occurred_at DESC,event_id DESC LIMIT $4`, scope.Realm, scope.AppID, scope.MerchantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []Event
	for rows.Next() {
		var item Event
		if err := rows.Scan(&item.ID, &item.OccurredAt, &item.ActorSubject, &item.Action, &item.ResourceType, &item.ResourceID, &item.Result, &item.Details); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}
