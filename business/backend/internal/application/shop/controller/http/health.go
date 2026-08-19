package http

import (
	"context"

	apihealth "github.com/liveshop-platform/module-platform/internal/application/shop/api/http/v1/health"
)

type HealthController struct{}

func NewHealth() *HealthController { return &HealthController{} }

func (c *HealthController) Get(_ context.Context, _ *apihealth.GetReq) (*apihealth.GetRes, error) {
	return &apihealth.GetRes{Status: "ok"}, nil
}
