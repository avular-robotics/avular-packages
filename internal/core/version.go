package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"
	pep440 "github.com/aquasecurity/go-pep440-version"
	debversion "github.com/knqyf263/go-deb-version"

	"avular-packages/internal/types"
)

// preparedConstraint is a pre-parsed version constraint ready for
// repeated comparison. For APT it holds a parsed Debian version; for
// Pip it holds a PEP 440 specifier set.
type preparedConstraint struct {
	op  types.ConstraintOp
	deb debversion.Version
	pep pep440.Specifiers
}

// versionCache memoizes parsed version objects to avoid repeated parsing
// during constraint evaluation and sorting.
type versionCache struct {
	depType types.DependencyType
	deb     map[string]debversion.Version
	pep     map[string]pep440.Version
	spec    map[string]pep440.Specifiers
}

// newVersionCache creates an empty cache for the given dependency type.
func newVersionCache(depType types.DependencyType) *versionCache {
	return &versionCache{
		depType: depType,
		deb:     map[string]debversion.Version{},
		pep:     map[string]pep440.Version{},
		spec:    map[string]pep440.Specifiers{},
	}
}

// debVersion returns a parsed Debian version, caching the result.
func (c *versionCache) debVersion(value string) (debversion.Version, error) {
	if parsed, ok := c.deb[value]; ok {
		return parsed, nil
	}
	parsed, err := debversion.NewVersion(value)
	if err != nil {
		return debversion.Version{}, err
	}
	c.deb[value] = parsed
	return parsed, nil
}

// pepVersion returns a parsed PEP 440 version, caching the result.
func (c *versionCache) pepVersion(value string) (pep440.Version, error) {
	if parsed, ok := c.pep[value]; ok {
		return parsed, nil
	}
	parsed, err := pep440.Parse(value)
	if err != nil {
		return pep440.Version{}, err
	}
	c.pep[value] = parsed
	return parsed, nil
}

// pepSpec returns parsed PEP 440 specifiers, caching the result.
func (c *versionCache) pepSpec(value string) (pep440.Specifiers, error) {
	if parsed, ok := c.spec[value]; ok {
		return parsed, nil
	}
	parsed, err := pep440.NewSpecifiers(value)
	if err != nil {
		return pep440.Specifiers{}, err
	}
	c.spec[value] = parsed
	return parsed, nil
}

// compare returns -1, 0, or 1 comparing two version strings using the
// cache's dependency type semantics. Returns 0 on parse errors.
func (c *versionCache) compare(a string, b string) int {
	switch c.depType {
	case types.DependencyTypeApt:
		v1, err := c.debVersion(a)
		if err != nil {
			return 0
		}
		v2, err := c.debVersion(b)
		if err != nil {
			return 0
		}
		return v1.Compare(v2)
	case types.DependencyTypePip:
		v1, err := c.pepVersion(a)
		if err != nil {
			return 0
		}
		v2, err := c.pepVersion(b)
		if err != nil {
			return 0
		}
		return v1.Compare(v2)
	default:
		return 0
	}
}

// bestCompatibleVersion selects the highest version from available that
// satisfies all of the dependency's constraints. Returns an error if
// no compatible version exists.
func bestCompatibleVersion(dep types.Dependency, available []string) (string, error) {
	if len(available) == 0 {
		return "", errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg(fmt.Sprintf("no available versions for %s", dep.Name))
	}
	cache := newVersionCache(dep.Type)
	parsedConstraints, err := prepareConstraints(dep.Type, dep.Constraints, cache)
	if err != nil {
		return "", err
	}
	var candidates []string
	for _, version := range available {
		ok, err := satisfiesAll(dep.Type, version, parsedConstraints, cache)
		if err != nil {
			return "", err
		}
		if ok {
			candidates = append(candidates, version)
		}
	}
	if len(candidates) == 0 {
		return "", errbuilder.New().
			WithCode(errbuilder.CodeFailedPrecondition).
			WithMsg(fmt.Sprintf("no compatible version for %s", dep.Name))
	}
	sort.Slice(candidates, func(i, j int) bool {
		return cache.compare(candidates[i], candidates[j]) > 0
	})
	return candidates[0], nil
}

// prepareConstraints parses each constraint's version string upfront so
// it can be reused across multiple candidate comparisons.
func prepareConstraints(depType types.DependencyType, constraints []types.Constraint, cache *versionCache) ([]preparedConstraint, error) {
	var out []preparedConstraint
	for _, constraint := range constraints {
		if constraint.Op == types.ConstraintOpNone {
			continue
		}
		switch depType {
		case types.DependencyTypeApt:
			parsed, err := cache.debVersion(constraint.Version)
			if err != nil {
				return nil, err
			}
			out = append(out, preparedConstraint{op: constraint.Op, deb: parsed})
		case types.DependencyTypePip:
			spec, err := cache.pepSpec(toPep440Spec(constraint))
			if err != nil {
				return nil, err
			}
			out = append(out, preparedConstraint{op: constraint.Op, pep: spec})
		default:
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("unsupported dependency type")
		}
	}
	return out, nil
}

// satisfiesAll dispatches to the type-specific constraint checker.
func satisfiesAll(depType types.DependencyType, version string, constraints []preparedConstraint, cache *versionCache) (bool, error) {
	if len(constraints) == 0 {
		return true, nil
	}
	switch depType {
	case types.DependencyTypeApt:
		return satisfiesDeb(version, constraints, cache)
	case types.DependencyTypePip:
		return satisfiesPep440(version, constraints, cache)
	default:
		return false, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("unsupported dependency type")
	}
}

// satisfiesDeb checks a Debian version against all prepared constraints.
func satisfiesDeb(version string, constraints []preparedConstraint, cache *versionCache) (bool, error) {
	v, err := cache.debVersion(version)
	if err != nil {
		return false, err
	}
	for _, constraint := range constraints {
		c := constraint.deb
		switch constraint.op {
		case types.ConstraintOpEq, types.ConstraintOpEq2:
			if !v.Equal(c) {
				return false, nil
			}
		case types.ConstraintOpGte:
			if v.LessThan(c) && !v.Equal(c) {
				return false, nil
			}
		case types.ConstraintOpLte:
			if v.GreaterThan(c) && !v.Equal(c) {
				return false, nil
			}
		case types.ConstraintOpGt:
			if !v.GreaterThan(c) {
				return false, nil
			}
		case types.ConstraintOpLt:
			if !v.LessThan(c) {
				return false, nil
			}
		default:
			return false, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("unsupported constraint operator")
		}
	}
	return true, nil
}

// satisfiesPep440 checks a PEP 440 version against all prepared specifiers.
func satisfiesPep440(version string, constraints []preparedConstraint, cache *versionCache) (bool, error) {
	parsed, err := cache.pepVersion(version)
	if err != nil {
		return false, err
	}
	for _, constraint := range constraints {
		if !constraint.pep.Check(parsed) {
			return false, nil
		}
	}
	return true, nil
}

// toPep440Spec converts an internal constraint to a PEP 440 specifier
// string (e.g. ">= 1.0", "~= 2.3").
func toPep440Spec(constraint types.Constraint) string {
	op := string(constraint.Op)
	switch constraint.Op {
	case types.ConstraintOpEq:
		op = "=="
	case types.ConstraintOpEq2:
		op = "=="
	case types.ConstraintOpNe:
		op = "!="
	case types.ConstraintOpCompat:
		op = "~="
	}
	return strings.TrimSpace(fmt.Sprintf("%s %s", op, constraint.Version))
}
