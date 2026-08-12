// Package identity 定义 Platform Identity 管理 HTTP 契约。
package identity

import (
	"github.com/gogf/gf/v2/frame/g"
	platformidentity "github.com/liveshop-platform/module-platform/internal/platform/registry/identity"
)

type AccountsReq struct {
	g.Meta `path:"/identity/accounts" method:"get" tags:"Platform-身份" summary:"读取后台账号"`
}
type AccountsRes []platformidentity.Account

type PutAccountReq struct {
	g.Meta          `path:"/identity/accounts/{realm}/{subject}" method:"put" tags:"Platform-身份" summary:"创建或更新后台账号"`
	Realm           string `json:"realm" in:"path"`
	Subject         string `json:"subject" in:"path"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Status          string `json:"status"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type PutAccountRes platformidentity.Account
