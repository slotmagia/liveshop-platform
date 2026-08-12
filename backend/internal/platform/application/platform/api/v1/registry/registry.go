// Package registry 定义模块注册表 HTTP 契约。
package registry

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/liveshop-platform/contracts/modulemanifest"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry/module"
)

type RegisterReleaseReq struct {
	g.Meta `path:"/releases" method:"post" tags:"Platform-模块注册表" summary:"注册不可变模块发布"`
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

type CapabilitiesReq struct {
	g.Meta `path:"/capabilities" method:"get" tags:"Platform-模块注册表" summary:"读取模块能力目录"`
}
type CapabilitiesRes struct {
	Revision uint64                                     `json:"revision"`
	Items    []platformregistry.ModuleCapabilityCatalog `json:"items"`
}

type ModulesReq struct {
	g.Meta `path:"/modules" method:"get" tags:"Platform-模块管理" summary:"读取模块及活动版本"`
}
type ModulesRes []platformregistry.ModuleInfo

type AdminCapabilitiesReq struct {
	g.Meta `path:"/capabilities" method:"get" tags:"Platform-模块管理" summary:"读取模块能力目录"`
}

type AdminActivateReq struct {
	g.Meta   `path:"/modules/{moduleID}/activate" method:"post" tags:"Platform-模块管理" summary:"激活模块版本"`
	ModuleID string `json:"moduleID" in:"path"`
	Version  string `json:"version"`
}
type AdminActivateRes struct{}

func (*AdminActivateRes) NoData() bool { return true }

type AdminDeactivateReq struct {
	g.Meta   `path:"/modules/{moduleID}/activation" method:"delete" tags:"Platform-模块管理" summary:"停用模块"`
	ModuleID string `json:"moduleID" in:"path"`
}
type AdminDeactivateRes struct{}

func (*AdminDeactivateRes) NoContent() bool { return true }
