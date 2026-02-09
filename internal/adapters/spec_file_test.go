package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestLoadProductWithInlineFeatures(t *testing.T) {
	adapter := NewSpecFileAdapter()
	spec, err := adapter.LoadProduct("../../fixtures/single-file-product.yaml")
	require.NoError(t, err)

	// Basic product fields
	assert.Equal(t, types.SpecKindProduct, spec.Kind)
	assert.Equal(t, "single-file-demo", spec.Metadata.Name)

	// Defaults
	assert.Equal(t, "24.04", spec.Defaults.TargetUbuntu)
	assert.Equal(t, []string{"./src"}, spec.Defaults.Workspace)
	assert.Equal(t, "fixtures/repo-index.yaml", spec.Defaults.RepoIndex)
	assert.Equal(t, "out", spec.Defaults.Output)

	// Inline schema
	require.NotNil(t, spec.Schema)
	assert.Equal(t, "v1", spec.Schema.SchemaVersion)
	require.Len(t, spec.Schema.Mappings, 3)

	rclcpp := spec.Schema.Mappings["rclcpp"]
	assert.Equal(t, types.DependencyTypeApt, rclcpp.Type)
	assert.Equal(t, "ros-humble-rclcpp", rclcpp.Package)

	numpy := spec.Schema.Mappings["numpy"]
	assert.Equal(t, types.DependencyTypePip, numpy.Type)
	assert.Equal(t, "numpy", numpy.Package)
	assert.Equal(t, ">=1.26,<2.0", numpy.Version)

	// Inline profile via compose
	require.Len(t, spec.Compose, 1)
	compose := spec.Compose[0]
	assert.Equal(t, "default-profile", compose.Name)
	assert.Equal(t, "inline", compose.Source)
	require.NotNil(t, compose.Profile)
	assert.True(t, compose.Profile.Inputs.PackageXML.Enabled)
	assert.Equal(t, []string{"debian_depend", "pip_depend"}, compose.Profile.Inputs.PackageXML.Tags)
	require.Len(t, compose.Profile.Packaging.Groups, 2)
	assert.Equal(t, "apt-individual", compose.Profile.Packaging.Groups[0].Name)
	assert.Equal(t, "pip-meta", compose.Profile.Packaging.Groups[1].Name)
}

func TestLoadProductBackwardsCompatible(t *testing.T) {
	// Existing product files without new fields should still load fine
	adapter := NewSpecFileAdapter()
	spec, err := adapter.LoadProduct("../../fixtures/product-sample.yaml")
	require.NoError(t, err)

	assert.Equal(t, types.SpecKindProduct, spec.Kind)
	assert.Equal(t, "sample-product", spec.Metadata.Name)

	// New fields should be zero values
	assert.Empty(t, spec.Defaults.TargetUbuntu)
	assert.Nil(t, spec.Defaults.Workspace)
	assert.Empty(t, spec.Defaults.RepoIndex)
	assert.Nil(t, spec.Schema)

	// Compose should use file-based profiles
	require.Len(t, spec.Compose, 1)
	assert.Equal(t, "local", spec.Compose[0].Source)
	assert.Nil(t, spec.Compose[0].Profile)
}
