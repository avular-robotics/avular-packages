package ports

import "avular-packages/internal/types"

type PolicyPort interface {
	ResolvePackagingMode(dep types.DependencyType, name string) (types.PackagingGroup, error)
}
