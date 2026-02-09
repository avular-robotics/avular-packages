package adapters

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
	"avular-packages/internal/shared"
	"avular-packages/internal/types"
)

type ProfileSourceAdapter struct {
	Spec SpecFileAdapter
}

func NewProfileSourceAdapter(spec SpecFileAdapter) ProfileSourceAdapter {
	return ProfileSourceAdapter{Spec: spec}
}

func (a ProfileSourceAdapter) LoadProfiles(product types.Spec, explicit []string) ([]types.Spec, error) {
	if len(explicit) > 0 {
		return a.loadProfilePaths(explicit)
	}
	var profiles []types.Spec
	for _, compose := range product.Compose {
		spec, err := a.loadComposeProfile(compose)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, spec)
	}
	return profiles, nil
}

func (a ProfileSourceAdapter) loadProfilePaths(paths []string) ([]types.Spec, error) {
	var profiles []types.Spec
	for _, path := range paths {
		spec, err := a.Spec.LoadProfile(path)
		if err != nil {
			return nil, err
		}
		enrichProfileSchemas(&spec, path)
		profiles = append(profiles, spec)
	}
	return profiles, nil
}

func (a ProfileSourceAdapter) loadComposeProfile(compose types.ComposeRef) (types.Spec, error) {
	switch strings.ToLower(strings.TrimSpace(compose.Source)) {
	case "local":
		if strings.TrimSpace(compose.Path) == "" {
			return types.Spec{}, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("compose path is required for local sources")
		}
		spec, err := a.Spec.LoadProfile(compose.Path)
		if err != nil {
			return types.Spec{}, err
		}
		enrichProfileSchemas(&spec, compose.Path)
		return spec, nil
	case "git":
		return a.loadGitProfile(compose)
	case "inline":
		return a.loadInlineProfile(compose)
	default:
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg(fmt.Sprintf("unsupported compose source: %s", compose.Source))
	}
}

func (a ProfileSourceAdapter) loadInlineProfile(compose types.ComposeRef) (types.Spec, error) {
	if compose.Profile == nil {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("compose source is 'inline' but no profile definition provided")
	}

	name := strings.TrimSpace(compose.Name)
	if name == "" {
		name = "inline"
	}

	version := strings.TrimSpace(compose.Version)
	if version == "" {
		version = "0.0.0"
	}

	return types.Spec{
		APIVersion: "v1",
		Kind:       types.SpecKindProfile,
		Metadata: types.Metadata{
			Name:    name,
			Version: version,
		},
		Inputs:      compose.Profile.Inputs,
		Packaging:   compose.Profile.Packaging,
		Resolutions: compose.Profile.Resolutions,
	}, nil
}

func (a ProfileSourceAdapter) loadGitProfile(compose types.ComposeRef) (types.Spec, error) {
	repo := strings.TrimSpace(compose.Name)
	if repo == "" {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("compose name must be git repository URL for git sources")
	}
	if strings.TrimSpace(compose.Path) == "" {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("compose path is required for git sources")
	}
	tempDir, err := os.MkdirTemp("", "avular-packages-compose-")
	if err != nil {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create temp directory for compose").
			WithCause(err)
	}
	defer os.RemoveAll(tempDir)

	args := []string{"clone", "--depth", "1"}
	if strings.TrimSpace(compose.Version) != "" {
		args = append(args, "--branch", compose.Version)
	}
	args = append(args, repo, tempDir)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return types.Spec{}, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to clone git compose source").
			WithCause(shared.CommandError(output, err))
	}

	specPath := filepath.Join(tempDir, compose.Path)
	spec, err := a.Spec.LoadProfile(specPath)
	if err != nil {
		return types.Spec{}, err
	}
	enrichProfileSchemas(&spec, specPath)
	return spec, nil
}

// enrichProfileSchemas discovers schema files in a schemas/ directory
// next to the profile spec file and prepends them to the profile's
// schema_files list.  This gives profile-adjacent schemas lower
// precedence than the profile's own explicit schema_files.
func enrichProfileSchemas(spec *types.Spec, profilePath string) {
	discovered := discoverProfileSchemaFiles(profilePath)
	if len(discovered) == 0 {
		return
	}
	// Prepend discovered schemas so the profile's explicit schema_files
	// (if any) take higher precedence.
	spec.Inputs.PackageXML.SchemaFiles = append(
		discovered,
		spec.Inputs.PackageXML.SchemaFiles...,
	)
}

// discoverProfileSchemaFiles looks for a schemas/ directory next to
// the given profile spec file and returns sorted YAML paths.
func discoverProfileSchemaFiles(profilePath string) []string {
	if profilePath == "" {
		return nil
	}
	dir := filepath.Dir(profilePath)
	schemasDir := filepath.Join(dir, "schemas")
	info, err := os.Stat(schemasDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			paths = append(paths, filepath.Join(schemasDir, name))
		}
	}
	sort.Strings(paths)
	return paths
}

var _ ports.ProfileSourcePort = ProfileSourceAdapter{}
