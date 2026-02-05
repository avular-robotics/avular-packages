package ports

import "avular-packages/internal/types"

type OutputReaderPort interface {
	ReadAptLock(path string) ([]types.AptLockEntry, error)
	ReadBundleManifest(path string) ([]types.BundleManifestEntry, error)
	ReadResolutionReport(path string) (types.ResolutionReport, error)
	ReadSnapshotIntent(path string) (types.SnapshotIntent, error)
}
