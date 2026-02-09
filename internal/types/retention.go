package types

import "time"

type SnapshotInfo struct {
	Repository string
	SnapshotID string
	Channel    string
	Prefix     string
	CreatedAt  time.Time
}

type SnapshotRetentionPolicy struct {
	KeepLast        int
	KeepDays        int
	ProtectChannels []string
	ProtectPrefixes []string
	DryRun          bool
}

type SnapshotPrunePlan struct {
	Keep   []SnapshotInfo
	Delete []SnapshotInfo
}
