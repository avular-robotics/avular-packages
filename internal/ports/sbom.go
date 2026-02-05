package ports

import "avular-packages/internal/types"

type SBOMPort interface {
	WriteSBOM(repoDir string, snapshotID string, createdAt string, locks []types.AptLockEntry) error
}
