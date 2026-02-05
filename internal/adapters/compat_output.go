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

type CompatibilityOutputAdapter struct {
	Dir string
}

func NewCompatibilityOutputAdapter(dir string) CompatibilityOutputAdapter {
	return CompatibilityOutputAdapter{Dir: dir}
}

func (a CompatibilityOutputAdapter) WriteGetDependencies(resolved []types.ResolvedDependency) error {
	var aptLines []string
	var pipLines []string
	for _, dep := range resolved {
		switch dep.Type {
		case types.DependencyTypeApt:
			aptLines = append(aptLines, fmt.Sprintf("%s=%s", dep.Package, dep.Version))
		case types.DependencyTypePip:
			pipLines = append(pipLines, fmt.Sprintf("%s==%s", dep.Package, dep.Version))
		}
	}
	sort.Strings(aptLines)
	sort.Strings(pipLines)

	if err := os.MkdirAll(a.Dir, 0755); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create output directory").
			WithCause(err)
	}
	aptPath := filepath.Join(a.Dir, "get-dependencies.apt")
	if err := os.WriteFile(aptPath, []byte(strings.Join(aptLines, "\n")), 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write get-dependencies apt output").
			WithCause(err)
	}
	pipPath := filepath.Join(a.Dir, "get-dependencies.pip")
	if err := os.WriteFile(pipPath, []byte(strings.Join(pipLines, "\n")), 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write get-dependencies pip output").
			WithCause(err)
	}
	return nil
}

func (a CompatibilityOutputAdapter) WriteRosdepMapping(resolved []types.ResolvedDependency) error {
	aptDeps := map[string]string{}
	for _, dep := range resolved {
		if dep.Type != types.DependencyTypeApt {
			continue
		}
		aptDeps[dep.Package] = dep.Version
	}
	var names []string
	for name := range aptDeps {
		names = append(names, name)
	}
	sort.Strings(names)
	var builder strings.Builder
	for _, name := range names {
		builder.WriteString(name)
		builder.WriteString(":\n")
		builder.WriteString("  ubuntu:\n")
		builder.WriteString("    apt:\n")
		builder.WriteString("      - ")
		builder.WriteString(name)
		builder.WriteString("=")
		builder.WriteString(aptDeps[name])
		builder.WriteString("\n")
	}
	if err := os.MkdirAll(a.Dir, 0755); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create output directory").
			WithCause(err)
	}
	path := filepath.Join(a.Dir, "rosdep-mapping.yaml")
	if err := os.WriteFile(path, []byte(builder.String()), 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write rosdep mapping output").
			WithCause(err)
	}
	return nil
}

var _ ports.CompatibilityPort = CompatibilityOutputAdapter{}
