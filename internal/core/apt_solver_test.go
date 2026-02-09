package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

// ---------------------------------------------------------------------------
// parseAptDepSpec
// ---------------------------------------------------------------------------

func TestParseAptDepSpec(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect aptDepSpec
	}{
		{
			name:   "simple package name",
			input:  "libfoo",
			expect: aptDepSpec{Name: "libfoo"},
		},
		{
			name:  "with version constraint >=",
			input: "libfoo (>= 1.2.0)",
			expect: aptDepSpec{
				Name: "libfoo",
				Constraints: []types.Constraint{
					{Name: "libfoo", Op: types.ConstraintOpGte, Version: "1.2.0", Source: "apt:dep"},
				},
			},
		},
		{
			name:  "with version constraint <<",
			input: "libfoo (<< 2.0.0)",
			expect: aptDepSpec{
				Name: "libfoo",
				Constraints: []types.Constraint{
					{Name: "libfoo", Op: types.ConstraintOpLt, Version: "2.0.0", Source: "apt:dep"},
				},
			},
		},
		{
			name:  "with version constraint >>",
			input: "libfoo (>> 1.0)",
			expect: aptDepSpec{
				Name: "libfoo",
				Constraints: []types.Constraint{
					{Name: "libfoo", Op: types.ConstraintOpGt, Version: "1.0", Source: "apt:dep"},
				},
			},
		},
		{
			name:  "with version constraint =",
			input: "libfoo (= 1.2.3-1)",
			expect: aptDepSpec{
				Name: "libfoo",
				Constraints: []types.Constraint{
					{Name: "libfoo", Op: types.ConstraintOpEq, Version: "1.2.3-1", Source: "apt:dep"},
				},
			},
		},
		{
			name:  "with version constraint <=",
			input: "libfoo (<= 3.0)",
			expect: aptDepSpec{
				Name: "libfoo",
				Constraints: []types.Constraint{
					{Name: "libfoo", Op: types.ConstraintOpLte, Version: "3.0", Source: "apt:dep"},
				},
			},
		},
		{
			name:   "with arch qualifier stripped",
			input:  "libfoo:amd64",
			expect: aptDepSpec{Name: "libfoo"},
		},
		{
			name:   "with arch qualifier and constraint",
			input:  "libfoo:amd64 (>= 1.0)",
			expect: aptDepSpec{Name: "libfoo", Constraints: []types.Constraint{{Name: "libfoo", Op: types.ConstraintOpGte, Version: "1.0", Source: "apt:dep"}}},
		},
		{
			name:   "with arch restriction bracket stripped",
			input:  "libfoo [amd64]",
			expect: aptDepSpec{Name: "libfoo"},
		},
		{
			name:   "empty string",
			input:  "",
			expect: aptDepSpec{},
		},
		{
			name:   "whitespace only",
			input:  "   ",
			expect: aptDepSpec{},
		},
		{
			name:   "invalid operator ignored",
			input:  "libfoo (!!! 1.0)",
			expect: aptDepSpec{Name: "libfoo"},
		},
		{
			name:   "constraint with only operator no version",
			input:  "libfoo (>=)",
			expect: aptDepSpec{Name: "libfoo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAptDepSpec(tt.input)
			assert.Equal(t, tt.expect.Name, got.Name)
			assert.Equal(t, len(tt.expect.Constraints), len(got.Constraints))
			for i := range tt.expect.Constraints {
				assert.Equal(t, tt.expect.Constraints[i].Op, got.Constraints[i].Op)
				assert.Equal(t, tt.expect.Constraints[i].Version, got.Constraints[i].Version)
				assert.Equal(t, tt.expect.Constraints[i].Name, got.Constraints[i].Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseAptAlternatives
// ---------------------------------------------------------------------------

func TestParseAptAlternatives(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "single dep",
			input:  "libfoo",
			expect: []string{"libfoo"},
		},
		{
			name:   "two alternatives",
			input:  "libfoo | libbar",
			expect: []string{"libfoo", "libbar"},
		},
		{
			name:   "three alternatives with constraints",
			input:  "libfoo (>= 1.0) | libbar | libbaz (= 2.0)",
			expect: []string{"libfoo", "libbar", "libbaz"},
		},
		{
			name:   "empty string",
			input:  "",
			expect: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAptAlternatives(tt.input)
			names := make([]string, len(got))
			for i, spec := range got {
				names[i] = spec.Name
			}
			assert.Equal(t, tt.expect, names)
		})
	}
}

// ---------------------------------------------------------------------------
// normalizeAptDepName
// ---------------------------------------------------------------------------

func TestNormalizeAptDepName(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"libfoo", "libfoo"},
		{"libfoo:amd64", "libfoo"},
		{"libfoo:arm64", "libfoo"},
		{"  libfoo  ", "libfoo"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expect, normalizeAptDepName(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// aptConstraintOp
// ---------------------------------------------------------------------------

func TestAptConstraintOp(t *testing.T) {
	tests := []struct {
		token  string
		expect types.ConstraintOp
		ok     bool
	}{
		{">=", types.ConstraintOpGte, true},
		{"<=", types.ConstraintOpLte, true},
		{"=", types.ConstraintOpEq, true},
		{"<<", types.ConstraintOpLt, true},
		{">>", types.ConstraintOpGt, true},
		{"!=", "", false},
		{"==", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got, ok := aptConstraintOp(tt.token)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.expect, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// uniqueInts
// ---------------------------------------------------------------------------

func TestUniqueInts(t *testing.T) {
	tests := []struct {
		name   string
		input  []int
		expect []int
	}{
		{"no duplicates", []int{1, 2, 3}, []int{1, 2, 3}},
		{"with duplicates", []int{1, 2, 2, 3, 1}, []int{1, 2, 3}},
		{"all same", []int{5, 5, 5}, []int{5}},
		{"empty", []int{}, []int{}},
		{"nil", nil, []int{}},
		{"single", []int{42}, []int{42}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueInts(tt.input)
			assert.Equal(t, tt.expect, got)
		})
	}
}

// ---------------------------------------------------------------------------
// buildProvideIndex
// ---------------------------------------------------------------------------

func TestBuildProvideIndex(t *testing.T) {
	aptPackages := map[string][]types.AptPackageVersion{
		"libfoo": {
			{Version: "1.0.0", Provides: []string{"libfoo-compat (= 1.0.0)"}},
			{Version: "2.0.0", Provides: []string{"libfoo-compat (= 2.0.0)", "libbar-api"}},
		},
		"libbaz": {
			{Version: "1.0.0"},
		},
	}

	providers := buildProvideIndex(aptPackages)

	// libfoo-compat should have two providers
	assert.Len(t, providers["libfoo-compat"], 2)
	assert.Equal(t, "libfoo", providers["libfoo-compat"][0].Name)
	assert.Equal(t, "1.0.0", providers["libfoo-compat"][0].Version)
	assert.Equal(t, "libfoo", providers["libfoo-compat"][1].Name)
	assert.Equal(t, "2.0.0", providers["libfoo-compat"][1].Version)

	// libbar-api should have one provider
	assert.Len(t, providers["libbar-api"], 1)

	// libbaz provides nothing
	_, hasBaz := providers["libbaz"]
	assert.False(t, hasBaz)
}

func TestBuildProvideIndexSkipsEmptyVersions(t *testing.T) {
	aptPackages := map[string][]types.AptPackageVersion{
		"libfoo": {
			{Version: "", Provides: []string{"libfoo-compat"}},
		},
	}

	providers := buildProvideIndex(aptPackages)
	assert.Empty(t, providers)
}

// ---------------------------------------------------------------------------
// sortAptPackageVersions
// ---------------------------------------------------------------------------

func TestSortAptPackageVersions(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	versions := []types.AptPackageVersion{
		{Version: "2.0.0"},
		{Version: "1.0.0"},
		{Version: "1.5.0"},
		{Version: "0.9.0"},
	}

	sorted := sortAptPackageVersions(versions, cache)

	assert.Equal(t, "0.9.0", sorted[0].Version)
	assert.Equal(t, "1.0.0", sorted[1].Version)
	assert.Equal(t, "1.5.0", sorted[2].Version)
	assert.Equal(t, "2.0.0", sorted[3].Version)
}

func TestSortAptPackageVersionsSkipsEmpty(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	versions := []types.AptPackageVersion{
		{Version: "1.0.0"},
	}

	sorted := sortAptPackageVersions(versions, cache)
	assert.Len(t, sorted, 1)
	assert.Equal(t, "1.0.0", sorted[0].Version)
}

// ---------------------------------------------------------------------------
// resolveAptWithSolver -- integration-level tests
// ---------------------------------------------------------------------------

func TestResolveAptWithSolverEmptyDeps(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"libfoo": {{Version: "1.0.0"}},
		},
	}

	result, err := resolveAptWithSolver(context.Background(), repo, nil)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestResolveAptWithSolverEmptyRepo(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{},
	}
	deps := []types.Dependency{
		{Name: "libfoo", Type: types.DependencyTypeApt},
	}

	_, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apt solver requires repo index")
}

func TestResolveAptWithSolverSinglePackage(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"libfoo": {
				{Version: "1.0.0"},
				{Version: "2.0.0"},
			},
		},
	}
	deps := []types.Dependency{
		{Name: "libfoo", Type: types.DependencyTypeApt},
	}

	result, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.NoError(t, err)
	assert.Contains(t, result, "libfoo")
	// SAT solver with cost minimization should prefer the latest version
	assert.Equal(t, "2.0.0", result["libfoo"])
}

func TestResolveAptWithSolverConstrainedVersion(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"libfoo": {
				{Version: "1.0.0"},
				{Version: "2.0.0"},
				{Version: "3.0.0"},
			},
		},
	}
	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpLte, Version: "2.0.0", Source: "apt:dep"},
			},
		},
	}

	result, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", result["libfoo"])
}

