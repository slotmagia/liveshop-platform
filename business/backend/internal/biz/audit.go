package biz

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

// AuditRepository reads the append-only security and configuration log.
type AuditRepository interface {
	List(ctx context.Context, scope model.AuditScope, limit int) ([]model.AuditEvent, error)
}

type Audit struct{ repository AuditRepository }

func NewAudit(repository AuditRepository) *Audit { return &Audit{repository: repository} }

func (a *Audit) List(ctx context.Context, scope model.AuditScope, limit int) ([]model.AuditEvent, error) {
	return a.repository.List(ctx, scope, model.NormalizeAuditLimit(limit))
}
