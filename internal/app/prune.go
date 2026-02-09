package app

import (
	"context"
	"strings"
	"time"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/adapters"
	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

func (s Service) PruneSnapshots(ctx context.Context, req PruneRequest) (PruneResult, error) {
	backend := strings.ToLower(strings.TrimSpace(req.RepoBackend))
	if backend == "" {
		backend = "file"
	}
	adapter, err := buildPruneAdapter(backend, req)
	if err != nil {
		return PruneResult{}, err
	}
	snapshots, err := adapter.ListSnapshots(ctx)
	if err != nil {
		return PruneResult{}, err
	}
	snapshots = annotateChannels(snapshots, req.ProtectChannels)
	policy := types.SnapshotRetentionPolicy{
		KeepLast:        req.KeepLast,
		KeepDays:        req.KeepDays,
		ProtectChannels: req.ProtectChannels,
		ProtectPrefixes: req.ProtectPrefixes,
		DryRun:          req.DryRun,
	}
	now := timeNow(s.Clock)
	plan := BuildPrunePlan(snapshots, policy, now)
	if policy.DryRun {
		return PruneResult{
			KeepCount:   len(plan.Keep),
			DeleteCount: len(plan.Delete),
			DryRun:      true,
		}, nil
	}
	var deleted []string
	for _, snapshot := range plan.Delete {
		if err := adapter.DeleteSnapshot(ctx, snapshot.SnapshotID); err != nil {
			return PruneResult{}, err
		}
		deleted = append(deleted, snapshot.SnapshotID)
	}
	return PruneResult{
		KeepCount:   len(plan.Keep),
		DeleteCount: len(deleted),
		Deleted:     deleted,
		DryRun:      false,
	}, nil
}

func timeNow(clock func() time.Time) time.Time {
	if clock == nil {
		return time.Now().UTC()
	}
	return clock().UTC()
}

func buildPruneAdapter(backend string, req PruneRequest) (ports.RepoSnapshotPort, error) {
	switch backend {
	case "file":
		repoDir := strings.TrimSpace(req.RepoDir)
		if repoDir == "" {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("repo dir is required for file backend")
		}
		adapter := adapters.NewRepoSnapshotFileAdapter(repoDir)
		return adapter, nil
	case "aptly":
		adapter := adapters.NewRepoSnapshotAptlyAdapter("", "", "", "", "", "", "")
		return adapter, nil
	case "proget":
		endpoint := strings.TrimSpace(req.ProGetEndpoint)
		feed := strings.TrimSpace(req.ProGetFeed)
		apiKey := strings.TrimSpace(req.ProGetAPIKey)
		if endpoint == "" || feed == "" {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("proget endpoint and feed are required")
		}
		if apiKey == "" {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("proget api key is required")
		}
		component := strings.TrimSpace(req.ProGetComponent)
		user := strings.TrimSpace(req.ProGetUser)
		adapter := adapters.NewRepoSnapshotProGetAdapter(
			endpoint,
			feed,
			component,
			"",
			user,
			apiKey,
			"",
			0,
			req.ProGetTimeoutSec,
			req.ProGetRetries,
			req.ProGetRetryDelayMs,
		)
		return adapter, nil
	default:
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("unsupported repo backend")
	}
}

func annotateChannels(snapshots []types.SnapshotInfo, channels []string) []types.SnapshotInfo {
	if len(channels) == 0 {
		return snapshots
	}
	channelSet := map[string]struct{}{}
	for _, channel := range channels {
		value := strings.ToLower(strings.TrimSpace(channel))
		if value == "" {
			continue
		}
		channelSet[value] = struct{}{}
	}
	if len(channelSet) == 0 {
		return snapshots
	}
	for i := range snapshots {
		name := strings.ToLower(strings.TrimSpace(snapshots[i].SnapshotID))
		if _, ok := channelSet[name]; ok {
			snapshots[i].Channel = snapshots[i].SnapshotID
		}
	}
	return snapshots
}
