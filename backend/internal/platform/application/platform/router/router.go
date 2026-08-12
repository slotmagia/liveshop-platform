// Package router 注册 Platform 的 auth、internal、runtime 和 admin HTTP 边界。
package router

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/controller"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/service"
	commonmw "github.com/liveshop-platform/module-platform/internal/platform/common/middleware"
	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
)

type Deps struct {
	Application    service.Application
	IAM            *iam.Store
	Workloads      *workloadidentity.Verifier
	Identities     *accessidentity.Verifier
	ModuleSessions *modulesession.Verifier
	CookieSecure   bool
}

func Register(root *ghttp.RouterGroup, deps Deps) {
	registerAuth(root, deps)
	registerInternal(root, deps)
	registerRuntime(root, deps)
	registerAdmin(root, deps)
}

func registerAuth(root *ghttp.RouterGroup, deps Deps) {
	root.Group("/auth", func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Bind(controller.NewAuth(deps.Application, deps.CookieSecure))
	})
	root.Group("/auth", func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(commonmw.UserIdentity(deps.Identities, deps.IAM))
		group.Bind(controller.NewMe())
	})
}

func registerInternal(root *ghttp.RouterGroup, deps Deps) {
	const prefix = "/internal/v1/module-registry"
	root.Group(prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(commonmw.Workload(deps.Workloads, "registry.release.write"))
		group.Bind(controller.NewRelease(deps.Application))
	})
	root.Group(prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(commonmw.Workload(deps.Workloads, "registry.activation.write"))
		group.Bind(controller.NewActivation(deps.Application))
	})
	root.Group(prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(commonmw.Workload(deps.Workloads, "registry.routes.read"))
		group.Bind(controller.NewRoutes(deps.Application))
	})
	root.Group(prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(commonmw.Workload(deps.Workloads, "registry.capabilities.read"))
		group.Bind(controller.NewCapabilities(deps.Application))
	})
}

func registerRuntime(root *ghttp.RouterGroup, deps Deps) {
	root.Group("/runtime/v1", func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(commonmw.UserIdentity(deps.Identities, deps.IAM))
		group.Bind(controller.NewRuntime(deps.Application))
	})
	root.Group("/runtime/v1", func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(commonmw.UserIdentity(deps.Identities, deps.IAM))
		group.Middleware(commonmw.RequirePlatformCapabilityReader)
		group.Bind(controller.NewCatalog(deps.Application))
	})
}

func registerAdmin(root *ghttp.RouterGroup, deps Deps) {
	registerAdminCapability(root, deps, "/admin/platform/registry", "platform.registry.manage", controller.NewAdminRegistry(deps.Application))
	registerAdminCapability(root, deps, "/admin/platform", "platform.settings.read", controller.NewSettingsReader(deps.Application))
	registerAdminCapability(root, deps, "/admin/platform", "platform.settings.write", controller.NewSettingsWriter(deps.Application))
	registerAdminCapability(root, deps, "/admin/platform", "platform.audit.read", controller.NewAudit(deps.Application))
	registerAdminCapability(root, deps, "/admin/platform", "platform.identity.manage", controller.NewIdentity(deps.Application))
	registerAdminCapability(root, deps, "/admin/platform", "platform.iam.manage", controller.NewIAM(deps.Application))
}

func registerAdminCapability(root *ghttp.RouterGroup, deps Deps, prefix, permission string, target any) {
	root.Group(prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(commonmw.PlatformModule(deps.ModuleSessions))
		group.Middleware(commonmw.RequirePermission(permission))
		group.Bind(target)
	})
}
