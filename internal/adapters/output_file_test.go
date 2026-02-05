package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestOutputFileAdapterFormats(t *testing.T) {
	dir := t.TempDir()
	adapter := NewOutputFileAdapter(dir)

	err := adapter.WriteAptLock([]types.AptLockEntry{
		{Package: "libb", Version: "2.0.0"},
		{Package: "liba", Version: "1.0.0"},
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "apt.lock"))
	require.NoError(t, err)
	if diff := cmp.Diff("liba=1.0.0\nlibb=2.0.0", strings.TrimSpace(string(data))); diff != "" {
		t.Fatalf("unexpected apt.lock content (-want +got):\n%s", diff)
	}

	err = adapter.WriteBundleManifest([]types.BundleManifestEntry{
		{Group: "g1", Mode: types.PackagingModeMetaBundle, Package: "pkg", Version: "1.0.0"},
	})
	require.NoError(t, err)

	err = adapter.WriteSnapshotIntent(types.SnapshotIntent{
		Repository:     "avular",
		Channel:        "dev",
		SnapshotPrefix: "pfx",
		SnapshotID:     "pfx-123",
		CreatedAt:      "2026-01-27T00:00:00Z",
	})
	require.NoError(t, err)

	intent, err := os.ReadFile(filepath.Join(dir, "snapshot.intent"))
	require.NoError(t, err)
	containsChecks := []struct {
		name string
		got  bool
		want bool
	}{
		{name: "repository", got: strings.Contains(string(intent), "repository=avular"), want: true},
		{name: "snapshot id", got: strings.Contains(string(intent), "snapshot_id=pfx-123"), want: true},
	}
	for _, tt := range containsChecks {
		if diff := cmp.Diff(tt.want, tt.got); diff != "" {
			t.Fatalf("unexpected snapshot.intent %s (-want +got):\n%s", tt.name, diff)
		}
	}

	err = adapter.WriteResolutionReport(types.ResolutionReport{
		Records: []types.ResolutionRecord{
			{Dependency: "apt:liba", Action: "force", Value: "1.0.0", Reason: "test", Owner: "team"},
		},
	})
	require.NoError(t, err)
}
