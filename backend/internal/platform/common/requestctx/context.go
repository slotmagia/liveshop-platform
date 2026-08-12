// Package requestctx 保存已验证的 HTTP 身份与授权上下文。
package requestctx

import (
	"context"

	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
)

type identityKey struct{}
type authorizationKey struct{}

func With(ctx context.Context, identity accessidentity.Claims, authorization iam.Authorization) context.Context {
	ctx = context.WithValue(ctx, identityKey{}, identity)
	return context.WithValue(ctx, authorizationKey{}, authorization)
}

func Identity(ctx context.Context) accessidentity.Claims {
	value, _ := ctx.Value(identityKey{}).(accessidentity.Claims)
	return value
}

func Authorization(ctx context.Context) iam.Authorization {
	value, _ := ctx.Value(authorizationKey{}).(iam.Authorization)
	return value
}

func Tenant(ctx context.Context) iam.Tenant {
	identity := Identity(ctx)
	return iam.Tenant{AppID: identity.AppID, MerchantID: identity.MerchantID}
}
