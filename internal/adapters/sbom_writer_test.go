package adapters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestSBOMWriterAdapter_WriteSBOM(t *testing.T) {
	dir := t.TempDir()
	adapter := NewSBOMWriterAdapter()

	locks := []types.AptLockEntry{
		{Package: "zlib1g", Version: "1:1.2.13-1"},
		{Package: "curl", Version: "8.5.0-1"},
	}

	err := adapter.WriteSBOM(dir, "snap-20260101", "2026-01-01T00:00:00Z", locks)
	require.NoError(t, err)

	path := filepath.Join(dir, "snapshots", "snap-20260101.sbom.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var doc struct {
		SPDXVersion       string `json:"SPDXVersion"`
		DataLicense       string `json:"dataLicense"`
		DocumentNamespace string `json:"documentNamespace"`
		Name              string `json:"name"`
		Packages          []struct {
			Name        string `json:"name"`
			VersionInfo string `json:"versionInfo"`
		} `json:"packages"`
	}
	require.NoError(t, json.Unmarshal(data, &doc))

	assert.Equal(t, "SPDX-2.3", doc.SPDXVersion)
	assert.Contains(t, doc.DocumentNamespace, DefaultSBOMNamespace)
	assert.Contains(t, doc.DocumentNamespace, "snap-20260101")
	assert.Equal(t, "avular-packages snapshot snap-20260101", doc.Name)
	require.Len(t, doc.Packages, 2)
	// Packages are sorted alphabetically.
	assert.Equal(t, "curl", doc.Packages[0].Name)
	assert.Equal(t, "8.5.0-1", doc.Packages[0].VersionInfo)
	assert.Equal(t, "zlib1g", doc.Packages[1].Name)
}

func TestSBOMWriterAdapter_CustomNamespace(t *testing.T) {
	dir := t.TempDir()
	adapter := SBOMWriterAdapter{NamespaceBase: "https://custom.example.com/sbom"}

	err := adapter.WriteSBOM(dir, "snap-1", "2026-01-01T00:00:00Z", nil)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "snapshots", "snap-1.sbom.json"))
	require.NoError(t, err)

	var doc struct {
		DocumentNamespace string `json:"documentNamespace"`
	}
	require.NoError(t, json.Unmarshal(data, &doc))
	assert.Equal(t, "https://custom.example.com/sbom/snap-1", doc.DocumentNamespace)
}

func TestSBOMWriterAdapter_EmptyNamespaceFallsBack(t *testing.T) {
	adapter := SBOMWriterAdapter{NamespaceBase: ""}
	assert.True(t, strings.HasPrefix(adapter.namespaceBase(), "https://avular.dev"))
}

func TestSBOMWriterAdapter_EmptyRepoDirErrors(t *testing.T) {
	adapter := NewSBOMWriterAdapter()
	err := adapter.WriteSBOM("", "snap-1", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo directory is empty")
}

func TestSBOMWriterAdapter_EmptySnapshotIDErrors(t *testing.T) {
	adapter := NewSBOMWriterAdapter()
	err := adapter.WriteSBOM(t.TempDir(), "", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot id is empty")
}

func TestSBOMWriterAdapter_DirectoryPermissions(t *testing.T) {
	dir := t.TempDir()
	adapter := NewSBOMWriterAdapter()
	err := adapter.WriteSBOM(dir, "perm-test", "2026-01-01T00:00:00Z", nil)
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(dir, "snapshots"))
	require.NoError(t, err)
	perm := info.Mode().Perm()
	// Verify group-writable bit is not set (0o750).
	assert.Zero(t, perm&0o002, "world-writable bit should not be set")
}
