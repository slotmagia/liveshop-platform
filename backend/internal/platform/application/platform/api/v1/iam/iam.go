// Package iam 定义 Platform IAM 管理 HTTP 契约。
package iam

import (
	"github.com/gogf/gf/v2/frame/g"
	platformiam "github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
)

type PermissionsReq struct {
	g.Meta `path:"/iam/permissions" method:"get" tags:"Platform-IAM" summary:"读取权限目录"`
}
type PermissionsRes []platformiam.Permission

type DepartmentsReq struct {
	g.Meta `path:"/iam/departments" method:"get" tags:"Platform-IAM" summary:"读取部门"`
}
type DepartmentsRes []platformiam.Department

type PutDepartmentReq struct {
	g.Meta          `path:"/iam/departments/{departmentID}" method:"put" tags:"Platform-IAM" summary:"创建或更新部门"`
	DepartmentID    int64  `json:"departmentID" in:"path"`
	ExpectedVersion int64  `json:"expectedVersion"`
	ParentID        *int64 `json:"parentId"`
	Name            string `json:"name"`
	Status          string `json:"status"`
}
type PutDepartmentRes platformiam.Department

type RolesReq struct {
	g.Meta `path:"/iam/roles" method:"get" tags:"Platform-IAM" summary:"读取角色"`
}
type RolesRes []platformiam.Role

type PutRoleReq struct {
	g.Meta          `path:"/iam/roles/{roleID}" method:"put" tags:"Platform-IAM" summary:"创建或更新角色"`
	RoleID          int64  `json:"roleID" in:"path"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	SuperAdmin      bool   `json:"superAdmin"`
}
type PutRoleRes platformiam.Role

type PutRolePolicyReq struct {
	g.Meta          `path:"/iam/roles/{roleID}/policy" method:"put" tags:"Platform-IAM" summary:"替换角色权限和数据范围"`
	RoleID          int64                   `json:"roleID" in:"path"`
	ExpectedVersion int64                   `json:"expectedVersion"`
	Permissions     []string                `json:"permissions"`
	Scopes          []platformiam.ScopeRule `json:"scopes"`
}
type PutRolePolicyRes platformiam.Role

type PutSubjectAssignmentReq struct {
	g.Meta      `path:"/iam/subjects/{subject}/assignment" method:"put" tags:"Platform-IAM" summary:"替换用户角色和部门分配"`
	Subject     string                             `json:"subject" in:"path"`
	RoleIDs     []int64                            `json:"roleIds"`
	Departments []platformiam.DepartmentMembership `json:"departments"`
}
type PutSubjectAssignmentRes struct{}

func (*PutSubjectAssignmentRes) NoData() bool { return true }

type SubjectAccessReq struct {
	g.Meta  `path:"/iam/subjects/{subject}/access" method:"get" tags:"Platform-IAM" summary:"读取用户有效授权"`
	Subject string `json:"subject" in:"path"`
}
type SubjectAccessRes platformiam.Authorization
