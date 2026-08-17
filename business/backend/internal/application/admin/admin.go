// Package admin assembles the admin surface, which serves the Platform
// contribution running inside the Admin Host under an Identity Module Capability.
package admin

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/application/admin/logic"
	"github.com/liveshop-platform/module-platform/internal/application/admin/router"
	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/email"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/sms"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/storage"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
)

// Config lists exactly the use cases the admin surface may reach.
type Config struct {
	Release        *biz.Release
	Settings       *biz.Settings
	Audit          *biz.Audit
	LiveProvider   *liveprovider.UseCase
	SMS            *sms.UseCase
	Email          *email.UseCase
	Storage        *storage.UseCase
	ModuleSessions *modulesession.Verifier
}

type Surface struct{ deps router.Deps }

func New(config Config) Surface {
	return Surface{deps: router.Deps{
		Application: logic.New(logic.Deps{
			Release:      config.Release,
			Settings:     config.Settings,
			Audit:        config.Audit,
			LiveProvider: config.LiveProvider,
			SMS:          config.SMS,
			Email:        config.Email,
			Storage:      config.Storage,
		}),
		ModuleSessions: config.ModuleSessions,
	}}
}

func (s Surface) RegisterHTTP(root *ghttp.RouterGroup) { router.RegisterHTTP(root, s.deps) }

func (s Surface) Application() router.Application { return s.deps.Application }
