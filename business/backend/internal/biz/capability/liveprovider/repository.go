// Package liveprovider owns Platform live-provider use cases and repository ports.
package liveprovider

import (
	"context"

	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
)

// Repository performs each command, immutable version snapshot and audit write
// in one database transaction. Implementations must make CommandKey stable and
// reject reuse with a different request hash.
type Repository interface {
	List(context.Context, providermodel.Scope, providermodel.Filter) ([]providermodel.Provider, error)
	Upsert(context.Context, providermodel.Scope, providermodel.Upsert, string) (providermodel.Provider, error)
	Retire(context.Context, providermodel.Scope, providermodel.Retire, string) (providermodel.Provider, error)
}

type AssignmentRepository interface {
	GetAssignments(context.Context, int64) (providermodel.AssignmentSet, error)
	PutAssignments(context.Context, providermodel.PutAssignments) (providermodel.AssignmentSet, error)
}
