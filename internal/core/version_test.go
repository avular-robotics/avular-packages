package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

// ---------------------------------------------------------------------------
// versionCache
// ---------------------------------------------------------------------------

func TestVersionCacheDebVersion(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)

	v1, err := cache.debVersion("1.0.0")
	require.NoError(t, err)

	// Second call should hit cache
	v2, err := cache.debVersion("1.0.0")
	require.NoError(t, err)
	assert.Equal(t, v1, v2)
}

func TestVersionCacheDebVersionInvalid(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	_, err := cache.debVersion("not-a-version!!!")
	require.Error(t, err)
}

func TestVersionCachePepVersion(t *testing.T) {
	cache := newVersionCache(types.DependencyTypePip)

	v1, err := cache.pepVersion("1.2.3")
	require.NoError(t, err)

	v2, err := cache.pepVersion("1.2.3")
	require.NoError(t, err)
	assert.Equal(t, v1, v2)
}

func TestVersionCachePepVersionInvalid(t *testing.T) {
	cache := newVersionCache(types.DependencyTypePip)
	_, err := cache.pepVersion("not-a-pep440!!!")
	require.Error(t, err)
}

func TestVersionCachePepSpec(t *testing.T) {
	cache := newVersionCache(types.DependencyTypePip)

	s1, err := cache.pepSpec(">=1.0,<2.0")
	require.NoError(t, err)

	s2, err := cache.pepSpec(">=1.0,<2.0")
	require.NoError(t, err)
	assert.Equal(t, s1, s2)
}

func TestVersionCachePepSpecInvalid(t *testing.T) {
	cache := newVersionCache(types.DependencyTypePip)
	_, err := cache.pepSpec(">>invalid<<")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// versionCache.compare
// ---------------------------------------------------------------------------

func TestVersionCacheCompareApt(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)

	assert.Equal(t, -1, cache.compare("1.0.0", "2.0.0"))
	assert.Equal(t, 0, cache.compare("1.0.0", "1.0.0"))
	assert.Equal(t, 1, cache.compare("2.0.0", "1.0.0"))
}

func TestVersionCacheComparePip(t *testing.T) {
	cache := newVersionCache(types.DependencyTypePip)

	assert.Equal(t, -1, cache.compare("1.0.0", "2.0.0"))
	assert.Equal(t, 0, cache.compare("1.0.0", "1.0.0"))
	assert.Equal(t, 1, cache.compare("2.0.0", "1.0.0"))
}

func TestVersionCacheCompareUnknownType(t *testing.T) {
	cache := newVersionCache("unknown")
	assert.Equal(t, 0, cache.compare("1.0.0", "2.0.0"))
}

func TestVersionCacheCompareInvalidVersion(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	// Invalid version should return 0
	assert.Equal(t, 0, cache.compare("not-valid!!!", "1.0.0"))
}

// ---------------------------------------------------------------------------
// bestCompatibleVersion
// ---------------------------------------------------------------------------

func TestBestCompatibleVersionNoAvailable(t *testing.T) {
	dep := types.Dependency{Name: "libfoo", Type: types.DependencyTypeApt}
	_, err := bestCompatibleVersion(dep, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available versions")
}

func TestBestCompatibleVersionNoConstraints(t *testing.T) {
	dep := types.Dependency{Name: "libfoo", Type: types.DependencyTypeApt}
	version, err := bestCompatibleVersion(dep, []string{"1.0.0", "2.0.0", "0.5.0"})
	require.NoError(t, err)
	// Should pick the highest
	assert.Equal(t, "2.0.0", version)
}

func TestBestCompatibleVersionWithConstraint(t *testing.T) {
	dep := types.Dependency{
		Name: "libfoo",
		Type: types.DependencyTypeApt,
		Constraints: []types.Constraint{
			{Name: "libfoo", Op: types.ConstraintOpLte, Version: "1.5.0"},
		},
	}
	version, err := bestCompatibleVersion(dep, []string{"1.0.0", "1.5.0", "2.0.0"})
	require.NoError(t, err)
	assert.Equal(t, "1.5.0", version)
}

func TestBestCompatibleVersionPinExact(t *testing.T) {
	dep := types.Dependency{
		Name: "libfoo",
		Type: types.DependencyTypeApt,
		Constraints: []types.Constraint{
			{Name: "libfoo", Op: types.ConstraintOpEq, Version: "1.0.0"},
		},
	}
	version, err := bestCompatibleVersion(dep, []string{"1.0.0", "2.0.0"})
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)
}

