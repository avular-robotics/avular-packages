package types

type AptLockEntry struct {
	Package string
	Version string
}

type BundleManifestEntry struct {
	Group   string
	Mode    PackagingMode
	Package string
	Version string
}

type ResolvedDependency struct {
	Type    DependencyType
	Package string
	Version string
}

type SnapshotIntent struct {
	Repository     string
	Channel        string
	SnapshotPrefix string
	SnapshotID     string
	CreatedAt      string
	SigningKey     string
}

type ResolutionRecord struct {
	Dependency string
	Action     string
	Value      string
	Reason     string
	Owner      string
	ExpiresAt  string
}

type ResolutionReport struct {
	Records []ResolutionRecord
}
