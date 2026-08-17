package app

import (
	"context"
	"fmt"

	"github.com/liveshop-platform/module-platform/internal/common/grpcauth"
	"github.com/liveshop-platform/module-platform/internal/common/grpcserver"
	"github.com/liveshop-platform/module-platform/internal/common/server"
	"github.com/liveshop-platform/module-platform/internal/config"
)

type instance struct {
	httpAddress string
	deps        Dependencies
	httpServer  *server.Server
	grpcServer  *grpcserver.Server
}

// bootstrap loads configuration, assembles dependencies and constructs both
// transports. Nothing listens until Run starts them.
func bootstrap(ctx context.Context) (*instance, error) {
	cfg, err := config.Load(ctx)
	if err != nil {
		return nil, err
	}
	deps, err := NewDependencies(cfg)
	if err != nil {
		return nil, fmt.Errorf("platform: assemble dependencies: %w", err)
	}
	applications := NewApplications(deps, cfg.InternalGrant.Token)

	httpServer := server.New(server.Config{
		AllowedOrigins: cfg.HTTP.AllowedOrigins,
		Ready:          deps.Ready,
	}, applications.HTTP()...)
	httpServer.SetAddr(cfg.Server.HTTP)

	grpcServer, err := grpcserver.New(grpcserver.Config{
		Address: cfg.Server.GRPC,
		TLS: grpcserver.TLSConfig{
			CertificateFile: cfg.GRPC.TLS.CertificateFile,
			PrivateKeyFile:  cfg.GRPC.TLS.PrivateKeyFile,
			ClientCAFile:    cfg.GRPC.TLS.ClientCAFile,
		},
		Workloads: []grpcauth.Workload{
			workload(cfg.WorkloadIdentity.GRPC.Gateway),
			workload(cfg.WorkloadIdentity.GRPC.Identity),
		},
	}, applications.Provisioning)
	if err != nil {
		_ = deps.Close()
		return nil, err
	}
	return &instance{httpAddress: cfg.Server.HTTP, deps: deps, httpServer: httpServer, grpcServer: grpcServer}, nil
}

func workload(cfg config.MTLSWorkload) grpcauth.Workload {
	return grpcauth.Workload{SPIFFEID: cfg.SPIFFEID, Subject: cfg.Subject, Permissions: cfg.Permissions}
}
