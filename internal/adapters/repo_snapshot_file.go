package adapters

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
)

type RepoSnapshotFileAdapter struct {
	Dir string
}

func NewRepoSnapshotFileAdapter(dir string) RepoSnapshotFileAdapter {
	return RepoSnapshotFileAdapter{Dir: dir}
}

func (a RepoSnapshotFileAdapter) Publish(ctx context.Context, snapshotID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(a.Dir) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("repo snapshot directory is empty")
	}
	if strings.TrimSpace(snapshotID) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("snapshot id is empty")
	}
	if strings.Contains(snapshotID, string(os.PathSeparator)) {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("snapshot id contains path separator")
	}
	snapshotsDir := filepath.Join(a.Dir, "snapshots")
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create snapshots directory").
			WithCause(err)
	}
	path := filepath.Join(snapshotsDir, snapshotID+".snapshot")
	if _, err := os.Stat(path); err == nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeAlreadyExists).
			WithMsg("snapshot already exists")
	}
	if err := os.WriteFile(path, []byte(snapshotID+"\n"), 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write snapshot metadata").
			WithCause(err)
	}
	return nil
}

func (a RepoSnapshotFileAdapter) Promote(ctx context.Context, snapshotID string, channel string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(a.Dir) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("repo snapshot directory is empty")
	}
	if strings.TrimSpace(snapshotID) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("snapshot id is empty")
	}
	if strings.TrimSpace(channel) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("channel is empty")
	}
	if strings.Contains(snapshotID, string(os.PathSeparator)) {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("snapshot id contains path separator")
	}
	channelsDir := filepath.Join(a.Dir, "channels")
	if err := os.MkdirAll(channelsDir, 0755); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create channels directory").
			WithCause(err)
	}
	path := filepath.Join(channelsDir, channel)
	if err := os.WriteFile(path, []byte(snapshotID+"\n"), 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write channel pointer").
			WithCause(err)
	}
	return nil
}

var _ ports.RepoSnapshotPort = RepoSnapshotFileAdapter{}
