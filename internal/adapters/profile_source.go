package adapters

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
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
		return a.Spec.LoadProfile(compose.Path)
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
			WithCause(fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err))
	}

	specPath := filepath.Join(tempDir, compose.Path)
	return a.Spec.LoadProfile(specPath)
}

var _ ports.ProfileSourcePort = ProfileSourceAdapter{}
