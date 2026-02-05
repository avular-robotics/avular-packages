package ports

import "avular-packages/internal/types"

type OutputPort interface {
	WriteAptLock(entries []types.AptLockEntry) error
	WriteBundleManifest(entries []types.BundleManifestEntry) error
	WriteSnapshotIntent(intent types.SnapshotIntent) error
	WriteResolutionReport(report types.ResolutionReport) error
}
