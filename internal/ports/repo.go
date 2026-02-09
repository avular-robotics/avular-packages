package ports

import (
	"context"

	"avular-packages/internal/types"
)

type RepoIndexPort interface {
	AvailableVersions(depType types.DependencyType, name string) ([]string, error)
	AptPackages() (map[string][]types.AptPackageVersion, error)
}

type RepoSnapshotPort interface {
	Publish(ctx context.Context, snapshotID string) error
	Promote(ctx context.Context, snapshotID string, channel string) error
	ListSnapshots(ctx context.Context) ([]types.SnapshotInfo, error)
	DeleteSnapshot(ctx context.Context, snapshotID string) error
}