func TestResolveAptWithSolverTransitiveDeps(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"app": {
				{Version: "1.0.0", Depends: []string{"liba (>= 1.0.0)"}},
			},
			"liba": {
				{Version: "1.0.0"},
				{Version: "2.0.0"},
			},
		},
	}
	deps := []types.Dependency{
		{Name: "app", Type: types.DependencyTypeApt},
	}

	result, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.NoError(t, err)
	assert.Contains(t, result, "app")
	assert.Contains(t, result, "liba")
	assert.Equal(t, "1.0.0", result["app"])
}

func TestResolveAptWithSolverAlternativeDeps(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"app": {
				{Version: "1.0.0", Depends: []string{"liba | libb"}},
			},
			"liba": {
				{Version: "1.0.0"},
			},
			"libb": {
				{Version: "1.0.0"},
			},
		},
	}
	deps := []types.Dependency{
		{Name: "app", Type: types.DependencyTypeApt},
	}

	result, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.NoError(t, err)
	assert.Contains(t, result, "app")
	// At least one of the alternatives must be selected
	_, hasA := result["liba"]
	_, hasB := result["libb"]
	assert.True(t, hasA || hasB, "solver must select at least one alternative")
}

func TestResolveAptWithSolverNoCandidate(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"libfoo": {
				{Version: "1.0.0"},
			},
		},
	}
	deps := []types.Dependency{
		{Name: "missing-pkg", Type: types.DependencyTypeApt},
	}

	_, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no apt candidates for missing-pkg")
}

