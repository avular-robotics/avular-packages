package app

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/types"
)

func (s Service) Inspect(req InspectRequest) (InspectResult, error) {
	outputDir := strings.TrimSpace(req.OutputDir)
	if outputDir == "" {
		return InspectResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("output directory is required")
	}
	aptLocks, err := s.OutputReader.ReadAptLock(filepath.Join(outputDir, "apt.lock"))
	if err != nil {
		return InspectResult{}, err
	}
	manifest, err := s.OutputReader.ReadBundleManifest(filepath.Join(outputDir, "bundle.manifest"))
	if err != nil {
		return InspectResult{}, err
	}
	report, err := s.OutputReader.ReadResolutionReport(filepath.Join(outputDir, "resolution.report"))
	if err != nil {
		return InspectResult{}, err
	}

	groupStats := summarizeManifest(manifest)
	groupPackages := summarizePackages(manifest)
	var summaries []InspectGroupSummary
	for _, name := range sortedKeys(groupStats) {
		stat := groupStats[name]
		packages := groupPackages[name]
		sort.Strings(packages)
		summaries = append(summaries, InspectGroupSummary{
			Name:     name,
			Mode:     stat.Mode,
			Count:    stat.Count,
			Packages: packages,
		})
	}
	return InspectResult{
		AptLockCount:      len(aptLocks),
		Groups:            summaries,
		ResolutionRecords: report.Records,
	}, nil
}

type groupSummary struct {
	Mode  types.PackagingMode
	Count int
}

func summarizeManifest(entries []types.BundleManifestEntry) map[string]groupSummary {
	stats := map[string]groupSummary{}
	for _, entry := range entries {
		stat := stats[entry.Group]
		if stat.Mode == "" {
			stat.Mode = entry.Mode
		}
		stat.Count++
		stats[entry.Group] = stat
	}
	return stats
}

func summarizePackages(entries []types.BundleManifestEntry) map[string][]string {
	packages := map[string][]string{}
	for _, entry := range entries {
		packages[entry.Group] = append(packages[entry.Group], entry.Package)
	}
	return packages
}

func sortedKeys[K comparable, V any](input map[K]V) []K {
	keys := make([]K, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])
	})
	return keys
}
