// Package registry is the private HTTP wire contract of the admin surface
// module management capability. It intentionally does not reuse the
// provisioning contract served to workload callers.
package registry

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/liveshop-platform/contracts/modulemanifest"
)

type ReleaseInfo struct {
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type ModuleInfo struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	ActiveVersion string        `json:"activeVersion,omitempty"`
	Releases      []ReleaseInfo `json:"releases"`
}

type CapabilityRelease struct {
	Version       string                                `json:"version"`
	Digest        string                                `json:"digest"`
	Active        bool                                  `json:"active"`
	Backend       modulemanifest.Backend                `json:"backend"`
	Permissions   []modulemanifest.PermissionDefinition `json:"permissions"`
	Contributions []modulemanifest.Contribution         `json:"contributions"`
}

type ModuleCapabilityCatalog struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	ActiveVersion string              `json:"activeVersion,omitempty"`
	Releases      []CapabilityRelease `json:"releases"`
}

type ModulesReq struct {
	g.Meta `path:"/modules" method:"get" tags:"Platform-模块管理" summary:"读取模块及活动版本"`
}
type ModulesRes []ModuleInfo

type CapabilitiesReq struct {
	g.Meta `path:"/capabilities" method:"get" tags:"Platform-模块管理" summary:"读取模块能力目录"`
}
type CapabilitiesRes struct {
	Revision uint64                    `json:"revision"`
	Items    []ModuleCapabilityCatalog `json:"items"`
}

type ActivateReq struct {
	g.Meta   `path:"/modules/{moduleId}/activate" method:"post" tags:"Platform-模块管理" summary:"激活模块版本"`
	ModuleID string `json:"moduleId" in:"path"`
	Version  string `json:"version"`
}
type ActivateRes struct{}

func (*ActivateRes) NoData() bool { return true }

type DeactivateReq struct {
	g.Meta   `path:"/modules/{moduleId}/deactivate" method:"delete" tags:"Platform-模块管理" summary:"停用模块"`
	ModuleID string `json:"moduleId" in:"path"`
}
type DeactivateRes struct{}

func (*DeactivateRes) NoContent() bool { return true }
