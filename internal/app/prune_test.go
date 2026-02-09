package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"avular-packages/internal/adapters"
)

func TestPruneSnapshotsFileBackend(t *testing.T) {
	dir := t.TempDir()
	adapter := adapters.NewRepoSnapshotFileAdapter(dir)
	ctx := t.Context()

	require.NoError(t, adapter.Publish(ctx, "snap-1"))
	require.NoError(t, adapter.Publish(ctx, "snap-2"))

	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "snapshots", "snap-1.snapshot"), oldTime, oldTime))
	require.NoError(t, os.Chtimes(filepath.Join(dir, "snapshots", "snap-2.snapshot"), newTime, newTime))

	service := NewService()
	result, err := service.PruneSnapshots(ctx, PruneRequest{
		RepoBackend: "file",
		RepoDir:     dir,
		KeepLast:    1,
		DryRun:      false,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.DeleteCount)

	_, err = os.Stat(filepath.Join(dir, "snapshots", "snap-1.snapshot"))
	require.Error(t, err)
	_, err = os.Stat(filepath.Join(dir, "snapshots", "snap-2.snapshot"))
	require.NoError(t, err)
}
