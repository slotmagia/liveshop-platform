// Package grpcserver is the Platform gRPC composition root.
package grpcserver

import (
	"context"
	"fmt"

	platformv1 "github.com/liveshop-platform/contracts/gen/go/platform/v1"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/controller"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/service"
	"github.com/liveshop-platform/module-platform/internal/platform/common/grpcauth"
	"github.com/lvtuopen-ai/kernel-go/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Config struct {
	Address   string
	TLS       TLSConfig
	Workloads []grpcauth.Workload
}

type Server struct {
	transport *grpcx.Server
	health    *health.Server
}

func New(config Config, application service.Registry) (*Server, error) {
	transportCredentials, err := serverCredentials(config.TLS)
	if err != nil {
		return nil, err
	}
	authorizer, err := grpcauth.New(config.Workloads, map[string]string{
		platformv1.PlatformRegistryService_GetRouteSnapshot_FullMethodName:     "platform.registry.routes.read",
		platformv1.PlatformRegistryService_GetCapabilityCatalog_FullMethodName: "platform.registry.capabilities.read",
	})
	if err != nil {
		return nil, err
	}
	transport, err := grpcx.NewServer(config.Address, grpcx.ServerOptions{
		TransportCredentials: transportCredentials,
		UnaryInterceptors: []grpc.UnaryServerInterceptor{
			authorizer.UnaryServerInterceptor(),
		},
		StreamInterceptors: []grpc.StreamServerInterceptor{
			authorizer.StreamServerInterceptor(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("platform: create gRPC server: %w", err)
	}
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus(platformv1.PlatformRegistryService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	grpc_health_v1.RegisterHealthServer(transport.Engine(), healthServer)
	controller.RegisterGRPCRegistry(transport.Engine(), application)
	return &Server{transport: transport, health: healthServer}, nil
}

func (s *Server) Address() string {
	if s == nil || s.transport == nil {
		return ""
	}
	return s.transport.Address()
}

func (s *Server) Serve() error {
	if s == nil || s.transport == nil || s.health == nil {
		return fmt.Errorf("platform: gRPC server is not initialized")
	}
	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	s.health.SetServingStatus(platformv1.PlatformRegistryService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)
	err := s.transport.Serve()
	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.health.SetServingStatus(platformv1.PlatformRegistryService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return err
}

func (s *Server) Stop(ctx context.Context) error {
	if s == nil || s.transport == nil {
		return nil
	}
	if s.health != nil {
		s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		s.health.SetServingStatus(platformv1.PlatformRegistryService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}
	return s.transport.Stop(ctx)
}
