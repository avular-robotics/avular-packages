package ports

import (
	"context"

	"avular-packages/internal/types"
)

type RepoIndexPort interface {
	AvailableVersions(depType types.DependencyType, name string) ([]string, error)
}

type RepoSnapshotPort interface {
	Publish(ctx context.Context, snapshotID string) error
	Promote(ctx context.Context, snapshotID string, channel string) error
}
