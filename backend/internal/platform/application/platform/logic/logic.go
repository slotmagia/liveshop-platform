// Package logic 实现 Platform HTTP 应用边界。
package logic

import (
	"errors"
	"net/http"

	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
)

type Logic struct{ deps platformregistry.Dependencies }

func New(deps platformregistry.Dependencies) *Logic { return &Logic{deps: deps} }

func unavailable(name string) error {
	return web.Error(http.StatusServiceUnavailable, errors.New(name+" service is unavailable"))
}

func iamError(err error) error {
	switch {
	case errors.Is(err, iam.ErrInvalid):
		return web.Error(http.StatusBadRequest, err)
	case errors.Is(err, iam.ErrNotFound):
		return web.Error(http.StatusNotFound, err)
	case errors.Is(err, iam.ErrConflict):
		return web.Error(http.StatusConflict, err)
	default:
		return web.Error(http.StatusServiceUnavailable, errors.New("IAM storage is unavailable"))
	}
}
