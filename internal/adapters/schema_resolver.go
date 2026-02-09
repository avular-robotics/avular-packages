package adapters

import (
	"os"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

// SchemaResolverAdapter implements SchemaResolverPort using layered
// schema.yaml files.  Each call to LoadSchema merges new mappings into
// the internal table; later loads override earlier ones per key.
type SchemaResolverAdapter struct {
	// merged holds the flattened mapping table after all layers.
	merged map[string]types.SchemaMapping

	// layers tracks load order for debugging / provenance.
	layers []string
}

// NewSchemaResolverAdapter returns an empty resolver ready for schema
// loading.
func NewSchemaResolverAdapter() *SchemaResolverAdapter {
	return &SchemaResolverAdapter{
		merged: make(map[string]types.SchemaMapping),
	}
}

// LoadSchema reads a schema.yaml file and merges its mappings.
// Keys in the new file override any existing entry (last-write wins).
func (a *SchemaResolverAdapter) LoadSchema(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("failed to read schema file: " + path).
			WithCause(err)
	}

	var schema types.SchemaFile
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("failed to parse schema file: " + path).
			WithCause(err)
	}

	if schema.SchemaVersion == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("schema file missing schema_version: " + path)
	}

	for key, mapping := range schema.Mappings {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}

		if mapping.Package == "" {
			return errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("schema key '" + normalizedKey + "' has empty package in " + path)
		}

		if mapping.Type != types.DependencyTypeApt && mapping.Type != types.DependencyTypePip {
			return errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("schema key '" + normalizedKey + "' has invalid type '" + string(mapping.Type) + "' in " + path)
		}

		if _, exists := a.merged[normalizedKey]; exists {
			log.Debug().
				Str("key", normalizedKey).
				Str("layer", path).
				Msg("schema key overridden by later layer")
		}

		a.merged[normalizedKey] = mapping
	}

	a.layers = append(a.layers, path)
	log.Debug().
		Str("path", path).
		Int("keys", len(schema.Mappings)).
		Int("total", len(a.merged)).
		Msg("schema layer loaded")

	return nil
}

// Resolve maps a single abstract key to a concrete Dependency.
func (a *SchemaResolverAdapter) Resolve(key string) (types.Dependency, bool, error) {
	normalizedKey := strings.TrimSpace(key)
	mapping, ok := a.merged[normalizedKey]
	if !ok {
		return types.Dependency{}, false, nil
	}

	dep := types.Dependency{
		Name: mapping.Package,
		Type: mapping.Type,
	}

	if mapping.Version != "" {
		constraint, err := parseSchemaVersion(mapping.Package, mapping.Version, mapping.Type)
		if err != nil {
			return types.Dependency{}, false, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("failed to parse version for schema key '" + normalizedKey + "'").
				WithCause(err)
		}
		dep.Constraints = constraint
	}

	return dep, true, nil
}

// ResolveAll maps a batch of ROS tag keys through the schema.
// Unknown keys (no schema entry) are returned separately.
func (a *SchemaResolverAdapter) ResolveAll(keys []types.ROSTagDependency) ([]types.Dependency, []string, error) {
	seen := make(map[string]struct{})
	var resolved []types.Dependency
	var unknown []string

	for _, tag := range keys {
		if _, dup := seen[tag.Key]; dup {
			continue
		}
		seen[tag.Key] = struct{}{}

		dep, ok, err := a.Resolve(tag.Key)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			unknown = append(unknown, tag.Key)
			continue
		}

		// Annotate constraint source with the schema provenance
		for i := range dep.Constraints {
			if dep.Constraints[i].Source == "" {
				dep.Constraints[i].Source = "schema:" + tag.Key
			}
		}

		resolved = append(resolved, dep)
	}

	return resolved, unknown, nil
}

// HasKey returns true if the key exists in any loaded schema layer.
func (a *SchemaResolverAdapter) HasKey(key string) bool {
	_, ok := a.merged[strings.TrimSpace(key)]
	return ok
}

// parseSchemaVersion turns a version constraint string into Constraint
// slices.  Supports comma-separated constraints like ">=1.0,<2.0".
func parseSchemaVersion(pkg string, version string, depType types.DependencyType) ([]types.Constraint, error) {
	parts := strings.Split(version, ",")
	var constraints []types.Constraint

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		// Detect operator prefix
		op, ver := splitConstraintOp(trimmed)

		name := pkg
		if depType == types.DependencyTypePip {
			name = normalizePipName(pkg)
		}

		constraints = append(constraints, types.Constraint{
			Name:    name,
			Op:      op,
			Version: ver,
			Source:  "schema",
		})
	}

	return constraints, nil
}

// splitConstraintOp parses ">=1.0" into (ConstraintOpGte, "1.0").
func splitConstraintOp(s string) (types.ConstraintOp, string) {
	for _, pair := range []struct {
		prefix string
		op     types.ConstraintOp
	}{
		{"~=", types.ConstraintOpCompat},
		{">=", types.ConstraintOpGte},
		{"<=", types.ConstraintOpLte},
		{"!=", types.ConstraintOpNe},
		{"==", types.ConstraintOpEq2},
		{">", types.ConstraintOpGt},
		{"<", types.ConstraintOpLt},
		{"=", types.ConstraintOpEq},
	} {
		if strings.HasPrefix(s, pair.prefix) {
			return pair.op, strings.TrimSpace(s[len(pair.prefix):])
		}
	}
	// No operator found; treat as exact version
	return types.ConstraintOpEq, strings.TrimSpace(s)
}

var _ ports.SchemaResolverPort = (*SchemaResolverAdapter)(nil)
