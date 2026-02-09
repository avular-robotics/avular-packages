package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/rs/zerolog/log"

	"avular-packages/internal/policies"
	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

type ResolverCore struct {
	RepoIndex    ports.RepoIndexPort
	Policy       ports.PolicyPort
	UseAptSolver bool
}

type ResolveResult struct {
	AptLocks       []types.AptLockEntry
	BundleManifest []types.BundleManifestEntry
	ResolvedDeps   []types.ResolvedDependency
	Resolution     types.ResolutionReport
}

func NewResolverCore(repoIndex ports.RepoIndexPort, policy ports.PolicyPort) ResolverCore {
	return ResolverCore{
		RepoIndex: repoIndex,
		Policy:    policy,
	}
}

func (r ResolverCore) Resolve(ctx context.Context, deps []types.Dependency, directives []types.ResolutionDirective) (ResolveResult, error) {
	if r.RepoIndex == nil || r.Policy == nil {
		return ResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("resolver requires repo index and policy ports")
	}

	merged := mergeDependencies(deps)
	directiveMap := mapDirectives(directives)

	result := ResolveResult{
		Resolution: types.ResolutionReport{Records: []types.ResolutionRecord{}},
	}

	aptSolverDeps := map[string]types.Dependency{}
	aptSolverGroups := map[string]types.PackagingGroup{}
	for _, dep := range merged {
		group, err := r.Policy.ResolvePackagingMode(dep.Type, dep.Name)
		if err != nil {
			return ResolveResult{}, err
		}
		pinned, err := applyGroupPins(dep, group)
		if err != nil {
			return ResolveResult{}, err
		}
		if r.UseAptSolver && dep.Type == types.DependencyTypeApt {
			updated, record, err := r.prepareDependency(pinned, directiveMap)
			if err != nil {
				return ResolveResult{}, err
			}
			if record.Action != "" {
				result.Resolution.Records = append(result.Resolution.Records, record)
			}
			key := normalizeDirectiveKey(fmt.Sprintf("%s:%s", updated.Type, updated.Name))
			aptSolverDeps[key] = updated
			aptSolverGroups[updated.Name] = group
			continue
		}

		version, record, err := r.resolveDependency(ctx, pinned, directiveMap)
		if err != nil {
			return ResolveResult{}, err
		}
		if record.Action != "" {
			result.Resolution.Records = append(result.Resolution.Records, record)
		}

		lockName := aptLockPackageName(dep)
		result.AptLocks = append(result.AptLocks, types.AptLockEntry{
			Package: lockName,
			Version: version,
		})
		result.ResolvedDeps = append(result.ResolvedDeps, types.ResolvedDependency{
			Type:    dep.Type,
			Package: dep.Name,
			Version: version,
		})

		result.BundleManifest = append(result.BundleManifest, types.BundleManifestEntry{
			Group:   group.Name,
			Mode:    group.Mode,
			Package: dep.Name,
			Version: version,
		})
	}

	if r.UseAptSolver && len(aptSolverDeps) > 0 {
		solved, err := resolveAptWithSolver(ctx, r.RepoIndex, mapValues(aptSolverDeps))
		if err != nil {
			return ResolveResult{}, err
		}
		lockSet := map[string]string{}
		for _, entry := range result.AptLocks {
			lockSet[entry.Package] = entry.Version
		}
		for name, version := range solved {
			lockSet[name] = version
			result.ResolvedDeps = append(result.ResolvedDeps, types.ResolvedDependency{
				Type:    types.DependencyTypeApt,
				Package: name,
				Version: version,
			})
		}
		result.AptLocks = result.AptLocks[:0]
		for name, version := range lockSet {
			result.AptLocks = append(result.AptLocks, types.AptLockEntry{
				Package: name,
				Version: version,
			})
		}
		for _, dep := range aptSolverDeps {
			version, ok := solved[dep.Name]
			if !ok {
				continue
			}
			group, ok := aptSolverGroups[dep.Name]
			if !ok {
				continue
			}
			result.BundleManifest = append(result.BundleManifest, types.BundleManifestEntry{
				Group:   group.Name,
				Mode:    group.Mode,
				Package: dep.Name,
				Version: version,
			})
		}
	}

	sort.Slice(result.AptLocks, func(i, j int) bool {
		return result.AptLocks[i].Package < result.AptLocks[j].Package
	})

	log.Ctx(ctx).Debug().Int("resolved", len(result.AptLocks)).Msg("resolver completed")
	return result, nil
}

func (r ResolverCore) prepareDependency(dep types.Dependency, directiveMap map[string]types.ResolutionDirective) (types.Dependency, types.ResolutionRecord, error) {
	directive, ok := directiveFor(dep, directiveMap)
	if !ok {
		return dep, types.ResolutionRecord{}, nil
	}
	updated, record, err := policies.ApplyResolution(dep, directive)
	if err != nil {
		return types.Dependency{}, record, err
	}
	return updated, record, nil
}

