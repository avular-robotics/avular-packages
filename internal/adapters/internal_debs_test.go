package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDebs_CopiesDebFiles(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create two .deb files and one non-deb file in src
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "foo_1.0_amd64.deb"), []byte("fakepkg"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "bar_2.0_amd64.deb"), []byte("fakepkg2"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "readme.txt"), []byte("not a deb"), 0o644))

	adapter := NewInternalDebsAdapter()
	err := adapter.CopyDebs(srcDir, destDir)
	require.NoError(t, err)

	// Verify .deb files were copied
	entries, err := os.ReadDir(destDir)
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Contains(t, names, "foo_1.0_amd64.deb")
	assert.Contains(t, names, "bar_2.0_amd64.deb")
	assert.NotContains(t, names, "readme.txt")
}

func TestCopyDebs_SkipsDirectories(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create a subdirectory and a .deb file
	require.NoError(t, os.Mkdir(filepath.Join(srcDir, "subdir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "pkg.deb"), []byte("deb"), 0o644))

	adapter := NewInternalDebsAdapter()
	err := adapter.CopyDebs(srcDir, destDir)
	require.NoError(t, err)

	entries, err := os.ReadDir(destDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "pkg.deb", entries[0].Name())
}

func TestCopyDebs_NonexistentSrcDir(t *testing.T) {
	adapter := NewInternalDebsAdapter()
	err := adapter.CopyDebs("/nonexistent/path", t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal deb dir not found")
}

func TestCopyDebs_EmptyDir(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	adapter := NewInternalDebsAdapter()
	err := adapter.CopyDebs(srcDir, destDir)
	require.NoError(t, err)

	entries, err := os.ReadDir(destDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestCopyDebFile_ContentPreserved(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	payload := []byte("binary-deb-content-here")

	srcPath := filepath.Join(srcDir, "test.deb")
	destPath := filepath.Join(destDir, "test.deb")
	require.NoError(t, os.WriteFile(srcPath, payload, 0o644))

	err := copyDebFile(srcPath, destPath)
	require.NoError(t, err)

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}
