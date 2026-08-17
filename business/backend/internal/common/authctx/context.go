// Package authctx carries a verified Identity-issued Module Capability.
// Only middleware writes it; application code only reads it.
package authctx

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
)

type capabilityKey struct{}

func WithCapability(ctx context.Context, capability modulesession.Claims) context.Context {
	return context.WithValue(ctx, capabilityKey{}, capability)
}

func Capability(ctx context.Context) modulesession.Claims {
	value, _ := ctx.Value(capabilityKey{}).(modulesession.Claims)
	return value
}

// RegistryActor is the verified operator recorded on module Registry changes.
func RegistryActor(ctx context.Context) model.RegistryAuditActor {
	capability := Capability(ctx)
	return model.RegistryAuditActor{
		Realm: capability.Realm.String(), MerchantID: capability.MerchantID, Subject: capability.Subject,
	}
}
