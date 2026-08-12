package main

import (
	"flag"
	"os"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/liveshop-platform/module-platform/internal/platform/app"
	"github.com/liveshop-platform/module-platform/pkg/gfinit"
	"github.com/lvtuopen-ai/kernel-go/lifecycle"
	"github.com/lvtuopen-ai/kernel-go/logctx"
)

func main() {
	configPath := flag.String("config", "./configs/platform.yaml", "path to YAML config")
	flag.Parse()
	initialized := gfinit.MustInit(gfinit.Options{Service: "platform", ConfigFile: *configPath})
	logLevel := g.Cfg().MustGet(initialized, "log.level").String()
	logFormat := g.Cfg().MustGet(initialized, "log.format").String()
	logctx.Configure(logctx.Options{Service: "platform", Level: logLevel, JSON: strings.EqualFold(logFormat, "json")})
	ctx, cancel := lifecycle.SignalContext(initialized)
	defer cancel()
	if err := app.Run(ctx); err != nil {
		logctx.FromContext(ctx).Error("platform stopped with an error", "error", err)
		os.Exit(1)
	}
}
