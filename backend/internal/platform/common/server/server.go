// Package server is the sole HTTP composition root for Platform surfaces.
package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	platformrouter "github.com/liveshop-platform/module-platform/internal/platform/application/platform/router"
	platformservice "github.com/liveshop-platform/module-platform/internal/platform/application/platform/service"
	commonmw "github.com/liveshop-platform/module-platform/internal/platform/common/middleware"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry"
)

var serverSequence atomic.Uint64

type Server struct {
	engine *ghttp.Server
}

type Config struct {
	CookieSecure   bool
	AllowedOrigins []string
}

func New(deps platformregistry.Dependencies, application platformservice.Application, config Config) *Server {
	origins := make(map[string]struct{}, len(config.AllowedOrigins))
	for _, origin := range config.AllowedOrigins {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins[origin] = struct{}{}
		}
	}

	engine := ghttp.GetServer(fmt.Sprintf("platform-http-%d", serverSequence.Add(1)))
	engine.SetAddr(":0")
	engine.SetDumpRouterMap(false)
	engine.SetAccessLogEnabled(false)
	engine.SetReadTimeout(15 * time.Second)
	engine.SetWriteTimeout(15 * time.Second)
	engine.SetIdleTimeout(60 * time.Second)
	engine.SetMaxHeaderBytes(1 << 20)
	engine.SetClientMaxBodySize(2 << 20)
	engine.SetGraceful(true)
	engine.SetGracefulShutdownTimeout(10)
	engine.Group("/", func(root *ghttp.RouterGroup) {
		root.Middleware(commonmw.RequestMetadata)
		root.Middleware(commonmw.CORS(origins))
		root.Middleware(commonmw.ValidateJSON)
		writeLive := func(request *ghttp.Request) {
			request.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
			request.Response.WriteJson(map[string]string{"status": "ok"})
		}
		writeReady := func(request *ghttp.Request) {
			if deps.Ready != nil {
				ctx, cancel := context.WithTimeout(request.GetCtx(), 2*time.Second)
				defer cancel()
				if err := deps.Ready(ctx); err != nil {
					request.Response.WriteStatus(http.StatusServiceUnavailable)
					request.Response.WriteJson(map[string]string{"status": "not-ready"})
					return
				}
			}
			writeLive(request)
		}
		root.GET("/livez", writeLive)
		root.GET("/readyz", writeReady)
		root.GET("/health", writeReady)
		platformrouter.Register(root, platformrouter.Deps{
			Application:    application,
			IAM:            deps.IAM,
			Workloads:      deps.Workloads,
			Identities:     deps.Identities,
			ModuleSessions: deps.ModuleVerifier,
			CookieSecure:   config.CookieSecure,
		})
	})
	return &Server{engine: engine}
}

func (s *Server) SetAddr(addr string) {
	s.engine.SetAddr(addr)
}

func (s *Server) Handler() http.Handler {
	return s.engine
}

func (s *Server) Start() error {
	return s.engine.Start()
}

func (s *Server) Shutdown() error {
	return s.engine.Shutdown()
}
