package http

import (
	"context"

	apiaudit "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/audit"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type AuditController struct{ service service.Audit }

func NewAudit(application service.Audit) *AuditController {
	return &AuditController{service: application}
}

func (c *AuditController) List(ctx context.Context, req *apiaudit.ListReq) (*apiaudit.ListRes, error) {
	items, err := c.service.ListAudit(ctx, req.Limit)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apiaudit.ListRes, 0, len(items))
	for _, item := range items {
		response = append(response, apiaudit.Event{
			ID:           item.ID,
			OccurredAt:   item.OccurredAt,
			ActorSubject: item.ActorSubject,
			Action:       item.Action,
			ResourceType: item.ResourceType,
			ResourceID:   item.ResourceID,
			Result:       item.Result,
			Details:      item.Details,
		})
	}
	return &response, nil
}