func TestBestCompatibleVersionNoMatch(t *testing.T) {
	dep := types.Dependency{
		Name: "libfoo",
		Type: types.DependencyTypeApt,
		Constraints: []types.Constraint{
			{Name: "libfoo", Op: types.ConstraintOpGte, Version: "5.0.0"},
		},
	}
	_, err := bestCompatibleVersion(dep, []string{"1.0.0", "2.0.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no compatible version")
}

func TestBestCompatibleVersionPip(t *testing.T) {
	dep := types.Dependency{
		Name: "numpy",
		Type: types.DependencyTypePip,
		Constraints: []types.Constraint{
			{Name: "numpy", Op: types.ConstraintOpGte, Version: "1.20.0"},
		},
	}
	version, err := bestCompatibleVersion(dep, []string{"1.19.0", "1.20.0", "1.26.0"})
	require.NoError(t, err)
	assert.Equal(t, "1.26.0", version)
}

func TestBestCompatibleVersionPipExact(t *testing.T) {
	dep := types.Dependency{
		Name: "flask",
		Type: types.DependencyTypePip,
		Constraints: []types.Constraint{
			{Name: "flask", Op: types.ConstraintOpEq2, Version: "2.3.0"},
		},
	}
	version, err := bestCompatibleVersion(dep, []string{"2.2.0", "2.3.0", "2.4.0"})
	require.NoError(t, err)
	assert.Equal(t, "2.3.0", version)
}

// ---------------------------------------------------------------------------
// satisfiesDeb
// ---------------------------------------------------------------------------

func TestSatisfiesDebGte(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	constraints, err := prepareConstraints(types.DependencyTypeApt, []types.Constraint{
		{Name: "libfoo", Op: types.ConstraintOpGte, Version: "1.0.0"},
	}, cache)
	require.NoError(t, err)

	ok, err := satisfiesDeb("1.0.0", constraints, cache)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = satisfiesDeb("2.0.0", constraints, cache)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = satisfiesDeb("0.9.0", constraints, cache)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSatisfiesDebLt(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	constraints, err := prepareConstraints(types.DependencyTypeApt, []types.Constraint{
		{Name: "libfoo", Op: types.ConstraintOpLt, Version: "2.0.0"},
	}, cache)
	require.NoError(t, err)

	ok, err := satisfiesDeb("1.0.0", constraints, cache)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = satisfiesDeb("2.0.0", constraints, cache)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSatisfiesDebRange(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	constraints, err := prepareConstraints(types.DependencyTypeApt, []types.Constraint{
		{Name: "libfoo", Op: types.ConstraintOpGte, Version: "1.0.0"},
		{Name: "libfoo", Op: types.ConstraintOpLt, Version: "2.0.0"},
	}, cache)
	require.NoError(t, err)

	ok, err := satisfiesDeb("1.5.0", constraints, cache)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = satisfiesDeb("2.0.0", constraints, cache)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = satisfiesDeb("0.9.0", constraints, cache)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSatisfiesDebNoConstraints(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	ok, err := satisfiesDeb("1.0.0", nil, cache)
	require.NoError(t, err)
	assert.True(t, ok)
}

// ---------------------------------------------------------------------------
// satisfiesPep440
// ---------------------------------------------------------------------------

func TestSatisfiesPep440(t *testing.T) {
	cache := newVersionCache(types.DependencyTypePip)
	constraints, err := prepareConstraints(types.DependencyTypePip, []types.Constraint{
		{Name: "numpy", Op: types.ConstraintOpGte, Version: "1.20.0"},
	}, cache)
	require.NoError(t, err)

	ok, err := satisfiesPep440("1.26.0", constraints, cache)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = satisfiesPep440("1.19.0", constraints, cache)
	require.NoError(t, err)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// toPep440Spec
// ---------------------------------------------------------------------------

func TestToPep440Spec(t *testing.T) {
	tests := []struct {
		op      types.ConstraintOp
		version string
		expect  string
	}{
		{types.ConstraintOpGte, "1.0.0", ">= 1.0.0"},
		{types.ConstraintOpLte, "2.0.0", "<= 2.0.0"},
		{types.ConstraintOpEq, "1.5.0", "== 1.5.0"},
		{types.ConstraintOpEq2, "1.5.0", "== 1.5.0"},
		{types.ConstraintOpNe, "1.0.0", "!= 1.0.0"},
		{types.ConstraintOpCompat, "1.2.0", "~= 1.2.0"},
		{types.ConstraintOpGt, "1.0.0", "> 1.0.0"},
		{types.ConstraintOpLt, "2.0.0", "< 2.0.0"},
	}

	for _, tt := range tests {
		t.Run(string(tt.op), func(t *testing.T) {
			constraint := types.Constraint{Op: tt.op, Version: tt.version}
			assert.Equal(t, tt.expect, toPep440Spec(constraint))
		})
	}
}

// ---------------------------------------------------------------------------
// prepareConstraints
// ---------------------------------------------------------------------------

func TestPrepareConstraintsSkipsNone(t *testing.T) {
	cache := newVersionCache(types.DependencyTypeApt)
	constraints, err := prepareConstraints(types.DependencyTypeApt, []types.Constraint{
		{Name: "libfoo", Op: types.ConstraintOpNone},
		{Name: "libfoo", Op: types.ConstraintOpGte, Version: "1.0.0"},
	}, cache)
	require.NoError(t, err)
	assert.Len(t, constraints, 1)
}

func TestPrepareConstraintsUnsupportedType(t *testing.T) {
	cache := newVersionCache("unknown")
	_, err := prepareConstraints("unknown", []types.Constraint{
		{Name: "libfoo", Op: types.ConstraintOpGte, Version: "1.0.0"},
	}, cache)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported dependency type")
}
