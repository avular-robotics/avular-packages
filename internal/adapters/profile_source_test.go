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

func TestLoadInlineComposeProfile(t *testing.T) {
	product := types.Spec{
		Kind: types.SpecKindProduct,
		Compose: []types.ComposeRef{
			{
				Name:    "inline-profile",
				Version: "1.0.0",
				Source:  "inline",
				Profile: &types.InlineProfile{
					Inputs: types.Inputs{
						PackageXML: types.PackageXMLInput{
							Enabled: true,
							Tags:    []string{"debian_depend", "pip_depend"},
						},
					},
					Packaging: types.Packaging{
						Groups: []types.PackagingGroup{
							{
								Name:    "apt-individual",
								Mode:    types.PackagingModeIndividual,
								Scope:   "runtime",
								Matches: []string{"apt:*"},
								Targets: []string{"24.04"},
							},
						},
					},
				},
			},
		},
	}

	specAdapter := NewSpecFileAdapter()
	source := NewProfileSourceAdapter(specAdapter)
	profiles, err := source.LoadProfiles(product, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 1)

	p := profiles[0]
	require.Equal(t, types.SpecKindProfile, p.Kind)
	require.Equal(t, "inline-profile", p.Metadata.Name)
	require.Equal(t, "1.0.0", p.Metadata.Version)
	require.True(t, p.Inputs.PackageXML.Enabled)
	require.Len(t, p.Packaging.Groups, 1)
	require.Equal(t, "apt-individual", p.Packaging.Groups[0].Name)
}

func TestLoadInlineProfileMissingDefinition(t *testing.T) {
	product := types.Spec{
		Kind: types.SpecKindProduct,
		Compose: []types.ComposeRef{
			{
				Name:    "broken",
				Source:  "inline",
				Profile: nil, // Missing!
			},
		},
	}

	specAdapter := NewSpecFileAdapter()
	source := NewProfileSourceAdapter(specAdapter)
	_, err := source.LoadProfiles(product, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no profile definition provided")
}

func TestLoadMixedComposeProfiles(t *testing.T) {
	// Mix of local file + inline profile
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "local.yaml")
	require.NoError(t, os.WriteFile(profilePath, []byte(sampleProfileSpec), 0644))

	product := types.Spec{
		Kind: types.SpecKindProduct,
		Compose: []types.ComposeRef{
			{
				Name:   "file-profile",
				Source: "local",
				Path:   profilePath,
			},
			{
				Name:   "inline-profile",
				Source: "inline",
				Profile: &types.InlineProfile{
					Inputs: types.Inputs{
						PackageXML: types.PackageXMLInput{Enabled: false},
					},
					Packaging: types.Packaging{
						Groups: []types.PackagingGroup{
							{
								Name:    "pip-meta",
								Mode:    types.PackagingModeMetaBundle,
								Scope:   "runtime",
								Matches: []string{"pip:*"},
								Targets: []string{"24.04"},
							},
						},
					},
				},
			},
		},
	}

	specAdapter := NewSpecFileAdapter()
	source := NewProfileSourceAdapter(specAdapter)
	profiles, err := source.LoadProfiles(product, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 2)

	// First is from file
	require.Equal(t, "base-profile", profiles[0].Metadata.Name)
	// Second is inline
	require.Equal(t, "inline-profile", profiles[1].Metadata.Name)
	require.Len(t, profiles[1].Packaging.Groups, 1)
	require.Equal(t, "pip-meta", profiles[1].Packaging.Groups[0].Name)
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
