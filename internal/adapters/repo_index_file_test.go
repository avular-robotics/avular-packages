package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/shared"
	"avular-packages/internal/types"
)

func TestRepoIndexFileAdapter_AvailableVersions(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "repo-index.yaml")
	content := `
apt:
  libfoo:
    - "1.0"
    - "2.0"
  libbar:
    - "3.0"
pip:
  requests:
    - "2.28.0"
    - "2.31.0"
  my_package:
    - "1.0.0"
`
	require.NoError(t, os.WriteFile(indexPath, []byte(content), 0o644))

	adapter := NewRepoIndexFileAdapter(indexPath)

	t.Run("apt known package", func(t *testing.T) {
		versions, err := adapter.AvailableVersions(types.DependencyTypeApt, "libfoo")
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0", "2.0"}, versions)
	})

	t.Run("apt unknown package", func(t *testing.T) {
		versions, err := adapter.AvailableVersions(types.DependencyTypeApt, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, versions)
	})

	t.Run("pip exact name", func(t *testing.T) {
		versions, err := adapter.AvailableVersions(types.DependencyTypePip, "requests")
		require.NoError(t, err)
		assert.Equal(t, []string{"2.28.0", "2.31.0"}, versions)
	})

	t.Run("pip normalized name lookup", func(t *testing.T) {
		// The index stores "my_package" as-is. When we look up
		// "My_Package" it normalizes to "my-package" (underscores become
		// hyphens). That won't match "my_package". The lookup falls
		// through and returns nil -- this is correct behavior since the
		// index keys must already be in normalized form for cross-name
		// matching to work.
		versions, err := adapter.AvailableVersions(types.DependencyTypePip, "My_Package")
		require.NoError(t, err)
		assert.Nil(t, versions)
	})

	t.Run("pip normalized key in index matches", func(t *testing.T) {
		// Build a separate adapter with a normalized pip key.
		dir2 := t.TempDir()
		idx2 := filepath.Join(dir2, "repo-index.yaml")
		content2 := "pip:\n  my-package:\n    - \"1.0.0\"\n"
		require.NoError(t, os.WriteFile(idx2, []byte(content2), 0o644))
		adapter2 := NewRepoIndexFileAdapter(idx2)

		versions, err := adapter2.AvailableVersions(types.DependencyTypePip, "My_Package")
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0"}, versions)
	})

	t.Run("unknown dependency type", func(t *testing.T) {
		_, err := adapter.AvailableVersions("unknown", "libfoo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown dependency type")
	})
}

func TestRepoIndexFileAdapter_AptPackages(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "repo-index.yaml")
	content := `
apt_packages:
  libfoo:
    - version: "1.0"
      depends: ["libbar (>= 1.0)"]
    - version: "2.0"
`
	require.NoError(t, os.WriteFile(indexPath, []byte(content), 0o644))

	adapter := NewRepoIndexFileAdapter(indexPath)
	packages, err := adapter.AptPackages()
	require.NoError(t, err)
	require.Contains(t, packages, "libfoo")
	assert.Len(t, packages["libfoo"], 2)
	assert.Equal(t, "1.0", packages["libfoo"][0].Version)
	assert.Equal(t, []string{"libbar (>= 1.0)"}, packages["libfoo"][0].Depends)
}

func TestRepoIndexFileAdapter_EmptyAptPackages(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "repo-index.yaml")
	content := `
apt:
  libfoo:
    - "1.0"
`
	require.NoError(t, os.WriteFile(indexPath, []byte(content), 0o644))

	adapter := NewRepoIndexFileAdapter(indexPath)
	packages, err := adapter.AptPackages()
	require.NoError(t, err)
	assert.Empty(t, packages)
}

func TestRepoIndexFileAdapter_Caching(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "repo-index.yaml")
	content := `
apt:
  libfoo:
    - "1.0"
`
	require.NoError(t, os.WriteFile(indexPath, []byte(content), 0o644))

	adapter := NewRepoIndexFileAdapter(indexPath)

	v1, err := adapter.AvailableVersions(types.DependencyTypeApt, "libfoo")
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0"}, v1)

	// Remove the file -- should still work from cache
	require.NoError(t, os.Remove(indexPath))

	v2, err := adapter.AvailableVersions(types.DependencyTypeApt, "libfoo")
	require.NoError(t, err)
	assert.Equal(t, v1, v2)
}

func TestRepoIndexFileAdapter_MissingFile(t *testing.T) {
	adapter := NewRepoIndexFileAdapter("/nonexistent/path/repo-index.yaml")
	_, err := adapter.AvailableVersions(types.DependencyTypeApt, "libfoo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo index file not found")
}

func TestRepoIndexFileAdapter_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "repo-index.yaml")
	require.NoError(t, os.WriteFile(indexPath, []byte("{{{{invalid yaml"), 0o644))

	adapter := NewRepoIndexFileAdapter(indexPath)
	_, err := adapter.AvailableVersions(types.DependencyTypeApt, "libfoo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo index format")
}

func TestRepoIndexFileAdapter_AptPackagesPopulatesAptVersions(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "repo-index.yaml")
	// When only apt_packages is provided (no apt), the loader should
	// back-fill the simple apt version map.
	content := `
apt_packages:
  libfoo:
    - version: "2.0"
    - version: "1.0"
`
	require.NoError(t, os.WriteFile(indexPath, []byte(content), 0o644))

	adapter := NewRepoIndexFileAdapter(indexPath)
	versions, err := adapter.AvailableVersions(types.DependencyTypeApt, "libfoo")
	require.NoError(t, err)
	assert.Len(t, versions, 2)
	assert.Contains(t, versions, "1.0")
	assert.Contains(t, versions, "2.0")
}

func TestNormalizePipName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"requests", "requests"},
		{"My_Package", "my-package"},
		{"Some.Lib", "some-lib"},
		{"  Mixed_Case.Pkg  ", "mixed-case-pkg"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, shared.NormalizePipName(tt.input))
		})
	}
}
