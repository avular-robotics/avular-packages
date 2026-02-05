package core

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/adapters"
	"avular-packages/internal/types"
)

const samplePackageXML = `<?xml version="1.0"?>
<package format="3">
  <name>sample_pkg</name>
  <version>0.1.0</version>
  <description>Sample</description>
  <maintainer email="dev@example.com">Dev</maintainer>
  <license>MIT</license>
  <export>
    <debian_depend>sample_pkg</debian_depend>
    <debian_depend>ros-sample-pkg</debian_depend>
    <debian_depend>libfoo</debian_depend>
  </export>
</package>
`

func TestDependencyBuilderFiltersWorkspaceDeps(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	require.NoError(t, os.MkdirAll(ws, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ws, "package.xml"), []byte(samplePackageXML), 0644))

	inputs := types.Inputs{
		PackageXML: types.PackageXMLInput{
			Enabled:    true,
			Tags:       []string{"debian_depend"},
			IncludeSrc: false,
			Prefix:     "ros-",
		},
	}

	builder := NewDependencyBuilder(adapters.NewWorkspaceAdapter(), adapters.NewPackageXMLAdapter())
	deps, err := builder.Build(t.Context(), inputs, []string{ws})
	require.NoError(t, err)

	var names []string
	for _, dep := range deps {
		names = append(names, dep.Name)
	}
	sort.Strings(names)
	expected := []string{"libfoo"}
	if diff := cmp.Diff(expected, names); diff != "" {
		t.Fatalf("unexpected dependency names (-want +got):\n%s", diff)
	}
}

func TestDependencyBuilderNormalizesPipNames(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantName     string
		wantType     types.DependencyType
		wantDepCount int
	}{
		{
			name:         "upper case",
			raw:          "Requests==2.31.0",
			wantName:     "requests",
			wantType:     types.DependencyTypePip,
			wantDepCount: 1,
		},
		{
			name:         "dot normalization",
			raw:          "Django.Filter==3.0.0",
			wantName:     "django-filter",
			wantType:     types.DependencyTypePip,
			wantDepCount: 1,
		},
		{
			name:         "underscore normalization",
			raw:          "my_pkg==1.0.0",
			wantName:     "my-pkg",
			wantType:     types.DependencyTypePip,
			wantDepCount: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			inputs := types.Inputs{
				Manual: types.ManualInputs{
					Python: []string{tt.raw},
				},
			}

			builder := NewDependencyBuilder(adapters.NewWorkspaceAdapter(), adapters.NewPackageXMLAdapter())
			deps, err := builder.Build(t.Context(), inputs, nil)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.wantDepCount, len(deps)); diff != "" {
				t.Fatalf("unexpected dependency count (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantType, deps[0].Type); diff != "" {
				t.Fatalf("unexpected dependency type (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantName, deps[0].Name); diff != "" {
				t.Fatalf("unexpected dependency name (-want +got):\n%s", diff)
			}
		})
	}
}
