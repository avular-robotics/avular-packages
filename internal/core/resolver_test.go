package core

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/policies"
	"avular-packages/internal/types"
)

type testRepoIndex struct {
	apt map[string][]string
	pip map[string][]string
}

func (t testRepoIndex) AvailableVersions(depType types.DependencyType, name string) ([]string, error) {
	switch depType {
	case types.DependencyTypeApt:
		return t.apt[name], nil
	case types.DependencyTypePip:
		return t.pip[name], nil
	default:
		return nil, nil
	}
}

func TestResolverBestCompatible(t *testing.T) {
	repo := testRepoIndex{
		apt: map[string][]string{
			"libfoo": {"1.0.0", "1.2.0", "2.0.0"},
		},
		pip: map[string][]string{
			"pandas": {"2.1.4", "2.1.3"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-group", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"ubuntu-22.04"}},
		{Name: "pip-group", Mode: types.PackagingModeMetaBundle, Matches: []string{"pip:*"}, Targets: []string{"ubuntu-22.04"}},
	}, "ubuntu-22.04")

	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpGte, Version: "1.0.0"},
			},
		},
		{
			Name: "pandas",
			Type: types.DependencyTypePip,
			Constraints: []types.Constraint{
				{Name: "pandas", Op: types.ConstraintOpEq2, Version: "2.1.4"},
			},
		},
	}

	result, err := resolver.Resolve(t.Context(), deps, nil)
	require.NoError(t, err)
	if diff := cmp.Diff(2, len(result.AptLocks)); diff != "" {
		t.Fatalf("unexpected apt locks count (-want +got):\n%s", diff)
	}
}

func TestResolverConflictRequiresDirective(t *testing.T) {
	repo := testRepoIndex{
		apt: map[string][]string{
			"libfoo": {"1.0.0", "1.2.0"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-group", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"ubuntu-22.04"}},
	}, "ubuntu-22.04")
	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpGte, Version: "2.0.0"},
				{Name: "libfoo", Op: types.ConstraintOpLt, Version: "2.0.0"},
			},
		},
	}
	_, err := resolver.Resolve(t.Context(), deps, nil)
	require.Error(t, err)
}

func TestResolverConflictWithDirective(t *testing.T) {
	repo := testRepoIndex{
		apt: map[string][]string{
			"libfoo": {"1.0.0", "1.2.0"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-group", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"ubuntu-22.04"}},
	}, "ubuntu-22.04")
	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpGte, Version: "2.0.0"},
				{Name: "libfoo", Op: types.ConstraintOpLt, Version: "2.0.0"},
			},
		},
	}
	directives := []types.ResolutionDirective{
		{Dependency: "apt:libfoo", Action: "force", Value: "1.2.0", Reason: "test", Owner: "test"},
	}
	result, err := resolver.Resolve(t.Context(), deps, directives)
	require.NoError(t, err)
	if diff := cmp.Diff(1, len(result.Resolution.Records)); diff != "" {
		t.Fatalf("unexpected resolution record count (-want +got):\n%s", diff)
	}
}

func TestResolverAppliesProductPriorityOverProfileAndPackageXML(t *testing.T) {
	repo := testRepoIndex{
		apt: map[string][]string{
			"libfoo": {"1.0.0", "1.2.0"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-group", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"ubuntu-22.04"}},
	}, "ubuntu-22.04")
	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpEq, Version: "1.2.0", Source: "product:manual:apt"},
			},
		},
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpEq, Version: "1.0.0", Source: "profile:manual:apt"},
			},
		},
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpGte, Version: "0.5.0", Source: "package_xml:debian_depend"},
			},
		},
	}

	result, err := resolver.Resolve(t.Context(), deps, nil)
	require.NoError(t, err)
	if diff := cmp.Diff(1, len(result.AptLocks)); diff != "" {
		t.Fatalf("unexpected apt locks count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("1.2.0", result.AptLocks[0].Version); diff != "" {
		t.Fatalf("unexpected version (-want +got):\n%s", diff)
	}
}

func TestResolverAppliesProfilePriorityOverPackageXML(t *testing.T) {
	repo := testRepoIndex{
		apt: map[string][]string{
			"libfoo": {"1.0.0", "1.2.0"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-group", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"ubuntu-22.04"}},
	}, "ubuntu-22.04")
	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpEq, Version: "1.0.0", Source: "profile:manual:apt"},
			},
		},
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpEq, Version: "1.2.0", Source: "package_xml:debian_depend"},
			},
		},
	}

	result, err := resolver.Resolve(t.Context(), deps, nil)
	require.NoError(t, err)
	if diff := cmp.Diff(1, len(result.AptLocks)); diff != "" {
		t.Fatalf("unexpected apt locks count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("1.0.0", result.AptLocks[0].Version); diff != "" {
		t.Fatalf("unexpected version (-want +got):\n%s", diff)
	}
}

