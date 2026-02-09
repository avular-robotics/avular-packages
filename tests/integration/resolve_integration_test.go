package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"avular-packages/internal/adapters"
	"avular-packages/internal/core"
	"avular-packages/internal/policies"
	"avular-packages/internal/types"
)

func TestResolveIntegration(t *testing.T) {
	root := repoRoot(t)
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

	builder := core.NewDependencyBuilder(adapters.NewWorkspaceAdapter(), adapters.NewPackageXMLAdapter())
	deps, err := builder.Build(t.Context(), composed.Inputs, []string{workspace})
	require.NoError(t, err)

	policy := policies.NewPackagingPolicy(composed.Packaging.Groups, "24.04")
	resolver := core.NewResolverCore(adapters.NewRepoIndexFileAdapter(repoIndex), policy)
	result, err := resolver.Resolve(t.Context(), deps, composed.Resolutions)
	require.NoError(t, err)
	require.NotEmpty(t, result.AptLocks)

	outDir := t.TempDir()
	output := adapters.NewOutputFileAdapter(outDir)
	require.NoError(t, output.WriteAptLock(result.AptLocks))
	require.NoError(t, output.WriteBundleManifest(result.BundleManifest))

	_, err = os.Stat(filepath.Join(outDir, "apt.lock"))
	require.NoError(t, err)
}

func loadProfiles(adapter adapters.SpecFileAdapter, product types.Spec, root string) ([]types.Spec, error) {
	var profiles []types.Spec
	for _, compose := range product.Compose {
		if compose.Source != "local" {
			return nil, errUnsupportedComposeSource
		}
		specPath := filepath.Join(root, compose.Path)
		spec, err := adapter.LoadProfile(specPath)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, spec)
	}
	return profiles, nil
}

func repoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}

var errUnsupportedComposeSource = os.ErrInvalid
