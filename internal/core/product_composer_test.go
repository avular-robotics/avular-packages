package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

// ---------------------------------------------------------------------------
// ProductComposer.Compose
// ---------------------------------------------------------------------------

func TestComposerBasicProductOnly(t *testing.T) {
	composer := NewProductComposer()
	product := types.Spec{
		APIVersion: "v1",
		Kind:       types.SpecKindProduct,
		Metadata:   types.Metadata{Name: "test-product", Version: "1.0.0"},
		Inputs: types.Inputs{
			Manual: types.ManualInputs{Apt: []string{"libfoo=1.0.0"}},
		},
	}

	result, err := composer.Compose(context.Background(), product, nil)
	require.NoError(t, err)
	assert.Equal(t, types.SpecKindProduct, result.Kind)
	assert.Equal(t, "test-product", result.Metadata.Name)
	assert.Equal(t, []string{"libfoo=1.0.0"}, result.Inputs.Manual.Apt)
}

func TestComposerRejectsNonProductSpec(t *testing.T) {
	composer := NewProductComposer()
	profile := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "base"},
	}

	_, err := composer.Compose(context.Background(), profile, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compose requires product spec")
}

func TestComposerRejectsNonProfileInProfiles(t *testing.T) {
	composer := NewProductComposer()
	product := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "prod"},
	}
	notAProfile := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "also-product"},
	}

	_, err := composer.Compose(context.Background(), product, []types.Spec{notAProfile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid profile spec kind")
}

func TestComposerMergesProfileInputsIntoProduct(t *testing.T) {
	composer := NewProductComposer()
	profile := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "base"},
		Inputs: types.Inputs{
			Manual:     types.ManualInputs{Apt: []string{"libbase"}, Python: []string{"numpy"}},
			PackageXML: types.PackageXMLInput{Enabled: true, Tags: []string{"debian_depend"}},
		},
	}
	product := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "prod"},
		Inputs: types.Inputs{
			Manual: types.ManualInputs{Apt: []string{"libprod"}},
		},
	}

	result, err := composer.Compose(context.Background(), product, []types.Spec{profile})
	require.NoError(t, err)

	// Profile apt + product apt
	assert.Equal(t, []string{"libbase", "libprod"}, result.Inputs.Manual.Apt)
	assert.Equal(t, []string{"numpy"}, result.Inputs.Manual.Python)
	assert.True(t, result.Inputs.PackageXML.Enabled)
	assert.Equal(t, []string{"debian_depend"}, result.Inputs.PackageXML.Tags)
}

func TestComposerProductOverridesPublish(t *testing.T) {
	composer := NewProductComposer()
	profile := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "base"},
		Publish:  types.Publish{Repository: types.PublishRepository{Name: "profile-repo"}},
	}
	product := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "prod"},
		Publish:  types.Publish{Repository: types.PublishRepository{Name: "product-repo"}},
	}

	result, err := composer.Compose(context.Background(), product, []types.Spec{profile})
	require.NoError(t, err)
	assert.Equal(t, "product-repo", result.Publish.Repository.Name)
}

func TestComposerMergesResolutionDirectives(t *testing.T) {
	composer := NewProductComposer()
	profile := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "base"},
		Resolutions: []types.ResolutionDirective{
			{Dependency: "apt:libfoo", Action: "force", Value: "1.0.0"},
		},
	}
	product := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "prod"},
		Resolutions: []types.ResolutionDirective{
			{Dependency: "apt:libbar", Action: "force", Value: "2.0.0"},
		},
	}

	result, err := composer.Compose(context.Background(), product, []types.Spec{profile})
	require.NoError(t, err)
	assert.Len(t, result.Resolutions, 2)
}

// ---------------------------------------------------------------------------
// mergePackagingGroups -- duplicate detection
// ---------------------------------------------------------------------------

