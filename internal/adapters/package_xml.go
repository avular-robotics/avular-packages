package adapters

import (
	"encoding/xml"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

type PackageXMLAdapter struct {
	mu    sync.Mutex
	cache map[string]packageXMLCacheEntry
}

func NewPackageXMLAdapter() *PackageXMLAdapter {
	return &PackageXMLAdapter{cache: map[string]packageXMLCacheEntry{}}
}

type packageXML struct {
	Name   string        `xml:"name"`
	Export exportSection `xml:"export"`

	// Standard ROS dependency tags (REP-149 / REP-140)
	Depend         []simpleDepend `xml:"depend"`
	ExecDepend     []simpleDepend `xml:"exec_depend"`
	BuildDepend    []simpleDepend `xml:"build_depend"`
	BuildExportDep []simpleDepend `xml:"build_export_depend"`
	RunDepend      []simpleDepend `xml:"run_depend"`
	TestDepend     []simpleDepend `xml:"test_depend"`
}

type exportSection struct {
	DebianDepends []simpleDepend `xml:"debian_depend"`
	PipDepends    []pipDepend    `xml:"pip_depend"`
}

type simpleDepend struct {
	Value string `xml:",chardata"`
}

type pipDepend struct {
	Value   string `xml:",chardata"`
	Version string `xml:"version,attr"`
}

type packageXMLCacheEntry struct {
	modTime    time.Time
	debianDeps []string
	pipDeps    []string
	rosTagDeps []types.ROSTagDependency
	name       string
}

func (a *PackageXMLAdapter) ParseDependencies(paths []string, tags []string) ([]string, []string, error) {
	wantDeb := hasTag(tags, "debian_depend")
	wantPip := hasTag(tags, "pip_depend")
	if !wantDeb && !wantPip {
		return nil, nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("no supported package.xml tags provided")
	}

	var debs []string
	var pips []string

	for _, path := range paths {
		entry, err := a.loadPackageXML(path)
		if err != nil {
			return nil, nil, err
		}
		if wantDeb {
			debs = append(debs, entry.debianDeps...)
		}
		if wantPip {
			pips = append(pips, entry.pipDeps...)
		}
	}

	return debs, pips, nil
}

func (a *PackageXMLAdapter) ParseROSTags(paths []string) ([]types.ROSTagDependency, error) {
	var result []types.ROSTagDependency
	for _, path := range paths {
		entry, err := a.loadPackageXML(path)
		if err != nil {
			return nil, err
		}
		result = append(result, entry.rosTagDeps...)
	}
	return result, nil
}

func (a *PackageXMLAdapter) ParsePackageNames(paths []string) ([]string, error) {
	var names []string
	for _, path := range paths {
		entry, err := a.loadPackageXML(path)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(entry.name) != "" {
			names = append(names, strings.TrimSpace(entry.name))
		}
	}
	return names, nil
}

func (a *PackageXMLAdapter) loadPackageXML(path string) (packageXMLCacheEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return packageXMLCacheEntry{}, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("failed to read package.xml").
			WithCause(err)
	}
	a.mu.Lock()
	if entry, ok := a.cache[path]; ok && entry.modTime.Equal(info.ModTime()) {
		a.mu.Unlock()
		return entry, nil
	}
	a.mu.Unlock()

	content, err := os.ReadFile(path)
	if err != nil {
		return packageXMLCacheEntry{}, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("failed to read package.xml").
			WithCause(err)
	}
	var pkg packageXML
	if err := xml.Unmarshal(content, &pkg); err != nil {
		return packageXMLCacheEntry{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("failed to parse package.xml").
			WithCause(err)
	}
	entry := packageXMLCacheEntry{
		modTime: info.ModTime(),
		name:    strings.TrimSpace(pkg.Name),
	}
	for _, dep := range pkg.Export.DebianDepends {
		value := strings.TrimSpace(dep.Value)
		if value != "" {
			entry.debianDeps = append(entry.debianDeps, value)
		}
	}
	for _, dep := range pkg.Export.PipDepends {
		value := strings.TrimSpace(dep.Value)
		if value == "" {
			continue
		}
		if dep.Version != "" {
			entry.pipDeps = append(entry.pipDeps, value+"=="+dep.Version)
			continue
		}
		entry.pipDeps = append(entry.pipDeps, value)
	}

	// Extract standard ROS dependency tags as abstract keys
	entry.rosTagDeps = collectROSTags(&pkg)

	a.mu.Lock()
	a.cache[path] = entry
	a.mu.Unlock()
	return entry, nil
}

// collectROSTags extracts all standard ROS dependency tags from the
// parsed package.xml and returns them as ROSTagDependency entries.
func collectROSTags(pkg *packageXML) []types.ROSTagDependency {
	var deps []types.ROSTagDependency

	for _, dep := range pkg.Depend {
		if key := strings.TrimSpace(dep.Value); key != "" {
			deps = append(deps, types.ROSTagDependency{Key: key, Scope: types.ROSDepScopeAll})
		}
	}
	for _, dep := range pkg.ExecDepend {
		if key := strings.TrimSpace(dep.Value); key != "" {
			deps = append(deps, types.ROSTagDependency{Key: key, Scope: types.ROSDepScopeExec})
		}
	}
	for _, dep := range pkg.BuildDepend {
		if key := strings.TrimSpace(dep.Value); key != "" {
			deps = append(deps, types.ROSTagDependency{Key: key, Scope: types.ROSDepScopeBuild})
		}
	}
	for _, dep := range pkg.BuildExportDep {
		if key := strings.TrimSpace(dep.Value); key != "" {
			deps = append(deps, types.ROSTagDependency{Key: key, Scope: types.ROSDepScopeBuildExec})
		}
	}
	for _, dep := range pkg.RunDepend {
		if key := strings.TrimSpace(dep.Value); key != "" {
			deps = append(deps, types.ROSTagDependency{Key: key, Scope: types.ROSDepScopeExec})
		}
	}
	for _, dep := range pkg.TestDepend {
		if key := strings.TrimSpace(dep.Value); key != "" {
			deps = append(deps, types.ROSTagDependency{Key: key, Scope: types.ROSDepScopeTest})
		}
	}

	return deps
}

func hasTag(tags []string, name string) bool {
	for _, tag := range tags {
		if tag == name {
			return true
		}
	}
	return false
}

var _ ports.PackageXMLPort = (*PackageXMLAdapter)(nil)
