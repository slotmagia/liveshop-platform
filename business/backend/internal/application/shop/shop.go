package shop

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/application/shop/logic"
	"github.com/liveshop-platform/module-platform/internal/application/shop/router"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
)

type Config struct {
	Telemetry      *telemetry.UseCase
	ModuleSessions *modulesession.Verifier
}

type Surface struct{ deps router.Deps }

func New(config Config) Surface {
	return Surface{deps: router.Deps{
		Application:    logic.New(logic.Deps{Telemetry: config.Telemetry}),
		ModuleSessions: config.ModuleSessions,
	}}
}

func (s Surface) RegisterHTTP(root *ghttp.RouterGroup) { router.RegisterHTTP(root, s.deps) }
