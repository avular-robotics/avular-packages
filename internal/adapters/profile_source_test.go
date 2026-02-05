package adapters

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestLoadGitComposeProfile(t *testing.T) {
	repoDir := t.TempDir()
	profilePath := filepath.Join(repoDir, "profiles", "base.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(profilePath), 0755))
	require.NoError(t, os.WriteFile(profilePath, []byte(sampleProfileSpec), 0644))

	runGit(t, repoDir, "init", "--initial-branch=main")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "init")

	product := types.Spec{
		Kind: types.SpecKindProduct,
		Compose: []types.ComposeRef{
			{
				Name:    repoDir,
				Version: "",
				Source:  "git",
				Path:    "profiles/base.yaml",
			},
		},
	}
	specAdapter := NewSpecFileAdapter()
	source := NewProfileSourceAdapter(specAdapter)
	profiles, err := source.LoadProfiles(product, nil)
	require.NoError(t, err)
	if diff := cmp.Diff(1, len(profiles)); diff != "" {
		t.Fatalf("unexpected profile count (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(types.SpecKindProfile, profiles[0].Kind); diff != "" {
		t.Fatalf("unexpected spec kind (-want +got):\n%s", diff)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=avular",
		"GIT_AUTHOR_EMAIL=dev@example.com",
		"GIT_COMMITTER_NAME=avular",
		"GIT_COMMITTER_EMAIL=dev@example.com",
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

const sampleProfileSpec = `api_version: "v1"
kind: "profile"
metadata:
  name: "base-profile"
  version: "2026.01"
  owners: ["platform"]
inputs:
  package_xml:
    enabled: true
    tags: ["debian_depend"]
packaging:
  groups:
    - name: "apt-individual"
      mode: "individual"
      scope: "runtime"
      matches: ["apt:*"]
      targets: ["24.04"]
`
