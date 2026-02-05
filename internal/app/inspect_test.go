package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestInspectApp(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "apt.lock"), []byte("libfoo=1.0.0\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bundle.manifest"), []byte("group,individual,libfoo,1.0.0\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "resolution.report"), []byte("libfoo,pin,1.0.0,reason,owner\n"), 0644))

	service := NewService()
	result, err := service.Inspect(InspectRequest{OutputDir: dir})
	require.NoError(t, err)
	if diff := cmp.Diff(1, result.AptLockCount); diff != "" {
		t.Fatalf("unexpected apt lock count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, len(result.Groups)); diff != "" {
		t.Fatalf("unexpected group count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("group", result.Groups[0].Name); diff != "" {
		t.Fatalf("unexpected group name (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, len(result.ResolutionRecords)); diff != "" {
		t.Fatalf("unexpected resolution record count (-want +got):\n%s", diff)
	}
}
