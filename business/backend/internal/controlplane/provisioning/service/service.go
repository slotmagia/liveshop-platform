// Package service declares the provisioning surface application boundary.
// Both the HTTP and the gRPC transport of this surface depend on it.
package service

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning/appmodel"
)

type Provisioning interface {
	RegisterRelease(ctx context.Context, document []byte) (appmodel.RegisteredRelease, error)
	Activate(ctx context.Context, activation appmodel.Activation) error
	Routes(ctx context.Context) (appmodel.RouteSnapshot, error)
	ActiveCapabilities(ctx context.Context) (appmodel.ActiveCapabilitySnapshot, error)
}
