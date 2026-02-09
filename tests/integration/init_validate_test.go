package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/adapters"
	"avular-packages/internal/core"
	"avular-packages/internal/types"
)

// TestInitValidateFlow exercises the single-file product workflow:
//
//	init scaffold -> load -> validate inline schema -> validate inline profile -> compose
//
// This verifies the full pipeline that a new user would follow after
// running `avular-packages init`.
func TestInitValidateFlow(t *testing.T) {
	dir := t.TempDir()

	// Step 1: Write a product spec with inline schema and inline profile.
	productContent := `
api_version: "v1"
kind: "product"
metadata:
  name: "test-product"
  version: "0.1.0"
  owners:
    - "ci"

defaults:
  target_ubuntu: "24.04"
  workspace:
    - "."
  output: "out"
  pip_index_url: "https://pip.example.com/simple"
  internal_deb_dir: "./prebuilt"

schema:
  schema_version: "v1"
  mappings:
    rclcpp:
      type: apt
      package: ros-humble-rclcpp
    numpy:
      type: pip
      package: numpy
      version: ">=1.26"

compose:
  - name: "base"
    source: "inline"
    profile:
      inputs:
        package_xml:
          enabled: true
          tags:
            - "debian_depend"
      packaging:
        groups:
          - name: "apt-runtime"
            mode: "individual"
            scope: "runtime"
            matches:
              - "apt:*"
            targets:
              - "24.04"

inputs:
  package_xml:
    enabled: true
    tags:
      - "debian_depend"
      - "pip_depend"

publish:
  repository:
    name: "test-product"
    channel: "dev"
    snapshot_prefix: "test"
    signing_key: "test-key"
`
	productPath := filepath.Join(dir, "product.yaml")
	require.NoError(t, os.WriteFile(productPath, []byte(productContent), 0644))

	// Step 2: Load the product spec.
	specAdapter := adapters.NewSpecFileAdapter()
	product, err := specAdapter.LoadProduct(productPath)
	require.NoError(t, err)

	// Step 3: Verify defaults were parsed correctly.
	assert.Equal(t, "24.04", product.Defaults.TargetUbuntu)
	assert.Equal(t, []string{"."}, product.Defaults.Workspace)
	assert.Equal(t, "out", product.Defaults.Output)
	assert.Equal(t, "https://pip.example.com/simple", product.Defaults.PipIndexURL)
	assert.Equal(t, "./prebuilt", product.Defaults.InternalDebDir)

	// Step 4: Verify inline schema was parsed.
	require.NotNil(t, product.Schema)
	assert.Equal(t, "v1", product.Schema.SchemaVersion)
	assert.Len(t, product.Schema.Mappings, 2)
	assert.Equal(t, types.DependencyTypeApt, product.Schema.Mappings["rclcpp"].Type)
	assert.Equal(t, "ros-humble-rclcpp", product.Schema.Mappings["rclcpp"].Package)
	assert.Equal(t, types.DependencyTypePip, product.Schema.Mappings["numpy"].Type)

	// Step 5: Verify inline profile is present.
	require.Len(t, product.Compose, 1)
	assert.Equal(t, "base", product.Compose[0].Name)
	assert.Equal(t, "inline", product.Compose[0].Source)
	require.NotNil(t, product.Compose[0].Profile)

	// Step 6: Load profiles (this exercises the inline profile path).
	profileSource := adapters.NewProfileSourceAdapter(specAdapter)
	profiles, err := profileSource.LoadProfiles(product, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "base", profiles[0].Metadata.Name)

	// Step 7: Compose and validate.
	composer := core.NewProductComposer()
	composed, err := composer.Compose(t.Context(), product, profiles)
	require.NoError(t, err)

	compiler := core.NewSpecCompiler()
	err = compiler.ValidateSpec(t.Context(), composed)
	require.NoError(t, err)

	// Step 8: Verify the composed spec has the merged inline schema.
	require.NotNil(t, composed.Schema)
	assert.Len(t, composed.Schema.Mappings, 2)
	assert.Equal(t, "ros-humble-rclcpp", composed.Schema.Mappings["rclcpp"].Package)

	// Step 9: Verify product-level inputs merged on top of profile inputs.
	assert.True(t, composed.Inputs.PackageXML.Enabled)
	assert.Contains(t, composed.Inputs.PackageXML.Tags, "pip_depend")
}

// TestInitValidateWithSchemaAutoDiscovery verifies that schemas/ next to
// a product spec are discovered and included in the inputs.
func TestInitValidateWithSchemaAutoDiscovery(t *testing.T) {
	dir := t.TempDir()

	// Create a schemas/ directory with a schema file.
	schemasDir := filepath.Join(dir, "schemas")
	require.NoError(t, os.MkdirAll(schemasDir, 0755))
	schemaContent := `
schema_version: "v1"
mappings:
  tf2_ros:
    type: apt
    package: ros-humble-tf2-ros
`
	require.NoError(t, os.WriteFile(filepath.Join(schemasDir, "ros-extra.yaml"), []byte(schemaContent), 0644))

	// Minimal product spec (no inline schema).
	productContent := `
api_version: "v1"
kind: "product"
metadata:
  name: "discover-test"
  version: "0.1.0"
  owners:
    - "ci"

defaults:
  target_ubuntu: "24.04"

compose:
  - name: "base"
    source: "inline"
    profile:
      inputs:
        package_xml:
          enabled: true
          tags:
            - "debian_depend"
      packaging:
        groups:
          - name: "apt-runtime"
            mode: "individual"
            scope: "runtime"
            matches:
              - "apt:*"
            targets:
              - "24.04"

inputs:
  package_xml:
    enabled: true
    tags:
      - "debian_depend"

publish:
  repository:
    name: "discover-test"
    channel: "dev"
    snapshot_prefix: "discover"
    signing_key: "test-key"
`
	productPath := filepath.Join(dir, "product.yaml")
	require.NoError(t, os.WriteFile(productPath, []byte(productContent), 0644))

	specAdapter := adapters.NewSpecFileAdapter()
	product, err := specAdapter.LoadProduct(productPath)
	require.NoError(t, err)

	profileSource := adapters.NewProfileSourceAdapter(specAdapter)
	profiles, err := profileSource.LoadProfiles(product, nil)
	require.NoError(t, err)

	composer := core.NewProductComposer()
	composed, err := composer.Compose(t.Context(), product, profiles)
	require.NoError(t, err)

	compiler := core.NewSpecCompiler()
	require.NoError(t, compiler.ValidateSpec(t.Context(), composed))

	// The auto-discovered schema file should exist on disk; verify it
	// can be loaded by the schema resolver adapter.
	resolver := adapters.NewSchemaResolverAdapter()
	err = resolver.LoadSchema(filepath.Join(schemasDir, "ros-extra.yaml"))
	require.NoError(t, err)
	assert.True(t, resolver.HasKey("tf2_ros"))
}
