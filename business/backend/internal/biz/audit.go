package biz

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

// AuditRepository reads the append-only security and configuration log.
type AuditRepository interface {
	List(ctx context.Context, scope model.AuditScope, limit int) ([]model.AuditEvent, error)
	Record(ctx context.Context, actor model.RegistryAuditActor, action, resourceType, resourceID string, details map[string]string) error
}

type Audit struct{ repository AuditRepository }

func NewAudit(repository AuditRepository) *Audit { return &Audit{repository: repository} }

func (a *Audit) List(ctx context.Context, scope model.AuditScope, limit int) ([]model.AuditEvent, error) {
	return a.repository.List(ctx, scope, model.NormalizeAuditLimit(limit))
}

// RecordRegistry writes a control-plane audit row after Registry has already
// accepted activate/deactivate. The two stores are not one transaction.
func (a *Audit) RecordRegistry(ctx context.Context, actor model.RegistryAuditActor, action, moduleID string, details map[string]string) error {
	if a == nil || a.repository == nil {
		return nil
	}
	return a.repository.Record(ctx, actor, action, "platform.module", moduleID, details)
}
