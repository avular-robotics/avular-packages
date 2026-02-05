package adapters

import (
	"os"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

type OutputReaderAdapter struct{}

func NewOutputReaderAdapter() OutputReaderAdapter {
	return OutputReaderAdapter{}
}

func (a OutputReaderAdapter) ReadSnapshotIntent(path string) (types.SnapshotIntent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return types.SnapshotIntent{}, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("snapshot.intent not found").
			WithCause(err)
	}
	intent := types.SnapshotIntent{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return types.SnapshotIntent{}, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("invalid snapshot.intent format")
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "repository":
			intent.Repository = value
		case "channel":
			intent.Channel = value
		case "snapshot_prefix":
			intent.SnapshotPrefix = value
		case "snapshot_id":
			intent.SnapshotID = value
		case "created_at":
			intent.CreatedAt = value
		case "signing_key":
			intent.SigningKey = value
		}
	}
	if strings.TrimSpace(intent.SnapshotID) == "" {
		return types.SnapshotIntent{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("snapshot.intent missing snapshot_id")
	}
	return intent, nil
}

func (a OutputReaderAdapter) ReadAptLock(path string) ([]types.AptLockEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("apt.lock not found").
			WithCause(err)
	}
	var entries []types.AptLockEntry
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("invalid apt.lock format")
		}
		entries = append(entries, types.AptLockEntry{
			Package: strings.TrimSpace(parts[0]),
			Version: strings.TrimSpace(parts[1]),
		})
	}
	return entries, nil
}

func (a OutputReaderAdapter) ReadBundleManifest(path string) ([]types.BundleManifestEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("bundle.manifest not found").
			WithCause(err)
	}
	var entries []types.BundleManifestEntry
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) != 4 {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("invalid bundle.manifest format")
		}
		entry := types.BundleManifestEntry{
			Group:   strings.TrimSpace(parts[0]),
			Mode:    types.PackagingMode(strings.TrimSpace(parts[1])),
			Package: strings.TrimSpace(parts[2]),
			Version: strings.TrimSpace(parts[3]),
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (a OutputReaderAdapter) ReadResolutionReport(path string) (types.ResolutionReport, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return types.ResolutionReport{}, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("resolution.report not found").
			WithCause(err)
	}
	var records []types.ResolutionRecord
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			return types.ResolutionReport{}, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("invalid resolution.report format")
		}
		record := types.ResolutionRecord{
			Dependency: strings.TrimSpace(parts[0]),
			Action:     strings.TrimSpace(parts[1]),
			Value:      strings.TrimSpace(parts[2]),
			Reason:     strings.TrimSpace(parts[3]),
			Owner:      strings.TrimSpace(parts[4]),
		}
		if len(parts) > 5 {
			record.ExpiresAt = strings.TrimSpace(parts[5])
		}
		records = append(records, record)
	}
	return types.ResolutionReport{Records: records}, nil
}

var _ ports.OutputReaderPort = OutputReaderAdapter{}
