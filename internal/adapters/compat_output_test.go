package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestWriteGetDependenciesOutputs(t *testing.T) {
	dir := t.TempDir()
	resolved := []types.ResolvedDependency{
		{Type: types.DependencyTypeApt, Package: "libfoo", Version: "1.2.0"},
		{Type: types.DependencyTypePip, Package: "requests", Version: "2.32.0"},
	}

	compat := NewCompatibilityOutputAdapter(dir)
	require.NoError(t, compat.WriteGetDependencies(resolved))

	checks := []struct {
		name      string
		filename  string
		substring string
	}{
		{name: "apt output", filename: "get-dependencies.apt", substring: "libfoo=1.2.0"},
		{name: "pip output", filename: "get-dependencies.pip", substring: "requests==2.32.0"},
	}

	for _, tt := range checks {
		content, err := os.ReadFile(filepath.Join(dir, tt.filename))
		require.NoError(t, err)
		if diff := cmp.Diff(true, strings.Contains(string(content), tt.substring)); diff != "" {
			t.Fatalf("unexpected %s content (-want +got):\n%s", tt.name, diff)
		}
	}
}

func TestWriteRosdepMappingOutput(t *testing.T) {
	dir := t.TempDir()
	resolved := []types.ResolvedDependency{
		{Type: types.DependencyTypeApt, Package: "libbar", Version: "2.0.0"},
		{Type: types.DependencyTypePip, Package: "requests", Version: "2.32.0"},
	}

	compat := NewCompatibilityOutputAdapter(dir)
	require.NoError(t, compat.WriteRosdepMapping(resolved))

	checks := []struct {
		name      string
		substring string
	}{
		{name: "rosdep mapping package", substring: "libbar"},
		{name: "rosdep mapping version", substring: "libbar=2.0.0"},
	}
	content, err := os.ReadFile(filepath.Join(dir, "rosdep-mapping.yaml"))
	require.NoError(t, err)
	for _, tt := range checks {
		if diff := cmp.Diff(true, strings.Contains(string(content), tt.substring)); diff != "" {
			t.Fatalf("unexpected %s content (-want +got):\n%s", tt.name, diff)
		}
	}
}
