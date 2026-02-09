package adapters

import (
	"avular-packages/internal/ports"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestRepoSnapshotFileAdapterPublish(t *testing.T) {
	tests := []struct {
		name       string
		snapshotID string
		setup      func(ports.RepoSnapshotPort, context.Context) error
		wantErr    bool
		wantCode   errbuilder.ErrCode
		wantFile   bool
	}{
		{
			name:       "first publish",
			snapshotID: "snap-1",
			wantErr:    false,
			wantFile:   true,
		},
		{
			name:       "duplicate publish",
			snapshotID: "snap-1",
			setup: func(adapter ports.RepoSnapshotPort, ctx context.Context) error {
				return adapter.Publish(ctx, "snap-1")
			},
			wantErr:  true,
			wantCode: errbuilder.CodeAlreadyExists,
			wantFile: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			adapter := NewRepoSnapshotFileAdapter(dir)
			ctx := t.Context()

			if tt.setup != nil {
				require.NoError(t, tt.setup(&adapter, ctx))
			}

			err := adapter.Publish(ctx, tt.snapshotID)
			if tt.wantErr {
				require.Error(t, err)
				if diff := cmp.Diff(tt.wantCode, errbuilder.CodeOf(err)); diff != "" {
					t.Fatalf("unexpected error code (-want +got):\n%s", diff)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.wantFile {
				path := filepath.Join(dir, "snapshots", tt.snapshotID+".snapshot")
				_, statErr := os.Stat(path)
				require.NoError(t, statErr)
			}
		})
	}
}

func TestRepoSnapshotFileAdapterPromote(t *testing.T) {
	tests := []struct {
		name       string
		snapshotID string
		channel    string
	}{
		{name: "dev channel", snapshotID: "snap-1", channel: "dev"},
		{name: "staging channel", snapshotID: "snap-2", channel: "staging"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			adapter := NewRepoSnapshotFileAdapter(dir)
			ctx := t.Context()

			require.NoError(t, adapter.Promote(ctx, tt.snapshotID, tt.channel))
			path := filepath.Join(dir, "channels", tt.channel)
			content, err := os.ReadFile(path)
			require.NoError(t, err)
			if diff := cmp.Diff(true, strings.Contains(string(content), tt.snapshotID)); diff != "" {
				t.Fatalf("unexpected channel content (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepoSnapshotFileAdapterListSnapshots(t *testing.T) {
	dir := t.TempDir()
	adapter := NewRepoSnapshotFileAdapter(dir)
	ctx := t.Context()

	require.NoError(t, adapter.Publish(ctx, "snap-1"))
	require.NoError(t, adapter.Publish(ctx, "snap-2"))
	require.NoError(t, adapter.Promote(ctx, "snap-2", "staging"))

	snapshots, err := adapter.ListSnapshots(ctx)
	require.NoError(t, err)
	require.Len(t, snapshots, 2)

	var hasChannel bool
	for _, snapshot := range snapshots {
		if snapshot.SnapshotID == "snap-2" && snapshot.Channel == "staging" {
			hasChannel = true
		}
	}
	require.True(t, hasChannel)
}

func TestRepoSnapshotFileAdapterDeleteSnapshot(t *testing.T) {
	dir := t.TempDir()
	adapter := NewRepoSnapshotFileAdapter(dir)
	ctx := t.Context()

	require.NoError(t, adapter.Publish(ctx, "snap-1"))
	require.NoError(t, adapter.DeleteSnapshot(ctx, "snap-1"))

	_, err := os.Stat(filepath.Join(dir, "snapshots", "snap-1.snapshot"))
	require.Error(t, err)
}
