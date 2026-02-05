package policies

import (
	"fmt"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/types"
)

type PackagingPolicy struct {
	Groups         []types.PackagingGroup
	TargetUbuntu   string
	exactByType    map[types.DependencyType]map[string]int
	exactAny       map[string]int
	prefixByType   map[types.DependencyType][]prefixPattern
	prefixAny      []prefixPattern
	wildcardByType map[types.DependencyType]int
	wildcardAny    int
}

func NewPackagingPolicy(groups []types.PackagingGroup, targetUbuntu string) PackagingPolicy {
	policy := PackagingPolicy{
		TargetUbuntu: targetUbuntu,
		wildcardAny:  -1,
	}
	for _, group := range groups {
		if !matchesTarget(targetUbuntu, group.Targets) {
			continue
		}
		policy.Groups = append(policy.Groups, group)
	}
	policy.compile()
	return policy
}

func (p PackagingPolicy) ResolvePackagingMode(depType types.DependencyType, name string) (types.PackagingGroup, error) {
	best := -1
	if matches, ok := p.exactByType[depType]; ok {
		if idx, found := matches[name]; found {
			best = minIndex(best, idx)
		}
	}
	if idx, found := p.exactAny[name]; found {
		best = minIndex(best, idx)
	}
	for _, entry := range p.prefixByType[depType] {
		if strings.HasPrefix(name, entry.prefix) {
			best = minIndex(best, entry.groupIndex)
		}
	}
	for _, entry := range p.prefixAny {
		if strings.HasPrefix(name, entry.prefix) {
			best = minIndex(best, entry.groupIndex)
		}
	}
	if idx, found := p.wildcardByType[depType]; found {
		best = minIndex(best, idx)
	}
	if p.wildcardAny >= 0 {
		best = minIndex(best, p.wildcardAny)
	}
	if best >= 0 && best < len(p.Groups) {
		return p.Groups[best], nil
	}
	return types.PackagingGroup{}, errbuilder.New().
		WithCode(errbuilder.CodeNotFound).
		WithMsg(fmt.Sprintf("no packaging group matches %s:%s", depType, name))
}

type prefixPattern struct {
	prefix     string
	groupIndex int
}

type parsedPattern struct {
	depType *types.DependencyType
	kind    patternKind
	name    string
}

type patternKind int

const (
	patternExact patternKind = iota
	patternPrefix
	patternWildcard
	patternInvalid
)

func (p *PackagingPolicy) compile() {
	p.exactByType = map[types.DependencyType]map[string]int{}
	p.exactAny = map[string]int{}
	p.prefixByType = map[types.DependencyType][]prefixPattern{}
	p.prefixAny = nil
	p.wildcardByType = map[types.DependencyType]int{}
	p.wildcardAny = -1
	for idx, group := range p.Groups {
		for _, pattern := range group.Matches {
			parsed, ok := parsePattern(pattern)
			if !ok {
				continue
			}
			switch parsed.kind {
			case patternWildcard:
				p.storeWildcard(parsed.depType, idx)
			case patternExact:
				p.storeExact(parsed.depType, parsed.name, idx)
			case patternPrefix:
				p.storePrefix(parsed.depType, parsed.name, idx)
			}
		}
	}
}

func (p *PackagingPolicy) storeExact(depType *types.DependencyType, name string, index int) {
	if depType == nil {
		if _, ok := p.exactAny[name]; !ok {
			p.exactAny[name] = index
		}
		return
	}
	if p.exactByType[*depType] == nil {
		p.exactByType[*depType] = map[string]int{}
	}
	if _, ok := p.exactByType[*depType][name]; !ok {
		p.exactByType[*depType][name] = index
	}
}

func (p *PackagingPolicy) storePrefix(depType *types.DependencyType, prefix string, index int) {
	entry := prefixPattern{prefix: prefix, groupIndex: index}
	if depType == nil {
		p.prefixAny = append(p.prefixAny, entry)
		return
	}
	p.prefixByType[*depType] = append(p.prefixByType[*depType], entry)
}

func (p *PackagingPolicy) storeWildcard(depType *types.DependencyType, index int) {
	if depType == nil {
		if p.wildcardAny < 0 {
			p.wildcardAny = index
		}
		return
	}
	if _, ok := p.wildcardByType[*depType]; !ok {
		p.wildcardByType[*depType] = index
	}
}

func parsePattern(pattern string) (parsedPattern, bool) {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return parsedPattern{kind: patternInvalid}, false
	}
	if trimmed == "*" {
		return parsedPattern{kind: patternWildcard}, true
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) == 2 {
		depType, ok := parseDepType(parts[0])
		if !ok {
			return parsedPattern{kind: patternInvalid}, false
		}
		name, kind := parseNamePattern(parts[1])
		if kind == patternInvalid {
			return parsedPattern{kind: patternInvalid}, false
		}
		return parsedPattern{depType: &depType, kind: kind, name: name}, true
	}
	name, kind := parseNamePattern(trimmed)
	if kind == patternInvalid {
		return parsedPattern{kind: patternInvalid}, false
	}
	return parsedPattern{kind: kind, name: name}, true
}

func parseDepType(token string) (types.DependencyType, bool) {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "apt":
		return types.DependencyTypeApt, true
	case "pip", "python":
		return types.DependencyTypePip, true
	default:
		return "", false
	}
}

func parseNamePattern(value string) (string, patternKind) {
	pattern := strings.TrimSpace(value)
	if pattern == "" {
		return "", patternInvalid
	}
	if pattern == "*" {
		return "", patternWildcard
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.TrimSuffix(pattern, "*"), patternPrefix
	}
	return pattern, patternExact
}

func minIndex(current int, candidate int) int {
	if candidate < 0 {
		return current
	}
	if current < 0 || candidate < current {
		return candidate
	}
	return current
}

func matchesTarget(target string, targets []string) bool {
	if target == "" {
		return true
	}
	normalizedTarget := normalizeUbuntuTarget(target)
	for _, entry := range targets {
		if normalizeUbuntuTarget(entry) == normalizedTarget {
			return true
		}
	}
	return false
}

func normalizeUbuntuTarget(value string) string {
	normalized := strings.TrimSpace(value)
	lower := strings.ToLower(normalized)
	if strings.HasPrefix(lower, "ubuntu-") {
		return strings.TrimSpace(normalized[len("ubuntu-"):])
	}
	return normalized
}