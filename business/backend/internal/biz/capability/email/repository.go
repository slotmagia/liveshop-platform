package email

import (
	"context"

	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
)

// Repository performs each email command, immutable version snapshot and audit
// write in one database transaction. CommandKey reuse with a different hash is
// a conflict.
type Repository interface {
	GetConfig(context.Context, emailmodel.Scope) (emailmodel.Config, error)
	UpsertConfig(context.Context, emailmodel.Scope, emailmodel.UpsertConfig, string) (emailmodel.Config, error)
	SetEnabled(context.Context, emailmodel.Scope, emailmodel.SetEnabled, string) (emailmodel.Config, error)
	LoadSecrets(context.Context, emailmodel.Scope) (emailmodel.ConfigSecrets, error)
}

// Sender delivers one test message through a concrete driver.
type Sender interface {
	Send(context.Context, emailmodel.Driver, map[string]string, emailmodel.TestSend) (string, error)
}
