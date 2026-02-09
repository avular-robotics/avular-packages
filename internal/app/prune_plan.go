package app

import (
	"sort"
	"strings"
	"time"

	"avular-packages/internal/types"
)

func BuildPrunePlan(snapshots []types.SnapshotInfo, policy types.SnapshotRetentionPolicy, now time.Time) types.SnapshotPrunePlan {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	normalized := normalizeRetentionPolicy(policy)
	protectedChannels := normalizeSet(normalized.ProtectChannels)
	protectedPrefixes := normalizeSet(normalized.ProtectPrefixes)

	keepIDs := map[string]struct{}{}
	grouped := map[string][]types.SnapshotInfo{}
	for _, snapshot := range snapshots {
		current := snapshot
		if strings.TrimSpace(current.Prefix) == "" {
			current.Prefix = inferSnapshotPrefix(current.SnapshotID)
		}
		if isProtected(current, protectedChannels, protectedPrefixes) {
			keepIDs[current.SnapshotID] = struct{}{}
		}
		if normalized.KeepDays > 0 && !current.CreatedAt.IsZero() {
			cutoff := now.AddDate(0, 0, -normalized.KeepDays)
			if !current.CreatedAt.Before(cutoff) {
				keepIDs[current.SnapshotID] = struct{}{}
			}
		}
		group := retentionGroupKey(current)
		grouped[group] = append(grouped[group], current)
	}

	if normalized.KeepLast > 0 {
		for _, group := range grouped {
			sorted := append([]types.SnapshotInfo(nil), group...)
			sort.Slice(sorted, func(i, j int) bool {
				if !sorted[i].CreatedAt.Equal(sorted[j].CreatedAt) {
					return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
				}
				return sorted[i].SnapshotID < sorted[j].SnapshotID
			})
			limit := normalized.KeepLast
			if limit > len(sorted) {
				limit = len(sorted)
			}
			for i := 0; i < limit; i++ {
				keepIDs[sorted[i].SnapshotID] = struct{}{}
			}
		}
	}

	var keep []types.SnapshotInfo
	var del []types.SnapshotInfo
	for _, snapshot := range snapshots {
		if _, ok := keepIDs[snapshot.SnapshotID]; ok {
			keep = append(keep, snapshot)
		} else {
			del = append(del, snapshot)
		}
	}
	return types.SnapshotPrunePlan{Keep: keep, Delete: del}
}

func normalizeRetentionPolicy(policy types.SnapshotRetentionPolicy) types.SnapshotRetentionPolicy {
	normalized := policy
	if normalized.KeepLast < 0 {
		normalized.KeepLast = 0
	}
	if normalized.KeepDays < 0 {
		normalized.KeepDays = 0
	}
	return normalized
}

func normalizeSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return set
}

func isProtected(snapshot types.SnapshotInfo, channels map[string]struct{}, prefixes map[string]struct{}) bool {
	if snapshot.Channel != "" {
		if _, ok := channels[strings.ToLower(snapshot.Channel)]; ok {
			return true
		}
	}
	if snapshot.Prefix != "" {
		if _, ok := prefixes[strings.ToLower(snapshot.Prefix)]; ok {
			return true
		}
	}
	return false
}

func inferSnapshotPrefix(snapshotID string) string {
	trimmed := strings.TrimSpace(snapshotID)
	if trimmed == "" {
		return ""
	}
	parts := strings.SplitN(trimmed, "-", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func retentionGroupKey(snapshot types.SnapshotInfo) string {
	if strings.TrimSpace(snapshot.Prefix) != "" {
		return "prefix:" + strings.ToLower(snapshot.Prefix)
	}
	if strings.TrimSpace(snapshot.Channel) != "" {
		return "channel:" + strings.ToLower(snapshot.Channel)
	}
	if strings.TrimSpace(snapshot.Repository) != "" {
		return "repo:" + strings.ToLower(snapshot.Repository)
	}
	return "default"
}
