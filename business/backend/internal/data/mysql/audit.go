package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

type AuditRepository struct{ db *sql.DB }

var _ biz.AuditRepository = (*AuditRepository)(nil)

func NewAuditRepository(db *sql.DB) *AuditRepository { return &AuditRepository{db: db} }

func (r *AuditRepository) Record(ctx context.Context, actor model.RegistryAuditActor, action, resourceType, resourceID string, details map[string]string) error {
	if !actor.Valid() {
		return errors.New("registry audit actor is required")
	}
	return transaction(ctx, r.db, 5*time.Second, func(ctx context.Context, tx *sql.Tx) error {
		return insertAuditEvent(ctx, tx, auditEntry{
			Realm:        actor.Realm,
			MerchantID:   actor.MerchantID,
			ActorSubject: actor.Subject,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Details:      details,
		})
	})
}

func (r *AuditRepository) List(ctx context.Context, scope model.AuditScope, limit int) ([]model.AuditEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_id,occurred_at,actor_subject,action,resource_type,resource_id,result,details
        FROM platform_audit_event WHERE realm=? AND merchant_id=?
        ORDER BY occurred_at DESC,event_id DESC LIMIT ?`, scope.Realm, scope.MerchantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []model.AuditEvent
	for rows.Next() {
		var item model.AuditEvent
		if err := rows.Scan(&item.ID, &item.OccurredAt, &item.ActorSubject, &item.Action, &item.ResourceType, &item.ResourceID, &item.Result, &item.Details); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}
