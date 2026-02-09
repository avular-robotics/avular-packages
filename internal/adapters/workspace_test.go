package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceAdapter_FindPackageXML(t *testing.T) {
	root := t.TempDir()
	// Create nested package.xml files.
	pkgA := filepath.Join(root, "src", "pkg_a")
	pkgB := filepath.Join(root, "src", "pkg_b")
	require.NoError(t, os.MkdirAll(pkgA, 0755))
	require.NoError(t, os.MkdirAll(pkgB, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "package.xml"), []byte("<package/>"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "package.xml"), []byte("<package/>"), 0644))
	// Random other file should be ignored.
	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "CMakeLists.txt"), []byte("cmake"), 0644))

	adapter := NewWorkspaceAdapter()
	paths, err := adapter.FindPackageXML(root)
	require.NoError(t, err)
	assert.Len(t, paths, 2)
}

func TestWorkspaceAdapter_SkipsBuildDirs(t *testing.T) {
	root := t.TempDir()
	// Create a package.xml inside a build directory -- should be skipped.
	for _, dir := range []string{"build", "install", "log", ".git"} {
		ignored := filepath.Join(root, dir, "pkg")
		require.NoError(t, os.MkdirAll(ignored, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(ignored, "package.xml"), []byte("<package/>"), 0644))
	}
	// Create a real one.
	real := filepath.Join(root, "src", "real_pkg")
	require.NoError(t, os.MkdirAll(real, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(real, "package.xml"), []byte("<package/>"), 0644))

	adapter := NewWorkspaceAdapter()
	paths, err := adapter.FindPackageXML(root)
	require.NoError(t, err)
	assert.Len(t, paths, 1)
	assert.Contains(t, paths[0], "real_pkg")
}

func TestWorkspaceAdapter_EmptyRootErrors(t *testing.T) {
	adapter := NewWorkspaceAdapter()
	_, err := adapter.FindPackageXML("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace root is empty")
}

func TestWorkspaceAdapter_NonExistentRootErrors(t *testing.T) {
	adapter := NewWorkspaceAdapter()
	_, err := adapter.FindPackageXML("/nonexistent/path/that/does/not/exist")
	require.Error(t, err)
}

func TestWorkspaceAdapter_EmptyWorkspaceReturnsNil(t *testing.T) {
	root := t.TempDir()
	adapter := NewWorkspaceAdapter()
	paths, err := adapter.FindPackageXML(root)
	require.NoError(t, err)
	assert.Nil(t, paths)
}
