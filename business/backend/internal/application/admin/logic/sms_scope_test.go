package logic

import (
	"context"
	"testing"

	"github.com/liveshop-platform/module-platform/internal/common/authctx"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
	"github.com/lvtuopen-ai/kernel-go/principal"
)

func TestSMSScopeFallsBackForInternalGrant(t *testing.T) {
	scope := smsScope(context.Background())
	if scope.Realm != "PLATFORM" || scope.Subject != "identity-compose" {
		t.Fatalf("scope=%+v", scope)
	}
}

func TestSMSScopeKeepsOperatorClaims(t *testing.T) {
	ctx := authctx.WithCapability(context.Background(), modulesession.Claims{
		Realm: principal.Realm("PLATFORM"), Subject: "operator-1", MerchantID: 0,
	})
	scope := smsScope(ctx)
	if scope.Realm != "PLATFORM" || scope.Subject != "operator-1" {
		t.Fatalf("scope=%+v", scope)
	}
}
