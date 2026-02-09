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

type aptVarKey struct {
	Name    string
	Version string
}

type aptDepSpec struct {
	Name        string
	Constraints []types.Constraint
}

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
	versionCache := newVersionCache(types.DependencyTypeApt)
	nameToVersionID := map[string]map[string]int{}
	packageVars := map[string][]int{}
	varMeta := map[int]types.AptPackageVersion{}
	varKey := map[int]aptVarKey{}
	varID := 0
	var costLits []solver.Lit
	var costWeights []int

	for name, versions := range aptPackages {
		ordered := sortAptPackageVersions(versions, versionCache)
		ids := make([]int, 0, len(ordered))
		for i, entry := range ordered {
			if entry.Version == "" {
				continue
			}
			varID++
			id := varID
			if nameToVersionID[name] == nil {
				nameToVersionID[name] = map[string]int{}
			}
			nameToVersionID[name][entry.Version] = id
			ids = append(ids, id)
			varMeta[id] = entry
			varKey[id] = aptVarKey{Name: name, Version: entry.Version}
			weight := len(ordered) - 1 - i
			costLits = append(costLits, solver.IntToLit(int32(id)))
			costWeights = append(costWeights, weight)
		}
		if len(ids) > 0 {
			packageVars[name] = ids
		}
	}
	if varID == 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeFailedPrecondition).
			WithMsg("apt solver received no package versions to solve")
	}

	providers := buildProvideIndex(aptPackages)
	var clauses [][]int
	for _, ids := range packageVars {
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				clauses = append(clauses, []int{-ids[i], -ids[j]})
			}
		}
	}
	for _, dep := range deps {
		if strings.TrimSpace(dep.Name) == "" {
			continue
		}
		candidates, err := candidatesForSpec(dep.Name, dep.Constraints, nameToVersionID, packageVars, providers, varMeta, versionCache)
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
	for id, meta := range varMeta {
		groups := append([]string{}, meta.Depends...)
		groups = append(groups, meta.PreDepends...)
		for _, group := range groups {
			alts := parseAptAlternatives(group)
			var candidates []int
			for _, alt := range alts {
				ids, err := candidatesForSpec(alt.Name, alt.Constraints, nameToVersionID, packageVars, providers, varMeta, versionCache)
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

	problem := solver.ParseSliceNb(clauses, varID)
	problem.SetCostFunc(costLits, costWeights)
	s := solver.New(problem)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if cost := s.Minimize(); cost < 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeFailedPrecondition).
			WithMsg("apt solver found no satisfiable solution")
	}
	model := s.Model()
	selected := map[string]string{}
	for id, key := range varKey {
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
	if idx := strings.Index(raw, "("); idx >= 0 {
		name = strings.TrimSpace(raw[:idx])
		constraintPart = strings.TrimSpace(raw[idx+1:])
		if strings.HasSuffix(constraintPart, ")") {
			constraintPart = strings.TrimSuffix(constraintPart, ")")
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
