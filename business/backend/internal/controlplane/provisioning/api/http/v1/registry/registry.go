// Package registry is the private HTTP wire contract of the provisioning
// surface, which serves machine callers holding a workload identity.
package registry

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/liveshop-platform/contracts/modulemanifest"
)

// RegisterReleaseReq carries no named field: its request body is the complete
// ModuleRelease manifest document, bounded by the server body size limit.
type RegisterReleaseReq struct {
	g.Meta `path:"/releases" method:"post" mime:"application/json" tags:"Platform-模块注册表" summary:"注册不可变模块发布" description:"请求体为完整的 ModuleRelease manifest JSON 文档"`
}

type RegisterReleaseRes struct {
	Digest string `json:"digest"`
}

type ActivateReq struct {
	g.Meta   `path:"/activate" method:"post" tags:"Platform-模块注册表" summary:"激活模块版本"`
	ModuleID string `json:"moduleId"`
	Version  string `json:"version"`
}

type ActivateRes struct{}

func (*ActivateRes) NoData() bool { return true }

type RoutesReq struct {
	g.Meta `path:"/routes" method:"get" tags:"Platform-模块注册表" summary:"读取活动路由快照"`
}

type RoutesRes struct {
	Revision uint64                       `json:"revision"`
	Routes   []modulemanifest.ActiveRoute `json:"routes"`
}
