package storage

import (
	"context"

	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
)

// Repository performs each storage command, immutable version snapshot and
// audit write in one database transaction. CommandKey reuse with a different
// hash is a conflict.
type Repository interface {
	ListChannels(context.Context, storagemodel.Scope, storagemodel.ChannelFilter) ([]storagemodel.Channel, error)
	UpsertChannel(context.Context, storagemodel.Scope, storagemodel.UpsertChannel, string) (storagemodel.Channel, error)
	SetEnabled(context.Context, storagemodel.Scope, storagemodel.SetEnabled, string) (storagemodel.Channel, error)
	SetDefault(context.Context, storagemodel.Scope, storagemodel.SetDefault, string) (storagemodel.Channel, error)
	Retire(context.Context, storagemodel.Scope, storagemodel.Retire, string) (storagemodel.Channel, error)
	LoadSecrets(context.Context, storagemodel.Scope, string) (storagemodel.ChannelSecrets, error)
}

// Sender writes one probe object through a concrete driver and can read back
// local-disk objects. Cloud objects are verified by their public URL.
type Sender interface {
	Put(context.Context, storagemodel.Driver, map[string]string, string, []byte) (url string, err error)
	GetLocal(string) (storagemodel.Object, error)
}
