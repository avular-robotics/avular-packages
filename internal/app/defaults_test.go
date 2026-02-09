package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"avular-packages/internal/types"
)

func TestApplySpecDefaults(t *testing.T) {
	defaults := types.SpecDefaults{
		TargetUbuntu: "24.04",
		Workspace:    []string{"./src"},
		RepoIndex:    "./repo-index.yaml",
		Output:       "build-out",
	}

	tests := []struct {
		name     string
		req      ResolveRequest
		expected ResolveRequest
	}{
		{
			name: "empty request gets all defaults",
			req:  ResolveRequest{},
			expected: ResolveRequest{
				TargetUbuntu: "24.04",
				Workspace:    []string{"./src"},
				RepoIndex:    "./repo-index.yaml",
				OutputDir:    "build-out",
			},
		},
		{
			name: "explicit values override defaults",
			req: ResolveRequest{
				TargetUbuntu: "22.04",
				Workspace:    []string{"/custom/ws"},
				RepoIndex:    "/custom/repo.yaml",
				OutputDir:    "/custom/out",
			},
			expected: ResolveRequest{
				TargetUbuntu: "22.04",
				Workspace:    []string{"/custom/ws"},
				RepoIndex:    "/custom/repo.yaml",
				OutputDir:    "/custom/out",
			},
		},
		{
			name: "partial override mixes with defaults",
			req: ResolveRequest{
				TargetUbuntu: "22.04",
				// Workspace, RepoIndex, OutputDir left empty -> defaults apply
			},
			expected: ResolveRequest{
				TargetUbuntu: "22.04",
				Workspace:    []string{"./src"},
				RepoIndex:    "./repo-index.yaml",
				OutputDir:    "build-out",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := applySpecDefaults(tc.req, defaults)
			assert.Equal(t, tc.expected.TargetUbuntu, result.TargetUbuntu)
			assert.Equal(t, tc.expected.Workspace, result.Workspace)
			assert.Equal(t, tc.expected.RepoIndex, result.RepoIndex)
			assert.Equal(t, tc.expected.OutputDir, result.OutputDir)
		})
	}
}

func TestApplySpecDefaultsEmpty(t *testing.T) {
	// Empty defaults should not change anything
	req := ResolveRequest{
		TargetUbuntu: "22.04",
		Workspace:    []string{"/ws"},
	}

	result := applySpecDefaults(req, types.SpecDefaults{})
	assert.Equal(t, "22.04", result.TargetUbuntu)
	assert.Equal(t, []string{"/ws"}, result.Workspace)
	assert.Empty(t, result.RepoIndex)
	assert.Empty(t, result.OutputDir)
}

func TestDiscoverProduct(t *testing.T) {
	// discoverProduct looks in current directory; in test context
	// there's no product.yaml so it should return empty
	result := discoverProduct()
	assert.Empty(t, result)
}

func TestDiscoverSchemaFilesNoDir(t *testing.T) {
	// No schemas/ directory exists -> nil
	result := discoverSchemaFiles("nonexistent/product.yaml")
	assert.Nil(t, result)
}

func TestDiscoverSchemaFilesEmptyPath(t *testing.T) {
	result := discoverSchemaFiles("")
	assert.Nil(t, result)
}

func TestDiscoverSchemaFilesWithDir(t *testing.T) {
	dir := t.TempDir()
	schemasDir := dir + "/schemas"
	assert.NoError(t, os.MkdirAll(schemasDir, 0755))
	assert.NoError(t, os.WriteFile(schemasDir+"/b.yaml", []byte("schema_version: v1\nmappings: {}"), 0644))
	assert.NoError(t, os.WriteFile(schemasDir+"/a.yml", []byte("schema_version: v1\nmappings: {}"), 0644))
	assert.NoError(t, os.WriteFile(schemasDir+"/readme.txt", []byte("not a schema"), 0644))

	productPath := dir + "/product.yaml"
	result := discoverSchemaFiles(productPath)
	// Should find .yaml and .yml, sorted alphabetically, skip .txt
	assert.Len(t, result, 2)
	assert.Contains(t, result[0], "a.yml")
	assert.Contains(t, result[1], "b.yaml")
}

func TestValidateInlineSchemaValid(t *testing.T) {
	schema := types.SchemaFile{
		SchemaVersion: "v1",
		Mappings: map[string]types.SchemaMapping{
			"rclcpp": {Type: types.DependencyTypeApt, Package: "ros-humble-rclcpp"},
		},
	}
	assert.NoError(t, validateInlineSchema(schema))
}