func TestMergePackagingGroupsRejectsDuplicates(t *testing.T) {
	target := types.Packaging{
		Groups: []types.PackagingGroup{
			{Name: "group-a", Mode: types.PackagingModeIndividual},
		},
	}
	incoming := types.Packaging{
		Groups: []types.PackagingGroup{
			{Name: "group-a", Mode: types.PackagingModeMetaBundle},
		},
	}

	err := mergePackagingGroups(&target, incoming)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate packaging group: group-a")
}

func TestMergePackagingGroupsAcceptsDistinct(t *testing.T) {
	target := types.Packaging{
		Groups: []types.PackagingGroup{
			{Name: "group-a"},
		},
	}
	incoming := types.Packaging{
		Groups: []types.PackagingGroup{
			{Name: "group-b"},
		},
	}

	err := mergePackagingGroups(&target, incoming)
	require.NoError(t, err)
	assert.Len(t, target.Groups, 2)
}

// ---------------------------------------------------------------------------
// validateComposeOrder
// ---------------------------------------------------------------------------

func TestValidateComposeOrderRejectsDuplicates(t *testing.T) {
	refs := []types.ComposeRef{
		{Name: "base", Version: "1.0.0"},
		{Name: "base", Version: "1.0.0"},
	}

	err := validateComposeOrder(refs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate compose entry")
}

func TestValidateComposeOrderAcceptsDifferentVersions(t *testing.T) {
	refs := []types.ComposeRef{
		{Name: "base", Version: "1.0.0"},
		{Name: "base", Version: "2.0.0"},
	}

	err := validateComposeOrder(refs)
	require.NoError(t, err)
}

func TestValidateComposeOrderEmpty(t *testing.T) {
	err := validateComposeOrder(nil)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// mergeInputs
// ---------------------------------------------------------------------------

func TestMergeInputsSetsEnabled(t *testing.T) {
	target := types.Inputs{}
	incoming := types.Inputs{
		PackageXML: types.PackageXMLInput{Enabled: true},
	}

	mergeInputs(&target, incoming)
	assert.True(t, target.PackageXML.Enabled)
}

func TestMergeInputsAppendsTags(t *testing.T) {
	target := types.Inputs{
		PackageXML: types.PackageXMLInput{Tags: []string{"a"}},
	}
	incoming := types.Inputs{
		PackageXML: types.PackageXMLInput{Tags: []string{"b", "c"}},
	}

	mergeInputs(&target, incoming)
	assert.Equal(t, []string{"a", "b", "c"}, target.PackageXML.Tags)
}

func TestMergeInputsSetsIncludeSrc(t *testing.T) {
	target := types.Inputs{}
	incoming := types.Inputs{
		PackageXML: types.PackageXMLInput{IncludeSrc: true},
	}

	mergeInputs(&target, incoming)
	assert.True(t, target.PackageXML.IncludeSrc)
}

func TestMergeInputsAppendsManualDeps(t *testing.T) {
	target := types.Inputs{
		Manual: types.ManualInputs{Apt: []string{"a"}, Python: []string{"x"}},
	}
	incoming := types.Inputs{
		Manual: types.ManualInputs{Apt: []string{"b"}, Python: []string{"y"}},
	}

	mergeInputs(&target, incoming)
	assert.Equal(t, []string{"a", "b"}, target.Manual.Apt)
	assert.Equal(t, []string{"x", "y"}, target.Manual.Python)
}

// ---------------------------------------------------------------------------
// mergeSchema
// ---------------------------------------------------------------------------

func TestMergeSchemaProfileIntoProduct(t *testing.T) {
	composer := NewProductComposer()
	profile := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "base"},
		Schema: &types.SchemaFile{
			SchemaVersion: "v1",
			Mappings: map[string]types.SchemaMapping{
				"rclcpp": {Type: types.DependencyTypeApt, Package: "ros-humble-rclcpp"},
				"numpy":  {Type: types.DependencyTypePip, Package: "numpy"},
			},
		},
	}
	product := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "prod"},
		Schema: &types.SchemaFile{
			SchemaVersion: "v1",
			Mappings: map[string]types.SchemaMapping{
				"numpy": {Type: types.DependencyTypePip, Package: "numpy", Version: ">=2.0"},
				"fmt":   {Type: types.DependencyTypeApt, Package: "libfmt-dev"},
			},
		},
	}

	result, err := composer.Compose(context.Background(), product, []types.Spec{profile})
	require.NoError(t, err)
	require.NotNil(t, result.Schema)
	assert.Len(t, result.Schema.Mappings, 3)
	// Profile's rclcpp preserved
	assert.Equal(t, "ros-humble-rclcpp", result.Schema.Mappings["rclcpp"].Package)
	// Product's numpy overrides profile's numpy
	assert.Equal(t, ">=2.0", result.Schema.Mappings["numpy"].Version)
	// Product's fmt added
	assert.Equal(t, "libfmt-dev", result.Schema.Mappings["fmt"].Package)
}

