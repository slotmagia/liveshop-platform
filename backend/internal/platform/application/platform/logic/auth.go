package logic

import (
	"context"
	"errors"
	"net/http"

	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/identity"
)

func (l *Logic) Login(ctx context.Context, input identity.LoginInput) (identity.Result, error) {
	if l.deps.Identity == nil {
		return identity.Result{}, unavailable("identity")
	}
	result, err := l.deps.Identity.Login(ctx, input)
	if err == nil {
		return result, nil
	}
	status := http.StatusUnauthorized
	if errors.Is(err, identity.ErrLocked) {
		status = http.StatusTooManyRequests
	} else if !errors.Is(err, identity.ErrInvalidCredentials) && !errors.Is(err, identity.ErrDisabled) {
		status = http.StatusServiceUnavailable
	}
	return identity.Result{}, web.Error(status, err)
}

func (l *Logic) Refresh(ctx context.Context, token string) (identity.Result, error) {
	if l.deps.Identity == nil {
		return identity.Result{}, unavailable("identity")
	}
	result, err := l.deps.Identity.Refresh(ctx, token)
	if err != nil {
		return identity.Result{}, web.Error(http.StatusUnauthorized, err)
	}
	return result, nil
}

func (l *Logic) Logout(ctx context.Context, token string) error {
	if l.deps.Identity == nil || token == "" {
		return nil
	}
	if err := l.deps.Identity.Logout(ctx, token); err != nil {
		return web.Error(http.StatusServiceUnavailable, err)
	}
	return nil
}
