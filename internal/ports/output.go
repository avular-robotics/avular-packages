package ports

import "avular-packages/internal/types"

type OutputPort interface {
	WriteAptLock(entries []types.AptLockEntry) error
	WriteAptPreferences(entries []types.AptLockEntry) error
	WriteAptInstallList(entries []types.AptLockEntry) error
	WriteBundleManifest(entries []types.BundleManifestEntry) error
	WriteSnapshotIntent(intent types.SnapshotIntent) error
	WriteSnapshotSources(intent types.SnapshotIntent, baseURL string, component string, archs []string) error
	WriteResolutionReport(report types.ResolutionReport) error
}
