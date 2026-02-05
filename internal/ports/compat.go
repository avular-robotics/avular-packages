package ports

import "avular-packages/internal/types"

type CompatibilityPort interface {
	WriteGetDependencies(resolved []types.ResolvedDependency) error
	WriteRosdepMapping(resolved []types.ResolvedDependency) error
}
