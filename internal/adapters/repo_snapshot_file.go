package adapters

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
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

func (a RepoSnapshotFileAdapter) ListSnapshots(ctx context.Context) ([]types.SnapshotInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Dir) == "" {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("repo snapshot directory is empty")
	}
	snapshotsDir := filepath.Join(a.Dir, "snapshots")
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.SnapshotInfo{}, nil
		}
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to read snapshots directory").
			WithCause(err)
	}
	var snapshots []types.SnapshotInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".snapshot") {
			continue
		}
		path := filepath.Join(snapshotsDir, name)
		info, err := entry.Info()
		if err != nil {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("failed to read snapshot info").
				WithCause(err)
		}
		snapshotID := strings.TrimSuffix(name, ".snapshot")
		content, err := os.ReadFile(path)
		if err == nil {
			line := strings.TrimSpace(string(content))
			if line != "" {
				snapshotID = line
			}
		}
		snapshots = append(snapshots, types.SnapshotInfo{
			SnapshotID: snapshotID,
			CreatedAt:  info.ModTime().UTC(),
		})
	}
	if err := applyChannelMappings(a.Dir, snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (a RepoSnapshotFileAdapter) DeleteSnapshot(ctx context.Context, snapshotID string) error {
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
	path := filepath.Join(a.Dir, "snapshots", snapshotID+".snapshot")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return errbuilder.New().
				WithCode(errbuilder.CodeNotFound).
				WithMsg("snapshot not found")
		}
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to delete snapshot").
			WithCause(err)
	}
	return nil
}

func applyChannelMappings(root string, snapshots []types.SnapshotInfo) error {
	channelsDir := filepath.Join(root, "channels")
	entries, err := os.ReadDir(channelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to read channels directory").
			WithCause(err)
	}
	mapping := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(channelsDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("failed to read channel pointer").
				WithCause(err)
		}
		snapshotID := strings.TrimSpace(string(content))
		if snapshotID == "" {
			continue
		}
		mapping[snapshotID] = entry.Name()
	}
	for i := range snapshots {
		if channel, ok := mapping[snapshots[i].SnapshotID]; ok {
			snapshots[i].Channel = channel
		}
	}
	return nil
}

var _ ports.RepoSnapshotPort = RepoSnapshotFileAdapter{}
