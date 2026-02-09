package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/crillab/gophersat/solver"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

// aptVarKey maps a SAT variable ID back to its package name and version.
type aptVarKey struct {
	Name    string
	Version string
}

// aptDepSpec represents a single parsed APT dependency specification
// with an optional version constraint (e.g. "libfoo (>= 1.0)").
type aptDepSpec struct {
	Name        string
	Constraints []types.Constraint
}

// aptSolverState holds all bookkeeping for one SAT solver invocation.
// Isolating this avoids passing seven maps through every helper call.
type aptSolverState struct {
	nameToVersionID map[string]map[string]int
	packageVars     map[string][]int
	varMeta         map[int]types.AptPackageVersion
	varKey          map[int]aptVarKey
	providers       map[string][]aptVarKey
	cache           *versionCache
	varID           int
	costLits        []solver.Lit
	costWeights     []int
}

// resolveAptWithSolver uses a SAT solver to select the best compatible set
// of APT packages for the given dependency list, including transitive
// dependencies declared in Depends and Pre-Depends fields.
func resolveAptWithSolver(ctx context.Context, repo ports.RepoIndexPort, deps []types.Dependency) (map[string]string, error) {
	if len(deps) == 0 {
		return map[string]string{}, nil
	}
	aptPackages, err := repo.AptPackages()
	if err != nil {
		return nil, err
	}
	if len(aptPackages) == 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeFailedPrecondition).
			WithMsg("apt solver requires repo index with apt package metadata")
	}

	state := buildSolverState(aptPackages)
	if state.varID == 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeFailedPrecondition).
			WithMsg("apt solver received no package versions to solve")
	}

	clauses, err := buildSolverClauses(state, deps)
	if err != nil {
		return nil, err
	}

	return solveSAT(ctx, state, clauses)
}

// buildSolverState enumerates every (package, version) pair as a SAT
// variable and builds lookup indexes for candidates and providers.
func buildSolverState(aptPackages map[string][]types.AptPackageVersion) aptSolverState {
	s := aptSolverState{
		nameToVersionID: map[string]map[string]int{},
		packageVars:     map[string][]int{},
		varMeta:         map[int]types.AptPackageVersion{},
		varKey:          map[int]aptVarKey{},
		cache:           newVersionCache(types.DependencyTypeApt),
	}

	for name, versions := range aptPackages {
		ordered := sortAptPackageVersions(versions, s.cache)
		ids := make([]int, 0, len(ordered))
		for i, entry := range ordered {
			if entry.Version == "" {
				continue
			}
			s.varID++
			id := s.varID
			if s.nameToVersionID[name] == nil {
				s.nameToVersionID[name] = map[string]int{}
			}
			s.nameToVersionID[name][entry.Version] = id
			ids = append(ids, id)
			s.varMeta[id] = entry
			s.varKey[id] = aptVarKey{Name: name, Version: entry.Version}
			weight := len(ordered) - 1 - i
			s.costLits = append(s.costLits, solver.IntToLit(int32(id))) //nolint:gosec // id is bounded by the number of package versions, well within int32 range
			s.costWeights = append(s.costWeights, weight)
		}
		if len(ids) > 0 {
			s.packageVars[name] = ids
		}
	}
	s.providers = buildProvideIndex(aptPackages)
	return s
}

// buildSolverClauses generates three kinds of SAT clauses:
//  1. At-most-one: only one version of each package can be selected.
//  2. Root demands: each requested dependency must have at least one candidate.
//  3. Transitive: if a version is selected its Depends/PreDepends must be satisfiable.
func buildSolverClauses(s aptSolverState, deps []types.Dependency) ([][]int, error) {
	var clauses [][]int

	// At-most-one per package
	for _, ids := range s.packageVars {
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				clauses = append(clauses, []int{-ids[i], -ids[j]})
			}
		}
	}

	// Root dependency demands
	for _, dep := range deps {
		if strings.TrimSpace(dep.Name) == "" {
			continue
		}
		candidates, err := candidatesForSpec(dep.Name, dep.Constraints, s.nameToVersionID, s.packageVars, s.providers, s.varMeta, s.cache)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeFailedPrecondition).
				WithMsg(fmt.Sprintf("no apt candidates for %s", dep.Name))
		}
		clauses = append(clauses, candidates)
	}

	// Transitive dependency clauses
	transitives, err := buildTransitiveClauses(s)
	if err != nil {
		return nil, err
	}
	clauses = append(clauses, transitives...)
	return clauses, nil
}

