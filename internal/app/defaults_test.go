package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"avular-packages/internal/types"
)

func TestApplySpecDefaults(t *testing.T) {
	defaults := types.SpecDefaults{
		TargetUbuntu: "24.04",
		Workspace:    []string{"./src"},
		RepoIndex:    "./repo-index.yaml",
		Output:       "build-out",
	}

	tests := []struct {
		name     string
		req      ResolveRequest
		expected ResolveRequest
	}{
		{
			name: "empty request gets all defaults",
			req:  ResolveRequest{},
			expected: ResolveRequest{
				TargetUbuntu: "24.04",
				Workspace:    []string{"./src"},
				RepoIndex:    "./repo-index.yaml",
				OutputDir:    "build-out",
			},
		},
		{
			name: "explicit values override defaults",
			req: ResolveRequest{
				TargetUbuntu: "22.04",
				Workspace:    []string{"/custom/ws"},
				RepoIndex:    "/custom/repo.yaml",
				OutputDir:    "/custom/out",
			},
			expected: ResolveRequest{
				TargetUbuntu: "22.04",
				Workspace:    []string{"/custom/ws"},
				RepoIndex:    "/custom/repo.yaml",
				OutputDir:    "/custom/out",
			},
		},
		{
			name: "partial override mixes with defaults",
			req: ResolveRequest{
				TargetUbuntu: "22.04",
				// Workspace, RepoIndex, OutputDir left empty -> defaults apply
			},
			expected: ResolveRequest{
				TargetUbuntu: "22.04",
				Workspace:    []string{"./src"},
				RepoIndex:    "./repo-index.yaml",
				OutputDir:    "build-out",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := applySpecDefaults(tc.req, defaults)
			assert.Equal(t, tc.expected.TargetUbuntu, result.TargetUbuntu)
			assert.Equal(t, tc.expected.Workspace, result.Workspace)
			assert.Equal(t, tc.expected.RepoIndex, result.RepoIndex)
			assert.Equal(t, tc.expected.OutputDir, result.OutputDir)
		})
	}
}

func TestApplySpecDefaultsEmpty(t *testing.T) {
	// Empty defaults should not change anything
	req := ResolveRequest{
		TargetUbuntu: "22.04",
		Workspace:    []string{"/ws"},
	}

	result := applySpecDefaults(req, types.SpecDefaults{})
	assert.Equal(t, "22.04", result.TargetUbuntu)
	assert.Equal(t, []string{"/ws"}, result.Workspace)
	assert.Empty(t, result.RepoIndex)
	assert.Empty(t, result.OutputDir)
}

func TestDiscoverProduct(t *testing.T) {
	// discoverProduct looks in current directory; in test context
	// there's no product.yaml so it should return empty
	result := discoverProduct()
	assert.Empty(t, result)
}
