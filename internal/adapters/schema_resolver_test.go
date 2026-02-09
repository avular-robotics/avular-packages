package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestSchemaResolverLoadAndResolve(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.yaml")
	require.NoError(t, os.WriteFile(schemaPath, []byte(`
schema_version: "v1"
target: "ubuntu-22.04"
mappings:
  fmt:
    type: apt
    package: libfmt-dev
    version: ">=9.1.0"
  rclcpp:
    type: apt
    package: ros-humble-rclcpp
  numpy:
    type: pip
    package: numpy
    version: ">=1.26,<2.0"
`), 0644))

	resolver := NewSchemaResolverAdapter()
	require.NoError(t, resolver.LoadSchema(schemaPath))

	// Resolve known apt key
	dep, ok, err := resolver.Resolve("fmt")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "libfmt-dev", dep.Name)
	assert.Equal(t, types.DependencyTypeApt, dep.Type)
	require.Len(t, dep.Constraints, 1)
	assert.Equal(t, types.ConstraintOpGte, dep.Constraints[0].Op)
	assert.Equal(t, "9.1.0", dep.Constraints[0].Version)

	// Resolve known apt key without version
	dep, ok, err = resolver.Resolve("rclcpp")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "ros-humble-rclcpp", dep.Name)
	assert.Empty(t, dep.Constraints)

	// Resolve known pip key with compound constraint
	dep, ok, err = resolver.Resolve("numpy")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "numpy", dep.Name)
	assert.Equal(t, types.DependencyTypePip, dep.Type)
	require.Len(t, dep.Constraints, 2)
	assert.Equal(t, types.ConstraintOpGte, dep.Constraints[0].Op)
	assert.Equal(t, "1.26", dep.Constraints[0].Version)
	assert.Equal(t, types.ConstraintOpLt, dep.Constraints[1].Op)
	assert.Equal(t, "2.0", dep.Constraints[1].Version)

	// Resolve unknown key
	_, ok, err = resolver.Resolve("unknown_pkg")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSchemaResolverLayering(t *testing.T) {
	dir := t.TempDir()

	base := filepath.Join(dir, "base.yaml")
	require.NoError(t, os.WriteFile(base, []byte(`
schema_version: "v1"
mappings:
  numpy:
    type: pip
    package: numpy
    version: ">=1.26,<2.0"
  fmt:
    type: apt
    package: libfmt-dev
`), 0644))

	overlay := filepath.Join(dir, "overlay.yaml")
	require.NoError(t, os.WriteFile(overlay, []byte(`
schema_version: "v1"
mappings:
  numpy:
    type: pip
    package: numpy
    version: "==1.26.4"
  opencv:
    type: apt
    package: libopencv-dev
    version: ">=4.5"
`), 0644))

	resolver := NewSchemaResolverAdapter()
	require.NoError(t, resolver.LoadSchema(base))
	require.NoError(t, resolver.LoadSchema(overlay))

	// numpy overridden by overlay
	dep, ok, err := resolver.Resolve("numpy")
	require.NoError(t, err)
	assert.True(t, ok)
	require.Len(t, dep.Constraints, 1)
	assert.Equal(t, types.ConstraintOpEq2, dep.Constraints[0].Op)
	assert.Equal(t, "1.26.4", dep.Constraints[0].Version)

	// fmt still from base
	dep, ok, err = resolver.Resolve("fmt")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "libfmt-dev", dep.Name)

	// opencv added by overlay
	dep, ok, err = resolver.Resolve("opencv")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "libopencv-dev", dep.Name)
}

func TestSchemaResolverResolveAll(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.yaml")
	require.NoError(t, os.WriteFile(schemaPath, []byte(`
schema_version: "v1"
mappings:
  fmt:
    type: apt
    package: libfmt-dev
  rclcpp:
    type: apt
    package: ros-humble-rclcpp
`), 0644))

	resolver := NewSchemaResolverAdapter()
	require.NoError(t, resolver.LoadSchema(schemaPath))

	tags := []types.ROSTagDependency{
		{Key: "fmt", Scope: types.ROSDepScopeExec},
		{Key: "rclcpp", Scope: types.ROSDepScopeBuild},
		{Key: "unknown_lib", Scope: types.ROSDepScopeAll},
		{Key: "fmt", Scope: types.ROSDepScopeBuild}, // duplicate
	}

	resolved, unknown, err := resolver.ResolveAll(tags)
	require.NoError(t, err)
	assert.Len(t, resolved, 2) // fmt + rclcpp (deduplicated)
	assert.Equal(t, []string{"unknown_lib"}, unknown)
}

func TestSchemaResolverValidation(t *testing.T) {
	dir := t.TempDir()

	// Missing schema_version
	noVersion := filepath.Join(dir, "no_version.yaml")
	require.NoError(t, os.WriteFile(noVersion, []byte(`
mappings:
  fmt:
    type: apt
    package: libfmt-dev
`), 0644))

	resolver := NewSchemaResolverAdapter()
	err := resolver.LoadSchema(noVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema_version")

	// Invalid type
	badType := filepath.Join(dir, "bad_type.yaml")
	require.NoError(t, os.WriteFile(badType, []byte(`
schema_version: "v1"
mappings:
  fmt:
    type: rpm
    package: libfmt-dev
`), 0644))

	resolver2 := NewSchemaResolverAdapter()
	err = resolver2.LoadSchema(badType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")

	// Empty package
	emptyPkg := filepath.Join(dir, "empty_pkg.yaml")
	require.NoError(t, os.WriteFile(emptyPkg, []byte(`
schema_version: "v1"
mappings:
  fmt:
    type: apt
    package: ""
`), 0644))

	resolver3 := NewSchemaResolverAdapter()
	err = resolver3.LoadSchema(emptyPkg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty package")
}

func TestSchemaResolverHasKey(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.yaml")
	require.NoError(t, os.WriteFile(schemaPath, []byte(`
schema_version: "v1"
mappings:
  fmt:
    type: apt
    package: libfmt-dev
`), 0644))

	resolver := NewSchemaResolverAdapter()
	require.NoError(t, resolver.LoadSchema(schemaPath))

	assert.True(t, resolver.HasKey("fmt"))
	assert.False(t, resolver.HasKey("nonexistent"))
}