func TestResolveAptWithSolverUnsatisfiableConstraints(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"libfoo": {
				{Version: "1.0.0"},
				{Version: "2.0.0"},
			},
		},
	}
	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpGte, Version: "5.0.0", Source: "apt:dep"},
			},
		},
	}

	_, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no apt candidates for libfoo")
}

func TestResolveAptWithSolverPreDepends(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"app": {
				{Version: "1.0.0", PreDepends: []string{"libc"}},
			},
			"libc": {
				{Version: "1.0.0"},
			},
		},
	}
	deps := []types.Dependency{
		{Name: "app", Type: types.DependencyTypeApt},
	}

	result, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.NoError(t, err)
	assert.Contains(t, result, "app")
	assert.Contains(t, result, "libc")
}

func TestResolveAptWithSolverProvidesVirtualPackage(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"app": {
				{Version: "1.0.0", Depends: []string{"mail-transport-agent"}},
			},
			"postfix": {
				{Version: "3.5.0", Provides: []string{"mail-transport-agent"}},
			},
		},
	}
	deps := []types.Dependency{
		{Name: "app", Type: types.DependencyTypeApt},
	}

	result, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.NoError(t, err)
	assert.Contains(t, result, "app")
	assert.Contains(t, result, "postfix")
}

func TestResolveAptWithSolverSkipsBlankDepNames(t *testing.T) {
	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"libfoo": {{Version: "1.0.0"}},
		},
	}
	deps := []types.Dependency{
		{Name: "", Type: types.DependencyTypeApt},
		{Name: "   ", Type: types.DependencyTypeApt},
		{Name: "libfoo", Type: types.DependencyTypeApt},
	}

	result, err := resolveAptWithSolver(context.Background(), repo, deps)
	require.NoError(t, err)
	assert.Contains(t, result, "libfoo")
}

func TestResolveAptWithSolverContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	repo := testRepoIndex{
		aptPackages: map[string][]types.AptPackageVersion{
			"libfoo": {{Version: "1.0.0"}},
		},
	}
	deps := []types.Dependency{
		{Name: "libfoo", Type: types.DependencyTypeApt},
	}

	_, err := resolveAptWithSolver(ctx, repo, deps)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// candidatesForSpec
// ---------------------------------------------------------------------------

func TestCandidatesForSpec(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)

	nameToVersionID := map[string]map[string]int{
		"libfoo": {"1.0.0": 1, "2.0.0": 2, "3.0.0": 3},
	}
	packageVars := map[string][]int{
		"libfoo": {1, 2, 3},
	}
	varMeta := map[int]types.AptPackageVersion{
		1: {Version: "1.0.0"},
		2: {Version: "2.0.0"},
		3: {Version: "3.0.0"},
	}
	providers := map[string][]aptVarKey{}

	t.Run("no constraints returns all", func(t *testing.T) {
		candidates, err := candidatesForSpec("libfoo", nil, nameToVersionID, packageVars, providers, varMeta, cache)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, candidates)
	})

	t.Run("gte constraint filters", func(t *testing.T) {
		constraints := []types.Constraint{
			{Name: "libfoo", Op: types.ConstraintOpGte, Version: "2.0.0"},
		}
		candidates, err := candidatesForSpec("libfoo", constraints, nameToVersionID, packageVars, providers, varMeta, cache)
		require.NoError(t, err)
		assert.Equal(t, []int{2, 3}, candidates)
	})

	t.Run("unknown package returns empty", func(t *testing.T) {
		candidates, err := candidatesForSpec("missing", nil, nameToVersionID, packageVars, providers, varMeta, cache)
		require.NoError(t, err)
		assert.Empty(t, candidates)
	})

	t.Run("virtual package via providers", func(t *testing.T) {
		providersWithVirtual := map[string][]aptVarKey{
			"virtual-pkg": {{Name: "libfoo", Version: "2.0.0"}},
		}
		candidates, err := candidatesForSpec("virtual-pkg", nil, nameToVersionID, packageVars, providersWithVirtual, varMeta, cache)
		require.NoError(t, err)
		assert.Equal(t, []int{2}, candidates)
	})
}
