// Package provisioning assembles the provisioning surface, which serves machine
// callers holding a workload identity over both HTTP and gRPC.
package provisioning

import (
	"github.com/gogf/gf/v2/net/ghttp"
	platformv1 "github.com/liveshop-platform/contracts/gen/go/platform/v1"
	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning/logic"
	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning/router"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
	grpclib "google.golang.org/grpc"
)

type Config struct {
	Release   *biz.Release
	Workloads *workloadidentity.Verifier
}

type Surface struct{ deps router.Deps }

func New(config Config) Surface {
	return Surface{deps: router.Deps{
		Application: logic.New(config.Release),
		Workloads:   config.Workloads,
	}}
}

func (s Surface) RegisterHTTP(root *ghttp.RouterGroup) { router.RegisterHTTP(root, s.deps) }

func (s Surface) RegisterGRPC(registrar grpclib.ServiceRegistrar) {
	router.RegisterGRPC(registrar, s.deps.Application)
}

func (s Surface) GRPCServiceNames() []string {
	return []string{platformv1.PlatformRegistryService_ServiceDesc.ServiceName}
}

// GRPCMethodPermissions states the workload permission that authorizes each
// published method. The transport enforces it without knowing the contract.
func (s Surface) GRPCMethodPermissions() map[string]string {
	return map[string]string{
		platformv1.PlatformRegistryService_GetRouteSnapshot_FullMethodName:            "platform.registry.routes.read",
		platformv1.PlatformRegistryService_GetActiveCapabilitySnapshot_FullMethodName: "platform.registry.active-capabilities.read",
	}
}
