package integration

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/adapters"
	"avular-packages/internal/core"
	"avular-packages/internal/policies"
	"avular-packages/tests/testutil"
	"avular-packages/internal/types"
)

// TestGoldenResolve performs a full resolve using the sample fixtures and
// compares the outputs against committed golden files. If the golden files
// do not exist yet (first run), they are written so they can be committed.
//
// To update golden files after an intentional change, delete the
// testdata/golden/ directory and re-run the test.
func TestGoldenResolve(t *testing.T) {
	root := testutil.RepoRoot(t)
	goldenDir := filepath.Join(root, "tests", "integration", "testdata", "golden")

	specAdapter := adapters.NewSpecFileAdapter()
	productPath := filepath.Join(root, "fixtures/product-sample.yaml")
	repoIndex := filepath.Join(root, "fixtures/repo-index.yaml")
	workspace := filepath.Join(root, "fixtures/workspace")

	product, err := specAdapter.LoadProduct(productPath)
	require.NoError(t, err)
	profiles, err := loadProfiles(specAdapter, product, root)
	require.NoError(t, err)

	composer := core.NewProductComposer()
	composed, err := composer.Compose(t.Context(), product, profiles)
	require.NoError(t, err)

	builder := core.NewDependencyBuilder(
		adapters.NewWorkspaceAdapter(),
		adapters.NewPackageXMLAdapter(),
	)
	deps, err := builder.Build(t.Context(), composed.Inputs, []string{workspace})
	require.NoError(t, err)

	policy := policies.NewPackagingPolicy(composed.Packaging.Groups, "24.04")
	resolver := core.NewResolverCore(adapters.NewRepoIndexFileAdapter(repoIndex), policy)
	result, err := resolver.Resolve(t.Context(), deps, composed.Resolutions)
	require.NoError(t, err)

	outDir := t.TempDir()
	output := adapters.NewOutputFileAdapter(outDir)
	require.NoError(t, output.WriteAptLock(result.AptLocks))
	require.NoError(t, output.WriteBundleManifest(result.BundleManifest))
	require.NoError(t, output.WriteResolutionReport(result.Resolution))

	// Compare each output against golden file
	goldenFiles := map[string]string{
		"apt.lock":          filepath.Join(outDir, "apt.lock"),
		"bundle.manifest":   filepath.Join(outDir, "bundle.manifest"),
		"resolution.report": filepath.Join(outDir, "resolution.report"),
	}

	for name, actualPath := range goldenFiles {
		t.Run(name, func(t *testing.T) {
			actual, err := os.ReadFile(actualPath)
			require.NoError(t, err)

			goldenPath := filepath.Join(goldenDir, name)
			if _, statErr := os.Stat(goldenPath); os.IsNotExist(statErr) {
				// Golden file doesn't exist yet -- write it.
				require.NoError(t, os.MkdirAll(goldenDir, 0o755))
				require.NoError(t, os.WriteFile(goldenPath, actual, 0o644))
				t.Logf("golden file written: %s (commit it)", goldenPath)
				return
			}

			expected, err := os.ReadFile(goldenPath)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual),
				"golden mismatch for %s -- delete testdata/golden/ and re-run to regenerate", name)
		})
	}
}