func (r ResolverCore) resolveDependency(ctx context.Context, dep types.Dependency, directiveMap map[string]types.ResolutionDirective) (string, types.ResolutionRecord, error) {
	available, err := r.RepoIndex.AvailableVersions(dep.Type, dep.Name)
	if err != nil {
		return "", types.ResolutionRecord{}, err
	}
	version, err := bestCompatibleVersion(dep, available)
	if err == nil {
		return version, types.ResolutionRecord{}, nil
	}

	directive, ok := directiveFor(dep, directiveMap)
	if !ok {
		return "", types.ResolutionRecord{}, errbuilder.New().
			WithCode(errbuilder.CodeFailedPrecondition).
			WithMsg(fmt.Sprintf("conflict without resolution directive: %s", dep.Name)).
			WithCause(err)
	}

	updated, record, err := policies.ApplyResolution(dep, directive)
	if err != nil {
		return "", types.ResolutionRecord{}, err
	}

	available, err = r.RepoIndex.AvailableVersions(updated.Type, updated.Name)
	if err != nil {
		return "", types.ResolutionRecord{}, err
	}
	version, err = bestCompatibleVersion(updated, available)
	if err != nil {
		return "", types.ResolutionRecord{}, err
	}
	log.Ctx(ctx).Debug().Str("dependency", dep.Name).Msg("resolution directive applied")
	return version, record, nil
}

func mergeDependencies(deps []types.Dependency) []types.Dependency {
	type key struct {
		depType types.DependencyType
		name    string
	}
	merged := map[key]types.Dependency{}
	for _, dep := range deps {
		k := key{depType: dep.Type, name: dep.Name}
		existing, ok := merged[k]
		if !ok {
			merged[k] = dep
			continue
		}
		existing.Constraints = append(existing.Constraints, dep.Constraints...)
		merged[k] = existing
	}
	var out []types.Dependency
	for _, dep := range merged {
		dep.Constraints = filterConstraintsByPriority(dep.Constraints)
		out = append(out, dep)
	}
	return out
}

func mapDirectives(directives []types.ResolutionDirective) map[string]types.ResolutionDirective {
	mapped := map[string]types.ResolutionDirective{}
	for _, directive := range directives {
		if directive.Dependency == "" {
			continue
		}
		mapped[normalizeDirectiveKey(directive.Dependency)] = directive
	}
	return mapped
}

func directiveFor(dep types.Dependency, directives map[string]types.ResolutionDirective) (types.ResolutionDirective, bool) {
	key := fmt.Sprintf("%s:%s", dep.Type, dep.Name)
	directive, ok := directives[key]
	return directive, ok
}

func applyGroupPins(dep types.Dependency, group types.PackagingGroup) (types.Dependency, error) {
	if len(group.Pins) == 0 {
		return dep, nil
	}
	for _, pin := range group.Pins {
		constraint, err := ParseConstraint(pin, "packaging:pin")
		if err != nil {
			return dep, err
		}
		if constraint.Name != dep.Name {
			continue
		}
		dep.Constraints = append(dep.Constraints, constraint)
	}
	return dep, nil
}

func filterConstraintsByPriority(constraints []types.Constraint) []types.Constraint {
	if len(constraints) == 0 {
		return constraints
	}
	maxPriority := -1
	for _, constraint := range constraints {
		priority := constraintPriority(constraint.Source)
		if priority > maxPriority {
			maxPriority = priority
		}
	}
	if maxPriority < 0 {
		return constraints
	}
	var top []types.Constraint
	for _, constraint := range constraints {
		if constraintPriority(constraint.Source) == maxPriority {
			top = append(top, constraint)
		}
	}
	hasHard := false
	for _, constraint := range top {
		if constraint.Op != types.ConstraintOpNone {
			hasHard = true
			break
		}
	}
	if hasHard {
		var hard []types.Constraint
		for _, constraint := range top {
			if constraint.Op != types.ConstraintOpNone {
				hard = append(hard, constraint)
			}
		}
		return hard
	}
	var fallback []types.Constraint
	for _, constraint := range constraints {
		if constraint.Op == types.ConstraintOpNone {
			continue
		}
		fallback = append(fallback, constraint)
	}
	return fallback
}

func normalizeDirectiveKey(value string) string {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return value
	}
	depType := strings.ToLower(strings.TrimSpace(parts[0]))
	name := strings.TrimSpace(parts[1])
	if depType == "pip" {
		name = normalizePipName(name)
	}
	return fmt.Sprintf("%s:%s", depType, name)
}

func aptLockPackageName(dep types.Dependency) string {
	if dep.Type == types.DependencyTypePip {
		return normalizeDebPackageName("python3-" + dep.Name)
	}
	return dep.Name
}

func normalizeDebPackageName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return normalized
}

func mapValues(values map[string]types.Dependency) []types.Dependency {
	out := make([]types.Dependency, 0, len(values))
	for _, dep := range values {
		out = append(out, dep)
	}
	return out
}

func constraintPriority(source string) int {
	normalized := strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.HasPrefix(normalized, "product:"):
		return 3
	case strings.HasPrefix(normalized, "profile:"):
		return 2
	case strings.HasPrefix(normalized, "package_xml:"):
		return 1
	default:
		return 0
	}
}
