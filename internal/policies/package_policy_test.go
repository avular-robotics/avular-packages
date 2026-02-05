package policies

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestPackagingPolicyMatchesByTarget(t *testing.T) {
	policy := NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-22", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"22.04"}},
		{Name: "apt-24", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"24.04"}},
	}, "24.04")

	group, err := policy.ResolvePackagingMode(types.DependencyTypeApt, "libfoo")
	require.NoError(t, err)
	if diff := cmp.Diff("apt-24", group.Name); diff != "" {
		t.Fatalf("unexpected group name (-want +got):\n%s", diff)
	}
}

func TestPackagingPolicyMatchesPattern(t *testing.T) {
	policy := NewPackagingPolicy([]types.PackagingGroup{
		{Name: "pip-group", Mode: types.PackagingModeMetaBundle, Matches: []string{"pip:requests*"}, Targets: []string{"24.04"}},
	}, "24.04")

	group, err := policy.ResolvePackagingMode(types.DependencyTypePip, "requests-oauth")
	require.NoError(t, err)
	if diff := cmp.Diff("pip-group", group.Name); diff != "" {
		t.Fatalf("unexpected group name (-want +got):\n%s", diff)
	}
}

func TestPackagingPolicyMatchesUbuntuPrefixedTarget(t *testing.T) {
	policy := NewPackagingPolicy([]types.PackagingGroup{
		{Name: "apt-24", Mode: types.PackagingModeIndividual, Matches: []string{"apt:*"}, Targets: []string{"ubuntu-24.04"}},
	}, "24.04")

	group, err := policy.ResolvePackagingMode(types.DependencyTypeApt, "libfoo")
	require.NoError(t, err)
	if diff := cmp.Diff("apt-24", group.Name); diff != "" {
		t.Fatalf("unexpected group name (-want +got):\n%s", diff)
	}
}