func TestResolverAppliesPackagingGroupPins(t *testing.T) {
	repo := testRepoIndex{
		apt: map[string][]string{
			"libfoo": {"1.0.0", "1.1.0"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{
			Name:    "apt-group",
			Mode:    types.PackagingModeIndividual,
			Matches: []string{"apt:*"},
			Targets: []string{"ubuntu-22.04"},
			Pins:    []string{"libfoo=1.0.0"},
		},
	}, "ubuntu-22.04")
	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpGte, Version: "1.0.0", Source: "package_xml:debian_depend"},
			},
		},
	}

	result, err := resolver.Resolve(t.Context(), deps, nil)
	require.NoError(t, err)
	if diff := cmp.Diff(1, len(result.AptLocks)); diff != "" {
		t.Fatalf("unexpected apt locks count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("1.0.0", result.AptLocks[0].Version); diff != "" {
		t.Fatalf("unexpected version (-want +got):\n%s", diff)
	}
}

func TestResolverFallsBackToLowerPriorityConstraints(t *testing.T) {
	repo := testRepoIndex{
		apt: map[string][]string{
			"libfoo": {"1.0.0", "2.0.0"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-group", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"ubuntu-22.04"}},
	}, "ubuntu-22.04")
	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpNone, Source: "product:manual:apt"},
			},
		},
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpLte, Version: "1.0.0", Source: "profile:manual:apt"},
			},
		},
	}

	result, err := resolver.Resolve(t.Context(), deps, nil)
	require.NoError(t, err)
	if diff := cmp.Diff(1, len(result.AptLocks)); diff != "" {
		t.Fatalf("unexpected apt locks count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("1.0.0", result.AptLocks[0].Version); diff != "" {
		t.Fatalf("unexpected version (-want +got):\n%s", diff)
	}
}

func TestResolverNormalizesPipDirectiveKey(t *testing.T) {
	repo := testRepoIndex{
		pip: map[string][]string{
			"requests": {"2.0.0"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{Name: "pip-group", Mode: types.PackagingModeMetaBundle, Matches: []string{"pip:*"}, Targets: []string{"ubuntu-22.04"}},
	}, "ubuntu-22.04")
	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "requests",
			Type: types.DependencyTypePip,
			Constraints: []types.Constraint{
				{Name: "requests", Op: types.ConstraintOpGte, Version: "3.0.0"},
			},
		},
	}
	directives := []types.ResolutionDirective{
		{Dependency: "pip:Requests", Action: "force", Value: "2.0.0", Reason: "test", Owner: "test"},
	}

	result, err := resolver.Resolve(t.Context(), deps, directives)
	require.NoError(t, err)
	if diff := cmp.Diff(1, len(result.Resolution.Records)); diff != "" {
		t.Fatalf("unexpected resolution record count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("2.0.0", result.AptLocks[0].Version); diff != "" {
		t.Fatalf("unexpected version (-want +got):\n%s", diff)
	}
}

func TestResolverFallsBackWhenHigherPriorityIsUnconstrained(t *testing.T) {
	repo := testRepoIndex{
		apt: map[string][]string{
			"libfoo": {"1.0.0", "1.2.0", "2.0.0"},
		},
	}
	policy := policies.NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-group", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"ubuntu-22.04"}},
	}, "ubuntu-22.04")
	resolver := NewResolverCore(repo, policy)

	deps := []types.Dependency{
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpNone, Source: "product:manual:apt"},
			},
		},
		{
			Name: "libfoo",
			Type: types.DependencyTypeApt,
			Constraints: []types.Constraint{
				{Name: "libfoo", Op: types.ConstraintOpLte, Version: "1.2.0", Source: "package_xml:debian_depend"},
			},
		},
	}

	result, err := resolver.Resolve(t.Context(), deps, nil)
	require.NoError(t, err)
	if diff := cmp.Diff(1, len(result.AptLocks)); diff != "" {
		t.Fatalf("unexpected apt locks count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("1.2.0", result.AptLocks[0].Version); diff != "" {
		t.Fatalf("unexpected version (-want +got):\n%s", diff)
	}
}
