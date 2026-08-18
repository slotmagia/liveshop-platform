package router

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/common/middleware"
	"github.com/liveshop-platform/module-platform/internal/common/web"
	notifygrpc "github.com/liveshop-platform/module-platform/internal/controlplane/notification/controller/grpc"
	notifyhttp "github.com/liveshop-platform/module-platform/internal/controlplane/notification/controller/http"
	"github.com/liveshop-platform/module-platform/internal/controlplane/notification/service"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
	grpclib "google.golang.org/grpc"
)

const Prefix = "/internal/platform/notify-events"

type Deps struct {
	Application service.Notification
	Workloads   *workloadidentity.Verifier
}

func RegisterHTTP(root *ghttp.RouterGroup, deps Deps) {
	root.Group(Prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(middleware.Workload(deps.Workloads, "platform.notify-event.dispatch"))
		group.Bind(notifyhttp.New(deps.Application))
	})
}

func RegisterGRPC(server grpclib.ServiceRegistrar, application service.Notification) {
	notifygrpc.Register(server, application)
}
