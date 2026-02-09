// Package testutil provides shared test helpers used across integration,
// e2e, and unit test packages.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// RepoRoot returns the absolute path to the repository root by walking
// up from the current working directory. It fails the test if the
// working directory cannot be determined.
func RepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
