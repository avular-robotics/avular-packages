package ports

import "avular-packages/internal/types"

// SchemaResolverPort resolves abstract dependency keys (from standard ROS
// package.xml tags) into concrete typed dependencies using schema mapping
// files.
//
// The resolver supports layered overrides: each call to LoadSchema adds a
// new layer.  When multiple schemas define the same key, the last-loaded
// layer wins.  This enables workspace -> profile -> product precedence.
type SchemaResolverPort interface {
	// LoadSchema loads a schema.yaml file and merges its mappings into the
	// resolver's internal table.  Later loads override earlier ones per key.
	LoadSchema(path string) error

	// Resolve maps a single abstract key to a concrete typed Dependency.
	// Returns (dep, true, nil) on hit, (zero, false, nil) on miss, or
	// (zero, false, err) on failure.
	Resolve(key string) (types.Dependency, bool, error)

	// ResolveAll maps a batch of ROSTagDependency keys.  Unknown keys are
	// collected and returned as the second value so callers can decide
	// whether to treat them as errors, warnings, or fall through.
	ResolveAll(keys []types.ROSTagDependency) (resolved []types.Dependency, unknown []string, err error)

	// HasKey returns true if the key exists in any loaded schema layer.
	HasKey(key string) bool
}