// buildTransitiveClauses emits implication clauses for every version's
// Depends and PreDepends entries: if variable X is true, at least one
// candidate satisfying its dependency group must also be true.
func buildTransitiveClauses(s aptSolverState) ([][]int, error) {
	var clauses [][]int
	for id, meta := range s.varMeta {
		groups := append([]string{}, meta.Depends...)
		groups = append(groups, meta.PreDepends...)
		for _, group := range groups {
			alts := parseAptAlternatives(group)
			var candidates []int
			for _, alt := range alts {
				ids, err := candidatesForSpec(alt.Name, alt.Constraints, s.nameToVersionID, s.packageVars, s.providers, s.varMeta, s.cache)
				if err != nil {
					return nil, err
				}
				candidates = append(candidates, ids...)
			}
			candidates = uniqueInts(candidates)
			if len(candidates) == 0 {
				clauses = append(clauses, []int{-id})
				continue
			}
			clause := append([]int{-id}, candidates...)
			clauses = append(clauses, uniqueInts(clause))
		}
	}
	return clauses, nil
}

// solveSAT feeds the clauses to gophersat's optimization solver, extracts
// the selected (name, version) pairs from the model, and returns them.
func solveSAT(ctx context.Context, s aptSolverState, clauses [][]int) (map[string]string, error) {
	problem := solver.ParseSliceNb(clauses, s.varID)
	problem.SetCostFunc(s.costLits, s.costWeights)
	sat := solver.New(problem)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if cost := sat.Minimize(); cost < 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeFailedPrecondition).
			WithMsg("apt solver found no satisfiable solution")
	}
	model := sat.Model()
	selected := map[string]string{}
	for id, key := range s.varKey {
		if id-1 < 0 || id-1 >= len(model) {
			continue
		}
		if !model[id-1] {
			continue
		}
		selected[key.Name] = key.Version
	}
	if len(selected) == 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeFailedPrecondition).
			WithMsg("apt solver produced empty selection")
	}
	return selected, nil
}

// buildProvideIndex creates a reverse map from virtual package names to the
// concrete (package, version) pairs that declare them via Provides fields.
func buildProvideIndex(aptPackages map[string][]types.AptPackageVersion) map[string][]aptVarKey {
	out := map[string][]aptVarKey{}
	for name, versions := range aptPackages {
		for _, entry := range versions {
			if entry.Version == "" {
				continue
			}
			for _, provide := range entry.Provides {
				parsed := parseAptDepSpec(provide)
				if parsed.Name == "" {
					continue
				}
				out[parsed.Name] = append(out[parsed.Name], aptVarKey{Name: name, Version: entry.Version})
			}
		}
	}
	return out
}

// parseAptAlternatives splits a pipe-separated dependency group (e.g.
// "libfoo | libbar (>= 2)") into individual aptDepSpec values.
func parseAptAlternatives(group string) []aptDepSpec {
	parts := strings.Split(group, "|")
	var out []aptDepSpec
	for _, part := range parts {
		spec := parseAptDepSpec(part)
		if spec.Name == "" {
			continue
		}
		out = append(out, spec)
	}
	return out
}

