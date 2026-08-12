package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	apiauth "github.com/liveshop-platform/module-platform/internal/platform/application/platform/api/v1/auth"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/service"
	"github.com/liveshop-platform/module-platform/internal/platform/common/requestctx"
	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
)

const refreshCookieName = "liveshop_refresh"

type AuthController struct {
	service      service.Auth
	cookieSecure bool
}

func NewAuth(application service.Auth, cookieSecure bool) *AuthController {
	return &AuthController{service: application, cookieSecure: cookieSecure}
}

func (c *AuthController) Login(ctx context.Context, req *apiauth.LoginReq) (*apiauth.LoginRes, error) {
	result, err := c.service.Login(ctx, service.LoginInput{Realm: req.Realm, AppID: req.AppID, MerchantID: req.MerchantID, Username: req.Username, Password: req.Password})
	if err != nil {
		return nil, err
	}
	c.setRefreshCookie(ctx, result.RefreshToken, 7*24*time.Hour)
	response := apiauth.LoginRes(result)
	return &response, nil
}

func (c *AuthController) Refresh(ctx context.Context, _ *apiauth.RefreshReq) (*apiauth.RefreshRes, error) {
	request := ghttp.RequestFromCtx(ctx)
	cookie, err := request.Request.Cookie(refreshCookieName)
	if err != nil {
		return nil, web.Error(http.StatusUnauthorized, service.ErrInvalidRefresh)
	}
	result, err := c.service.Refresh(ctx, cookie.Value)
	if err != nil {
		c.clearRefreshCookie(ctx)
		return nil, err
	}
	c.setRefreshCookie(ctx, result.RefreshToken, 7*24*time.Hour)
	response := apiauth.RefreshRes(result)
	return &response, nil
}

func (c *AuthController) Logout(ctx context.Context, _ *apiauth.LogoutReq) (*apiauth.LogoutRes, error) {
	request := ghttp.RequestFromCtx(ctx)
	token := ""
	if cookie, err := request.Request.Cookie(refreshCookieName); err == nil {
		token = cookie.Value
	}
	if err := c.service.Logout(ctx, token); err != nil {
		return nil, err
	}
	c.clearRefreshCookie(ctx)
	return &apiauth.LogoutRes{}, nil
}

type MeController struct{}

func NewMe() *MeController { return &MeController{} }

func (c *MeController) Me(ctx context.Context, _ *apiauth.MeReq) (*apiauth.MeRes, error) {
	return &apiauth.MeRes{Identity: requestctx.Identity(ctx), Authorization: requestctx.Authorization(ctx)}, nil
}

func (c *AuthController) setRefreshCookie(ctx context.Context, value string, ttl time.Duration) {
	request := ghttp.RequestFromCtx(ctx)
	http.SetCookie(request.Response.BufferWriter, &http.Cookie{Name: refreshCookieName, Value: value, Path: "/auth", HttpOnly: true, Secure: c.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(ttl.Seconds())})
}

func (c *AuthController) clearRefreshCookie(ctx context.Context) {
	request := ghttp.RequestFromCtx(ctx)
	http.SetCookie(request.Response.BufferWriter, &http.Cookie{Name: refreshCookieName, Path: "/auth", HttpOnly: true, Secure: c.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}
