package i18n

import (
	"github.com/gogf/gf/v2/net/ghttp"
	bizloc "github.com/liveshop-platform/module-platform/internal/biz/capability/localization"
	"github.com/liveshop-platform/module-platform/internal/controlplane/i18n/router"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
)

type Config struct {
	Localization *bizloc.UseCase
	Workloads    *workloadidentity.Verifier
	Grant        string
}

type Surface struct{ deps router.Deps }

func New(config Config) Surface {
	return Surface{deps: router.Deps{UseCase: config.Localization, Workloads: config.Workloads, Grant: config.Grant}}
}

func (s Surface) RegisterHTTP(root *ghttp.RouterGroup) { router.RegisterHTTP(root, s.deps) }
