package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

type OutputFileAdapter struct {
	Dir string
}

func NewOutputFileAdapter(dir string) OutputFileAdapter {
	return OutputFileAdapter{Dir: dir}
}

func (a OutputFileAdapter) WriteAptLock(entries []types.AptLockEntry) error {
	path, err := a.ensurePath("apt.lock")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Package < entries[j].Package
	})
	var lines []string
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s=%s", entry.Package, entry.Version))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func (a OutputFileAdapter) WriteBundleManifest(entries []types.BundleManifestEntry) error {
	path, err := a.ensurePath("bundle.manifest")
	if err != nil {
		return err
	}
	ordered := append([]types.BundleManifestEntry(nil), entries...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Group != ordered[j].Group {
			return ordered[i].Group < ordered[j].Group
		}
		if ordered[i].Package != ordered[j].Package {
			return ordered[i].Package < ordered[j].Package
		}
		if ordered[i].Version != ordered[j].Version {
			return ordered[i].Version < ordered[j].Version
		}
		return ordered[i].Mode < ordered[j].Mode
	})
	var lines []string
	for _, entry := range ordered {
		lines = append(lines, fmt.Sprintf("%s,%s,%s,%s", entry.Group, entry.Mode, entry.Package, entry.Version))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func (a OutputFileAdapter) WriteSnapshotIntent(intent types.SnapshotIntent) error {
	path, err := a.ensurePath("snapshot.intent")
	if err != nil {
		return err
	}
	content := fmt.Sprintf(
		"repository=%s\nchannel=%s\nsnapshot_prefix=%s\nsnapshot_id=%s\ncreated_at=%s\nsigning_key=%s\n",
		intent.Repository,
		intent.Channel,
		intent.SnapshotPrefix,
		intent.SnapshotID,
		intent.CreatedAt,
		intent.SigningKey,
	)
	return os.WriteFile(path, []byte(content), 0644)
}

func (a OutputFileAdapter) WriteResolutionReport(report types.ResolutionReport) error {
	path, err := a.ensurePath("resolution.report")
	if err != nil {
		return err
	}
	ordered := append([]types.ResolutionRecord(nil), report.Records...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Dependency != ordered[j].Dependency {
			return ordered[i].Dependency < ordered[j].Dependency
		}
		if ordered[i].Action != ordered[j].Action {
			return ordered[i].Action < ordered[j].Action
		}
		if ordered[i].Value != ordered[j].Value {
			return ordered[i].Value < ordered[j].Value
		}
		if ordered[i].Owner != ordered[j].Owner {
			return ordered[i].Owner < ordered[j].Owner
		}
		return ordered[i].Reason < ordered[j].Reason
	})
	var lines []string
	for _, record := range ordered {
		lines = append(lines, fmt.Sprintf(
			"%s,%s,%s,%s,%s,%s",
			record.Dependency,
			record.Action,
			record.Value,
			record.Reason,
			record.Owner,
			record.ExpiresAt,
		))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func (a OutputFileAdapter) ensurePath(filename string) (string, error) {
	if a.Dir == "" {
		return "", errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("output directory is empty")
	}
	if err := os.MkdirAll(a.Dir, 0755); err != nil {
		return "", errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create output directory").
			WithCause(err)
	}
	return filepath.Join(a.Dir, filename), nil
}

var _ ports.OutputPort = OutputFileAdapter{}
