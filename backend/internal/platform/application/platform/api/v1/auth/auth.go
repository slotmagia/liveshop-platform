// Package auth 定义 Platform 认证 HTTP 契约。
package auth

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/identity"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
)

type LoginReq struct {
	g.Meta     `path:"/login" method:"post" tags:"Platform-认证" summary:"后台登录"`
	Realm      string `json:"realm"`
	AppID      int64  `json:"appId"`
	MerchantID int64  `json:"merchantId"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

type LoginRes identity.Result

type RefreshReq struct {
	g.Meta `path:"/refresh" method:"post" tags:"Platform-认证" summary:"轮换刷新令牌"`
}

type RefreshRes identity.Result

type LogoutReq struct {
	g.Meta `path:"/logout" method:"post" tags:"Platform-认证" summary:"退出并撤销刷新会话"`
}

type LogoutRes struct{}

func (*LogoutRes) NoContent() bool { return true }

type MeReq struct {
	g.Meta `path:"/me" method:"get" tags:"Platform-认证" summary:"读取当前身份和授权"`
}

type MeRes struct {
	Identity      accessidentity.Claims `json:"identity"`
	Authorization iam.Authorization     `json:"authorization"`
}
