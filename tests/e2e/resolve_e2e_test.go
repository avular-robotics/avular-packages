package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCommandE2E(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	cmd := exec.Command("go", "run", "./cmd/avular-packages", "resolve",
		"--product", "fixtures/product-sample.yaml",
		"--repo-index", "fixtures/repo-index.yaml",
		"--output", outDir,
		"--target-ubuntu", "24.04",
		"--workspace", "fixtures/workspace",
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	require.FileExists(t, filepath.Join(outDir, "apt.lock"))
	require.FileExists(t, filepath.Join(outDir, "bundle.manifest"))
	require.FileExists(t, filepath.Join(outDir, "snapshot.intent"))
	require.FileExists(t, filepath.Join(outDir, "resolution.report"))
}

func repoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
