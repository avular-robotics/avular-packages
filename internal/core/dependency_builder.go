package core

import (
	"context"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/rs/zerolog/log"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

type DependencyBuilder struct {
	Workspace  ports.WorkspacePort
	PackageXML ports.PackageXMLPort
}

func NewDependencyBuilder(workspace ports.WorkspacePort, pkgXML ports.PackageXMLPort) DependencyBuilder {
	return DependencyBuilder{
		Workspace:  workspace,
		PackageXML: pkgXML,
	}
}

func (b DependencyBuilder) Build(ctx context.Context, inputs types.Inputs, workspaceRoots []string) ([]types.Dependency, error) {
	var deps []types.Dependency

	manualApt, err := parseEntries(inputs.Manual.Apt, types.DependencyTypeApt, "manual:apt")
	if err != nil {
		return nil, err
	}
	manualPip, err := parseEntries(inputs.Manual.Python, types.DependencyTypePip, "manual:pip")
	if err != nil {
		return nil, err
	}
	deps = append(deps, manualApt...)
	deps = append(deps, manualPip...)

	if inputs.PackageXML.Enabled {
		if len(workspaceRoots) == 0 {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("package_xml enabled but no workspace roots provided")
		}
		var packageXMLPaths []string
		for _, root := range workspaceRoots {
			paths, err := b.Workspace.FindPackageXML(root)
			if err != nil {
				return nil, err
			}
			packageXMLPaths = append(packageXMLPaths, paths...)
		}
		debianDeps, pipDeps, err := b.PackageXML.ParseDependencies(packageXMLPaths, inputs.PackageXML.Tags)
		if err != nil {
			return nil, err
		}
		if !inputs.PackageXML.IncludeSrc {
			packageNames, err := b.PackageXML.ParsePackageNames(packageXMLPaths)
			if err != nil {
				return nil, err
			}
			debianDeps = filterWorkspaceDeps(debianDeps, packageNames, inputs.PackageXML.Prefix)
			pipDeps = filterWorkspaceDeps(pipDeps, packageNames, inputs.PackageXML.Prefix)
		}

		debianParsed, err := parseEntries(debianDeps, types.DependencyTypeApt, "package_xml:debian_depend")
		if err != nil {
			return nil, err
		}
		pipParsed, err := parseEntries(pipDeps, types.DependencyTypePip, "package_xml:pip_depend")
		if err != nil {
			return nil, err
		}
		deps = append(deps, debianParsed...)
		deps = append(deps, pipParsed...)
	}

	log.Ctx(ctx).Debug().Int("deps", len(deps)).Msg("dependencies collected")
	return deps, nil
}

func (b DependencyBuilder) BuildFromSpecs(ctx context.Context, product types.Spec, profiles []types.Spec, inputs types.Inputs, workspaceRoots []string) ([]types.Dependency, error) {
	var deps []types.Dependency

	productApt, err := parseEntries(product.Inputs.Manual.Apt, types.DependencyTypeApt, "product:manual:apt")
	if err != nil {
		return nil, err
	}
	productPip, err := parseEntries(product.Inputs.Manual.Python, types.DependencyTypePip, "product:manual:pip")
	if err != nil {
		return nil, err
	}
	deps = append(deps, productApt...)
	deps = append(deps, productPip...)

	for _, profile := range profiles {
		profileApt, err := parseEntries(profile.Inputs.Manual.Apt, types.DependencyTypeApt, "profile:manual:apt")
		if err != nil {
			return nil, err
		}
		profilePip, err := parseEntries(profile.Inputs.Manual.Python, types.DependencyTypePip, "profile:manual:pip")
		if err != nil {
			return nil, err
		}
		deps = append(deps, profileApt...)
		deps = append(deps, profilePip...)
	}

	if inputs.PackageXML.Enabled {
		if len(workspaceRoots) == 0 {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("package_xml enabled but no workspace roots provided")
		}
		var packageXMLPaths []string
		for _, root := range workspaceRoots {
			paths, err := b.Workspace.FindPackageXML(root)
			if err != nil {
				return nil, err
			}
			packageXMLPaths = append(packageXMLPaths, paths...)
		}
		debianDeps, pipDeps, err := b.PackageXML.ParseDependencies(packageXMLPaths, inputs.PackageXML.Tags)
		if err != nil {
			return nil, err
		}
		if !inputs.PackageXML.IncludeSrc {
			packageNames, err := b.PackageXML.ParsePackageNames(packageXMLPaths)
			if err != nil {
				return nil, err
			}
			debianDeps = filterWorkspaceDeps(debianDeps, packageNames, inputs.PackageXML.Prefix)
			pipDeps = filterWorkspaceDeps(pipDeps, packageNames, inputs.PackageXML.Prefix)
		}

		debianParsed, err := parseEntries(debianDeps, types.DependencyTypeApt, "package_xml:debian_depend")
		if err != nil {
			return nil, err
		}
		pipParsed, err := parseEntries(pipDeps, types.DependencyTypePip, "package_xml:pip_depend")
		if err != nil {
			return nil, err
		}
		deps = append(deps, debianParsed...)
		deps = append(deps, pipParsed...)
	}

	log.Ctx(ctx).Debug().Int("deps", len(deps)).Msg("dependencies collected")
	return deps, nil
}

func parseEntries(entries []string, depType types.DependencyType, source string) ([]types.Dependency, error) {
	var deps []types.Dependency
	for _, entry := range entries {
		constraint, err := ParseConstraint(entry, source)
		if err != nil {
			return nil, err
		}
		if depType == types.DependencyTypePip {
			constraint.Name = normalizePipName(constraint.Name)
		}
		deps = append(deps, types.Dependency{
			Name:        constraint.Name,
			Type:        depType,
			Constraints: []types.Constraint{constraint},
		})
	}
	return deps, nil
}

func filterWorkspaceDeps(deps []string, workspaceNames []string, prefix string) []string {
	ignore := map[string]struct{}{}
	normalizedPrefix := strings.TrimSpace(prefix)
	for _, name := range workspaceNames {
		ignore[name] = struct{}{}
		hyphenName := strings.ReplaceAll(name, "_", "-")
		ignore[hyphenName] = struct{}{}
		if normalizedPrefix != "" {
			ignore[normalizedPrefix+name] = struct{}{}
			ignore[normalizedPrefix+hyphenName] = struct{}{}
		}
	}
	var filtered []string
	for _, dep := range deps {
		name := strings.TrimSpace(dep)
		if _, ok := ignore[name]; ok {
			continue
		}
		filtered = append(filtered, dep)
	}
	return filtered
}

func normalizePipName(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "-", ".", "-")
	return replacer.Replace(lower)
}