// parseAptDepSpec parses a single APT dependency token such as
// "libfoo (>= 1.2) [amd64]" into an aptDepSpec with name and optional
// version constraint. Architecture qualifiers and arch filters are stripped.
func parseAptDepSpec(value string) aptDepSpec {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return aptDepSpec{}
	}
	if idx := strings.Index(raw, " ["); idx >= 0 {
		raw = strings.TrimSpace(raw[:idx])
	}
	name := raw
	constraintPart := ""
	if before, after, ok := strings.Cut(raw, "("); ok {
		name = strings.TrimSpace(before)
		constraintPart = strings.TrimSpace(after)
		if before, ok := strings.CutSuffix(constraintPart, ")"); ok {
			constraintPart = before
		}
	}
	name = normalizeAptDepName(name)
	if name == "" {
		return aptDepSpec{}
	}
	if constraintPart == "" {
		return aptDepSpec{Name: name}
	}
	fields := strings.Fields(constraintPart)
	if len(fields) < 2 {
		return aptDepSpec{Name: name}
	}
	opToken := fields[0]
	version := fields[1]
	op, ok := aptConstraintOp(opToken)
	if !ok {
		return aptDepSpec{Name: name}
	}
	return aptDepSpec{
		Name: name,
		Constraints: []types.Constraint{
			{
				Name:    name,
				Op:      op,
				Version: version,
				Source:  "apt:dep",
			},
		},
	}
}

// normalizeAptDepName strips architecture suffixes (":amd64") and
// whitespace from a raw APT package name.
func normalizeAptDepName(value string) string {
	name := strings.TrimSpace(value)
	if name == "" {
		return ""
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = strings.TrimSpace(name[:idx])
	}
	return name
}

// aptConstraintOp maps an APT version relation token (e.g. ">=", "<<")
// to the internal ConstraintOp type. Returns false if the token is not
// a recognized APT relation operator.
func aptConstraintOp(token string) (types.ConstraintOp, bool) {
	switch token {
	case ">=":
		return types.ConstraintOpGte, true
	case "<=":
		return types.ConstraintOpLte, true
	case "=":
		return types.ConstraintOpEq, true
	case "<<":
		return types.ConstraintOpLt, true
	case ">>":
		return types.ConstraintOpGt, true
	default:
		return "", false
	}
}

// candidatesForSpec returns the SAT variable IDs of all package versions
// that satisfy the given name and constraints, including versions from
// virtual package providers.
func candidatesForSpec(
	name string,
	constraints []types.Constraint,
	nameToVersionID map[string]map[string]int,
	packageVars map[string][]int,
	providers map[string][]aptVarKey,
	varMeta map[int]types.AptPackageVersion,
	cache *versionCache,
) ([]int, error) {
	var out []int
	if ids, ok := packageVars[name]; ok {
		for _, id := range ids {
			meta := varMeta[id]
			ok, err := versionSatisfiesConstraints(meta.Version, constraints, cache)
			if err != nil {
				return nil, err
			}
			if ok {
				out = append(out, id)
			}
		}
	}
	if provides, ok := providers[name]; ok {
		for _, provider := range provides {
			versionIDs, ok := nameToVersionID[provider.Name]
			if !ok {
				continue
			}
			id, ok := versionIDs[provider.Version]
			if !ok {
				continue
			}
			meta := varMeta[id]
			okMatch, err := versionSatisfiesConstraints(meta.Version, constraints, cache)
			if err != nil {
				return nil, err
			}
			if okMatch {
				out = append(out, id)
			}
		}
	}
	return uniqueInts(out), nil
}

// versionSatisfiesConstraints checks whether a version string satisfies
// all of the given constraints using Debian version comparison semantics.
func versionSatisfiesConstraints(version string, constraints []types.Constraint, cache *versionCache) (bool, error) {
	if len(constraints) == 0 {
		return true, nil
	}
	parsed, err := prepareConstraints(types.DependencyTypeApt, constraints, cache)
	if err != nil {
		return false, err
	}
	return satisfiesDeb(version, parsed, cache)
}

// sortAptPackageVersions returns a new slice sorted by Debian version
// ascending. Unparseable versions fall back to lexicographic ordering.
func sortAptPackageVersions(values []types.AptPackageVersion, cache *versionCache) []types.AptPackageVersion {
	ordered := append([]types.AptPackageVersion(nil), values...)
	sort.Slice(ordered, func(i, j int) bool {
		vi, err := cache.debVersion(ordered[i].Version)
		if err != nil {
			return ordered[i].Version < ordered[j].Version
		}
		vj, err := cache.debVersion(ordered[j].Version)
		if err != nil {
			return ordered[i].Version < ordered[j].Version
		}
		return vi.Compare(vj) < 0
	})
	return ordered
}

// uniqueInts deduplicates a slice of ints while preserving order.
func uniqueInts(values []int) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