// TestGoldenResolveStructure verifies the structural properties of the
// resolve output independent of exact values -- counts, names present, etc.
func TestGoldenResolveStructure(t *testing.T) {
	root := testutil.RepoRoot(t)

	specAdapter := adapters.NewSpecFileAdapter()
	productPath := filepath.Join(root, "fixtures/product-sample.yaml")
	repoIndex := filepath.Join(root, "fixtures/repo-index.yaml")
	workspace := filepath.Join(root, "fixtures/workspace")

	product, err := specAdapter.LoadProduct(productPath)
	require.NoError(t, err)
	profiles, err := loadProfiles(specAdapter, product, root)
	require.NoError(t, err)

	composer := core.NewProductComposer()
	composed, err := composer.Compose(t.Context(), product, profiles)
	require.NoError(t, err)

	builder := core.NewDependencyBuilder(
		adapters.NewWorkspaceAdapter(),
		adapters.NewPackageXMLAdapter(),
	)
	deps, err := builder.Build(t.Context(), composed.Inputs, []string{workspace})
	require.NoError(t, err)

	policy := policies.NewPackagingPolicy(composed.Packaging.Groups, "24.04")
	resolver := core.NewResolverCore(adapters.NewRepoIndexFileAdapter(repoIndex), policy)
	result, err := resolver.Resolve(t.Context(), deps, composed.Resolutions)
	require.NoError(t, err)

	t.Run("apt locks are sorted", func(t *testing.T) {
		names := make([]string, 0, len(result.AptLocks))
		for _, entry := range result.AptLocks {
			names = append(names, entry.Package)
		}
		sorted := make([]string, len(names))
		copy(sorted, names)
		sort.Strings(sorted)
		assert.Equal(t, sorted, names, "apt locks must be sorted by package name")
	})

	t.Run("expected packages resolved", func(t *testing.T) {
		resolved := map[string]string{}
		for _, entry := range result.AptLocks {
			resolved[entry.Package] = entry.Version
		}
		// From the fixture: libfoo, libbar, python3-requests
		assert.Contains(t, resolved, "libfoo")
		assert.Contains(t, resolved, "libbar")
		assert.Contains(t, resolved, "python3-requests")
	})

	t.Run("bundle manifest has entries for each group", func(t *testing.T) {
		groups := map[string]struct{}{}
		for _, entry := range result.BundleManifest {
			groups[entry.Group] = struct{}{}
		}
		assert.Contains(t, groups, "apt-individual")
		assert.Contains(t, groups, "pip-meta")
	})

	t.Run("resolved deps contain all types", func(t *testing.T) {
		aptFound := false
		pipFound := false
		for _, dep := range result.ResolvedDeps {
			if dep.Type == types.DependencyTypeApt {
				aptFound = true
			}
			if dep.Type == types.DependencyTypePip {
				pipFound = true
			}
		}
		assert.True(t, aptFound, "should have at least one apt resolved dep")
		assert.True(t, pipFound, "should have at least one pip resolved dep")
	})

	t.Run("versions are from repo index", func(t *testing.T) {
		resolved := map[string]string{}
		for _, entry := range result.AptLocks {
			resolved[entry.Package] = entry.Version
		}
		// libfoo should pick the highest: 1.1.0
		assert.Equal(t, "1.1.0", resolved["libfoo"])
		// libbar has only one version: 2.0.0
		assert.Equal(t, "2.0.0", resolved["libbar"])
		// requests is pinned to 2.31.0 in the package.xml
		assert.Equal(t, "2.31.0", resolved["python3-requests"])
	})
}

// TestGoldenDependencyBuilder verifies that the dependency builder
// correctly extracts dependencies from fixtures.
func TestGoldenDependencyBuilder(t *testing.T) {
	root := testutil.RepoRoot(t)
	workspace := filepath.Join(root, "fixtures/workspace")

	specAdapter := adapters.NewSpecFileAdapter()
	productPath := filepath.Join(root, "fixtures/product-sample.yaml")

	product, err := specAdapter.LoadProduct(productPath)
	require.NoError(t, err)
	profiles, err := loadProfiles(specAdapter, product, root)
	require.NoError(t, err)

	composer := core.NewProductComposer()
	composed, err := composer.Compose(t.Context(), product, profiles)
	require.NoError(t, err)

	builder := core.NewDependencyBuilder(
		adapters.NewWorkspaceAdapter(),
		adapters.NewPackageXMLAdapter(),
	)
	deps, err := builder.Build(t.Context(), composed.Inputs, []string{workspace})
	require.NoError(t, err)

	depNames := make([]string, 0, len(deps))
	for _, dep := range deps {
		depNames = append(depNames, dep.Name)
	}

	t.Run("contains expected deps from package.xml", func(t *testing.T) {
		assert.Contains(t, depNames, "libfoo")
		assert.Contains(t, depNames, "libbar")
		assert.Contains(t, depNames, "requests")
	})

	t.Run("dep types are correct", func(t *testing.T) {
		typeMap := map[string]types.DependencyType{}
		for _, dep := range deps {
			typeMap[dep.Name] = dep.Type
		}
		assert.Equal(t, types.DependencyTypeApt, typeMap["libfoo"])
		assert.Equal(t, types.DependencyTypeApt, typeMap["libbar"])
		assert.Equal(t, types.DependencyTypePip, typeMap["requests"])
	})

	t.Run("requests has version constraint", func(t *testing.T) {
		for _, dep := range deps {
			if dep.Name == "requests" {
				found := false
				for _, c := range dep.Constraints {
					if strings.Contains(c.Version, "2.31.0") {
						found = true
					}
				}
				assert.True(t, found, "requests should have a 2.31.0 version constraint")
				return
			}
		}
		t.Fatal("requests dep not found")
	})
}
