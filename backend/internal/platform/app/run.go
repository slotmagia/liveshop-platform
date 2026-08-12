// Package app owns Platform dependency assembly and process lifecycle.
package app

import (
	"context"
	"errors"
	"time"

	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry"
	"github.com/lvtuopen-ai/kernel-go/logctx"
)

func Run(ctx context.Context) error {
	runtime, err := bootstrap(ctx)
	if err != nil {
		return err
	}
	defer platformregistry.Close()

	if err := runtime.httpServer.Start(); err != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = runtime.grpcServer.Stop(stopCtx)
		return err
	}
	grpcErrors := make(chan error, 1)
	go func() {
		grpcErrors <- runtime.grpcServer.Serve()
	}()
	logctx.FromContext(ctx).Info("platform registry listening", "address", runtime.cfg.Server.HTTP)
	logctx.FromContext(ctx).Info("platform gRPC registry listening", "address", runtime.grpcServer.Address())
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-grpcErrors:
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return errors.Join(serveErr, runtime.grpcServer.Stop(stopCtx), runtime.httpServer.Shutdown())
}
