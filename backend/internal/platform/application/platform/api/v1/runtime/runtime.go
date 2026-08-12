// Package runtime 定义 Host 运行时 HTTP 契约。
package runtime

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry/module"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
)

type ContributionsReq struct {
	g.Meta  `path:"/contributions" method:"get" tags:"Platform-运行时" summary:"按 Surface 读取可见贡献"`
	Surface string `json:"surface" in:"query"`
}
type ContributionsRes struct {
	Revision uint64                               `json:"revision"`
	Items    []modulemanifest.RuntimeContribution `json:"items"`
}

type ModuleSessionReq struct {
	g.Meta         `path:"/module-sessions" method:"post" tags:"Platform-运行时" summary:"签发贡献绑定的模块会话"`
	ModuleID       string `json:"moduleId"`
	ModuleVersion  string `json:"moduleVersion"`
	ContributionID string `json:"contributionId"`
	Surface        string `json:"surface"`
}
type ModuleSessionRes struct {
	Token                 string                    `json:"token"`
	ExpiresIn             int64                     `json:"expiresIn"`
	AuthorizationRevision uint64                    `json:"authorizationRevision"`
	Permissions           []string                  `json:"permissions"`
	DataScopes            []modulesession.DataScope `json:"dataScopes"`
	Tenant                Tenant                    `json:"tenant"`
}
type Tenant struct {
	AppID      int64 `json:"appId"`
	MerchantID int64 `json:"merchantId"`
}

type MyAuthorizationReq struct {
	g.Meta `path:"/iam/me" method:"get" tags:"Platform-运行时" summary:"读取当前有效授权"`
}
type MyAuthorizationRes iam.Authorization

type ModuleCatalogReq struct {
	g.Meta `path:"/module-catalog" method:"get" tags:"Platform-运行时" summary:"读取管理端模块能力目录"`
}
type ModuleCatalogRes struct {
	Revision uint64                                     `json:"revision"`
	Items    []platformregistry.ModuleCapabilityCatalog `json:"items"`
}
