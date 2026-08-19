package router

import (
	"github.com/gogf/gf/v2/net/ghttp"
	shophttp "github.com/liveshop-platform/module-platform/internal/application/shop/controller/http"
	"github.com/liveshop-platform/module-platform/internal/application/shop/service"
	"github.com/liveshop-platform/module-platform/internal/common/middleware"
	"github.com/liveshop-platform/module-platform/internal/common/web"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
	"github.com/lvtuopen-ai/kernel-go/principal"
)

const Prefix = "/shop/platform"

type Application interface {
	service.TrackEvents
}

type Deps struct {
	Application    Application
	ModuleSessions *modulesession.Verifier
}

func RegisterHTTP(root *ghttp.RouterGroup, deps Deps) {
	root.Group(Prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Bind(shophttp.NewHealth())
	})
	root.Group(Prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(middleware.PlatformModule(deps.ModuleSessions, principal.SurfaceShop))
		group.Middleware(middleware.RequirePermission("platform.track-event.write"))
		group.Bind(shophttp.NewTrackEventsWriter(deps.Application))
	})
}
