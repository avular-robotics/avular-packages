package app

import (
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestBuildPrunePlanKeepLastPerPrefix(t *testing.T) {
	now := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)
	snapshots := []types.SnapshotInfo{
		{SnapshotID: "alpha-aaa", Prefix: "alpha", CreatedAt: now.Add(-2 * time.Hour)},
		{SnapshotID: "alpha-bbb", Prefix: "alpha", CreatedAt: now.Add(-1 * time.Hour)},
		{SnapshotID: "beta-ccc", Prefix: "beta", CreatedAt: now.Add(-3 * time.Hour)},
		{SnapshotID: "beta-ddd", Prefix: "beta", CreatedAt: now.Add(-30 * time.Minute)},
	}
	policy := types.SnapshotRetentionPolicy{KeepLast: 1}

	plan := BuildPrunePlan(snapshots, policy, now)
	kept := snapshotIDs(plan.Keep)
	deleted := snapshotIDs(plan.Delete)

	require.ElementsMatch(t, []string{"alpha-bbb", "beta-ddd"}, kept)
	require.ElementsMatch(t, []string{"alpha-aaa", "beta-ccc"}, deleted)
}

func TestBuildPrunePlanKeepDays(t *testing.T) {
	now := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)
	snapshots := []types.SnapshotInfo{
		{SnapshotID: "pfx-recent", Prefix: "pfx", CreatedAt: now.AddDate(0, 0, -1)},
		{SnapshotID: "pfx-old", Prefix: "pfx", CreatedAt: now.AddDate(0, 0, -10)},
	}
	policy := types.SnapshotRetentionPolicy{KeepDays: 3}

	plan := BuildPrunePlan(snapshots, policy, now)
	kept := snapshotIDs(plan.Keep)
	deleted := snapshotIDs(plan.Delete)

	require.ElementsMatch(t, []string{"pfx-recent"}, kept)
	require.ElementsMatch(t, []string{"pfx-old"}, deleted)
}

func TestBuildPrunePlanProtectChannelsAndPrefixes(t *testing.T) {
	now := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)
	snapshots := []types.SnapshotInfo{
		{SnapshotID: "dev-111", Channel: "dev", Prefix: "dev", CreatedAt: now.AddDate(0, 0, -30)},
		{SnapshotID: "core-222", Prefix: "core", CreatedAt: now.AddDate(0, 0, -30)},
		{SnapshotID: "misc-333", Prefix: "misc", CreatedAt: now.AddDate(0, 0, -30)},
	}
	policy := types.SnapshotRetentionPolicy{
		KeepLast:        0,
		KeepDays:        0,
		ProtectChannels: []string{"dev"},
		ProtectPrefixes: []string{"core"},
	}

	plan := BuildPrunePlan(snapshots, policy, now)
	kept := snapshotIDs(plan.Keep)
	deleted := snapshotIDs(plan.Delete)

	require.ElementsMatch(t, []string{"dev-111", "core-222"}, kept)
	require.ElementsMatch(t, []string{"misc-333"}, deleted)
}

func TestBuildPrunePlanDeterministicOrdering(t *testing.T) {
	now := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)
	snapshots := []types.SnapshotInfo{
		{SnapshotID: "alpha-ccc", Prefix: "alpha", CreatedAt: now.Add(-1 * time.Hour)},
		{SnapshotID: "alpha-bbb", Prefix: "alpha", CreatedAt: now.Add(-1 * time.Hour)},
		{SnapshotID: "alpha-aaa", Prefix: "alpha", CreatedAt: now.Add(-1 * time.Hour)},
	}
	policy := types.SnapshotRetentionPolicy{KeepLast: 1}

	plan := BuildPrunePlan(snapshots, policy, now)
	kept := snapshotIDs(plan.Keep)
	sort.Strings(kept)
	if diff := cmp.Diff([]string{"alpha-aaa"}, kept); diff != "" {
		t.Fatalf("unexpected kept snapshots (-want +got):\n%s", diff)
	}
}

func snapshotIDs(items []types.SnapshotInfo) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.SnapshotID)
	}
	return ids
}