func TestValidateInlineSchemaMissingVersion(t *testing.T) {
	schema := types.SchemaFile{
		Mappings: map[string]types.SchemaMapping{
			"rclcpp": {Type: types.DependencyTypeApt, Package: "ros-humble-rclcpp"},
		},
	}
	err := validateInlineSchema(schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema_version")
}

func TestValidateInlineSchemaInvalidType(t *testing.T) {
	schema := types.SchemaFile{
		SchemaVersion: "v1",
		Mappings: map[string]types.SchemaMapping{
			"rclcpp": {Type: "npm", Package: "rclcpp"},
		},
	}
	err := validateInlineSchema(schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestValidateInlineSchemaEmptyPackage(t *testing.T) {
	schema := types.SchemaFile{
		SchemaVersion: "v1",
		Mappings: map[string]types.SchemaMapping{
			"rclcpp": {Type: types.DependencyTypeApt, Package: ""},
		},
	}
	err := validateInlineSchema(schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty package")
}

func TestValidateInlineProfileValid(t *testing.T) {
	profile := types.InlineProfile{
		Packaging: types.Packaging{
			Groups: []types.PackagingGroup{
				{Name: "apt-runtime", Mode: types.PackagingModeIndividual},
			},
		},
	}
	assert.NoError(t, validateInlineProfile("test", profile))
}

func TestValidateInlineProfileNoGroups(t *testing.T) {
	profile := types.InlineProfile{
		Packaging: types.Packaging{},
	}
	err := validateInlineProfile("test", profile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one packaging group")
}

func TestValidateInlineProfileEmptyGroupName(t *testing.T) {
	profile := types.InlineProfile{
		Packaging: types.Packaging{
			Groups: []types.PackagingGroup{
				{Name: "", Mode: types.PackagingModeIndividual},
			},
		},
	}
	err := validateInlineProfile("test", profile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty name")
}

func TestApplyBuildDefaults(t *testing.T) {
	defaults := types.SpecDefaults{
		TargetUbuntu:   "24.04",
		Workspace:      []string{"./src"},
		RepoIndex:      "./repo-index.yaml",
		Output:         "build-out",
		PipIndexURL:    "https://pip.example.com/simple",
		InternalDebDir: "./prebuilt",
		InternalSrc:    []string{"./internal-pkg"},
	}

	t.Run("empty request gets all defaults", func(t *testing.T) {
		req := BuildRequest{}
		result := applyBuildDefaults(req, defaults)
		assert.Equal(t, "24.04", result.TargetUbuntu)
		assert.Equal(t, []string{"./src"}, result.Workspace)
		assert.Equal(t, "./repo-index.yaml", result.RepoIndex)
		assert.Equal(t, "build-out", result.OutputDir)
		assert.Equal(t, "https://pip.example.com/simple", result.PipIndexURL)
		assert.Equal(t, "./prebuilt", result.InternalDebDir)
		assert.Equal(t, []string{"./internal-pkg"}, result.InternalSrc)
	})

	t.Run("explicit values override defaults", func(t *testing.T) {
		req := BuildRequest{
			TargetUbuntu:   "22.04",
			PipIndexURL:    "https://custom.pip/simple",
			InternalDebDir: "/custom/debs",
			InternalSrc:    []string{"/custom/src"},
		}
		result := applyBuildDefaults(req, defaults)
		assert.Equal(t, "22.04", result.TargetUbuntu)
		assert.Equal(t, "https://custom.pip/simple", result.PipIndexURL)
		assert.Equal(t, "/custom/debs", result.InternalDebDir)
		assert.Equal(t, []string{"/custom/src"}, result.InternalSrc)
	})

	t.Run("empty defaults leave request unchanged", func(t *testing.T) {
		req := BuildRequest{PipIndexURL: "https://pip.test"}
		result := applyBuildDefaults(req, types.SpecDefaults{})
		assert.Equal(t, "https://pip.test", result.PipIndexURL)
		assert.Empty(t, result.TargetUbuntu)
		assert.Empty(t, result.InternalDebDir)
	})
}

func TestCheckResolveDefaultsHints(t *testing.T) {
	defaults := types.SpecDefaults{
		TargetUbuntu: "24.04",
		RepoIndex:    "./repo.yaml",
	}

	t.Run("no hints when request is empty", func(t *testing.T) {
		hints := checkResolveDefaultsHints(ResolveRequest{}, defaults)
		assert.Empty(t, hints)
	})

	t.Run("hints when flag duplicates default", func(t *testing.T) {
		req := ResolveRequest{
			TargetUbuntu: "24.04",
			RepoIndex:    "./repo.yaml",
		}
		hints := checkResolveDefaultsHints(req, defaults)
		assert.Len(t, hints, 2)
		assert.Contains(t, hints[0], "--target-ubuntu")
		assert.Contains(t, hints[1], "--repo-index")
	})

	t.Run("no hints when default is empty", func(t *testing.T) {
		req := ResolveRequest{TargetUbuntu: "24.04"}
		hints := checkResolveDefaultsHints(req, types.SpecDefaults{})
		assert.Empty(t, hints)
	})
}

func TestCheckBuildDefaultsHints(t *testing.T) {
	defaults := types.SpecDefaults{
		PipIndexURL:    "https://pip.example.com/simple",
		InternalDebDir: "./prebuilt",
	}

	t.Run("build-specific hints emitted", func(t *testing.T) {
		req := BuildRequest{
			PipIndexURL:    "https://pip.example.com/simple",
			InternalDebDir: "./prebuilt",
		}
		hints := checkBuildDefaultsHints(req, defaults)
		assert.Len(t, hints, 2)
		assert.Contains(t, hints[0], "--pip-index-url")
		assert.Contains(t, hints[1], "--internal-deb-dir")
	})

	t.Run("no hints for empty request", func(t *testing.T) {
		hints := checkBuildDefaultsHints(BuildRequest{}, defaults)
		assert.Empty(t, hints)
	})
}
