package notification

import (
	"github.com/gogf/gf/v2/net/ghttp"
	platformv1 "github.com/liveshop-platform/contracts/gen/go/platform/v1"
	biznotify "github.com/liveshop-platform/module-platform/internal/biz/capability/notification"
	"github.com/liveshop-platform/module-platform/internal/controlplane/notification/logic"
	"github.com/liveshop-platform/module-platform/internal/controlplane/notification/router"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
	grpclib "google.golang.org/grpc"
)

type Config struct {
	Notification *biznotify.UseCase
	Workloads    *workloadidentity.Verifier
}

type Surface struct{ deps router.Deps }

func New(config Config) Surface {
	return Surface{deps: router.Deps{
		Application: logic.New(config.Notification),
		Workloads:   config.Workloads,
	}}
}

func (s Surface) RegisterHTTP(root *ghttp.RouterGroup) { router.RegisterHTTP(root, s.deps) }

func (s Surface) RegisterGRPC(registrar grpclib.ServiceRegistrar) {
	router.RegisterGRPC(registrar, s.deps.Application)
}

func (s Surface) GRPCServiceNames() []string {
	return []string{platformv1.PlatformNotificationService_ServiceDesc.ServiceName}
}

func (s Surface) GRPCMethodPermissions() map[string]string {
	return map[string]string{
		platformv1.PlatformNotificationService_Dispatch_FullMethodName:    "platform.notify-event.dispatch",
		platformv1.PlatformNotificationService_GetDelivery_FullMethodName: "platform.notify-event.dispatch",
	}
}
