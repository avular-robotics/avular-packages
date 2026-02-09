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
	Workspace      ports.WorkspacePort
	PackageXML     ports.PackageXMLPort
	SchemaResolver ports.SchemaResolverPort
}

func NewDependencyBuilder(workspace ports.WorkspacePort, pkgXML ports.PackageXMLPort) DependencyBuilder {
	return DependencyBuilder{
		Workspace:  workspace,
		PackageXML: pkgXML,
	}
}

// WithSchemaResolver attaches a schema resolver for mapping standard ROS
// tags to concrete typed dependencies.
func (b DependencyBuilder) WithSchemaResolver(sr ports.SchemaResolverPort) DependencyBuilder {
	b.SchemaResolver = sr
	return b
}

func (b DependencyBuilder) Build(ctx context.Context, inputs types.Inputs, workspaceRoots []string) ([]types.Dependency, error) {
	return b.BuildWithSchema(ctx, inputs, workspaceRoots, nil)
}

// BuildWithSchema is like Build but accepts an optional inline schema
// that is loaded before any file-based schema_files.
func (b DependencyBuilder) BuildWithSchema(ctx context.Context, inputs types.Inputs, workspaceRoots []string, inlineSchema *types.SchemaFile) ([]types.Dependency, error) {
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

		// Parse export-section typed dependencies (debian_depend, pip_depend)
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

		// Parse standard ROS tags and resolve through schema
		schemaDeps, err := b.resolveROSTags(ctx, packageXMLPaths, inputs, inlineSchema)
		if err != nil {
			return nil, err
		}
		deps = append(deps, schemaDeps...)
	}

	log.Ctx(ctx).Debug().Int("deps", len(deps)).Msg("dependencies collected")
	return deps, nil
}

func (b DependencyBuilder) BuildFromSpecs(ctx context.Context, product types.Spec, profiles []types.Spec, inputs types.Inputs, workspaceRoots []string) ([]types.Dependency, error) {
	return b.BuildFromSpecsWithSchema(ctx, product, profiles, inputs, workspaceRoots, nil)
}

// BuildFromSpecsWithSchema is like BuildFromSpecs but accepts an
// optional inline schema loaded before file-based schemas.
func (b DependencyBuilder) BuildFromSpecsWithSchema(ctx context.Context, product types.Spec, profiles []types.Spec, inputs types.Inputs, workspaceRoots []string, inlineSchema *types.SchemaFile) ([]types.Dependency, error) {
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

		// Parse export-section typed dependencies (debian_depend, pip_depend)
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

		// Parse standard ROS tags and resolve through schema
		schemaDeps, err := b.resolveROSTags(ctx, packageXMLPaths, inputs, inlineSchema)
		if err != nil {
			return nil, err
		}
		deps = append(deps, schemaDeps...)
	}

	log.Ctx(ctx).Debug().Int("deps", len(deps)).Msg("dependencies collected")
	return deps, nil
}

// resolveROSTags parses standard ROS tags from package.xml files and
// resolves them through the schema mapping.  If no SchemaResolver is
// configured, or no inline schema and no schema_files are listed,
// this is a no-op.
//
// Load order (later overrides earlier per key):
//  1. inlineSchema (from product/profile spec `schema:` field)
//  2. inputs.PackageXML.SchemaFiles (file-based schemas)
func (b DependencyBuilder) resolveROSTags(ctx context.Context, packageXMLPaths []string, inputs types.Inputs, inlineSchema *types.SchemaFile) ([]types.Dependency, error) {
	if b.SchemaResolver == nil {
		return nil, nil
	}

	hasInline := inlineSchema != nil && len(inlineSchema.Mappings) > 0
	hasFiles := len(inputs.PackageXML.SchemaFiles) > 0

	if !hasInline && !hasFiles {
		return nil, nil
	}

	// Load inline schema first (lowest precedence)
	if hasInline {
		if err := b.SchemaResolver.LoadSchemaInline(*inlineSchema); err != nil {
			return nil, err
		}
	}

	// Load file-based schema layers (override inline per key)
	for _, schemaPath := range inputs.PackageXML.SchemaFiles {
		if err := b.SchemaResolver.LoadSchema(schemaPath); err != nil {
			return nil, err
		}
	}

	// Parse abstract ROS tags
	rosTags, err := b.PackageXML.ParseROSTags(packageXMLPaths)
	if err != nil {
		return nil, err
	}
	if len(rosTags) == 0 {
		return nil, nil
	}

	// Filter out workspace-internal packages (same as export-tag filtering)
	if !inputs.PackageXML.IncludeSrc {
		packageNames, err := b.PackageXML.ParsePackageNames(packageXMLPaths)
		if err != nil {
			return nil, err
		}
		ignore := buildIgnoreSet(packageNames, inputs.PackageXML.Prefix)
		rosTags = filterROSTags(rosTags, ignore)
	}

	// Resolve through schema
	resolved, unknown, err := b.SchemaResolver.ResolveAll(rosTags)
	if err != nil {
		return nil, err
	}

	if len(unknown) > 0 {
		log.Ctx(ctx).Warn().
			Strs("keys", unknown).
			Int("count", len(unknown)).
			Msg("ROS tag keys not found in schema (skipped)")
	}

	log.Ctx(ctx).Debug().
		Int("ros_tags", len(rosTags)).
		Int("resolved", len(resolved)).
		Int("unknown", len(unknown)).
		Msg("schema-resolved ROS tag dependencies")

	return resolved, nil
}

// buildIgnoreSet builds a set of package names to exclude from
// dependency resolution (workspace-internal packages).
func buildIgnoreSet(packageNames []string, prefix string) map[string]struct{} {
	ignore := map[string]struct{}{}
	normalizedPrefix := strings.TrimSpace(prefix)
	for _, name := range packageNames {
		ignore[name] = struct{}{}
		hyphenName := strings.ReplaceAll(name, "_", "-")
		ignore[hyphenName] = struct{}{}
		if normalizedPrefix != "" {
			ignore[normalizedPrefix+name] = struct{}{}
			ignore[normalizedPrefix+hyphenName] = struct{}{}
		}
	}
	return ignore
}

// filterROSTags removes ROS tags whose keys match workspace-internal
// package names.
func filterROSTags(tags []types.ROSTagDependency, ignore map[string]struct{}) []types.ROSTagDependency {
	var filtered []types.ROSTagDependency
	for _, tag := range tags {
		if _, skip := ignore[tag.Key]; skip {
			continue
		}
		if _, skip := ignore[strings.ReplaceAll(tag.Key, "_", "-")]; skip {
			continue
		}
		filtered = append(filtered, tag)
	}
	return filtered
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