func TestMergeSchemaNoSchemaOnEitherSide(t *testing.T) {
	composer := NewProductComposer()
	profile := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "base"},
	}
	product := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "prod"},
	}

	result, err := composer.Compose(context.Background(), product, []types.Spec{profile})
	require.NoError(t, err)
	assert.Nil(t, result.Schema)
}

func TestMergeSchemaOnlyOnProduct(t *testing.T) {
	composer := NewProductComposer()
	profile := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "base"},
	}
	product := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "prod"},
		Schema: &types.SchemaFile{
			SchemaVersion: "v1",
			Mappings: map[string]types.SchemaMapping{
				"rclcpp": {Type: types.DependencyTypeApt, Package: "ros-humble-rclcpp"},
			},
		},
	}

	result, err := composer.Compose(context.Background(), product, []types.Spec{profile})
	require.NoError(t, err)
	require.NotNil(t, result.Schema)
	assert.Len(t, result.Schema.Mappings, 1)
	assert.Equal(t, "ros-humble-rclcpp", result.Schema.Mappings["rclcpp"].Package)
}

func TestMergeSchemaMultipleProfilesLayered(t *testing.T) {
	composer := NewProductComposer()
	profile1 := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "p1"},
		Schema: &types.SchemaFile{
			SchemaVersion: "v1",
			Mappings: map[string]types.SchemaMapping{
				"rclcpp": {Type: types.DependencyTypeApt, Package: "old-rclcpp"},
				"a":      {Type: types.DependencyTypeApt, Package: "pkg-a"},
			},
		},
	}
	profile2 := types.Spec{
		Kind:     types.SpecKindProfile,
		Metadata: types.Metadata{Name: "p2"},
		Schema: &types.SchemaFile{
			SchemaVersion: "v1",
			Mappings: map[string]types.SchemaMapping{
				"rclcpp": {Type: types.DependencyTypeApt, Package: "new-rclcpp"},
				"b":      {Type: types.DependencyTypePip, Package: "pkg-b"},
			},
		},
	}
	product := types.Spec{
		Kind:     types.SpecKindProduct,
		Metadata: types.Metadata{Name: "prod"},
		Compose: []types.ComposeRef{
			{Name: "p1", Version: "1"},
			{Name: "p2", Version: "1"},
		},
	}

	result, err := composer.Compose(context.Background(), product, []types.Spec{profile1, profile2})
	require.NoError(t, err)
	require.NotNil(t, result.Schema)
	// profile2 overrides profile1's rclcpp
	assert.Equal(t, "new-rclcpp", result.Schema.Mappings["rclcpp"].Package)
	// Both unique keys present
	assert.Equal(t, "pkg-a", result.Schema.Mappings["a"].Package)
	assert.Equal(t, "pkg-b", result.Schema.Mappings["b"].Package)
}
