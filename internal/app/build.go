package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/adapters"
	"avular-packages/internal/types"
)

func (s Service) Build(ctx context.Context, req BuildRequest) (BuildResult, error) {
	// Determine whether the resolve phase should run.
	// With auto-discovery and spec defaults the user no longer needs to
	// supply --product, --repo-index, and --target-ubuntu explicitly;
	// the product spec can provide all of them.
	productPath := strings.TrimSpace(req.ProductPath)
	if productPath == "" {
		productPath = discoverProduct()
	}

	// If we found a product, load it to apply build-specific defaults
	// before evaluating outputDir and other fields.
	if productPath != "" {
		product, err := s.SpecLoader.LoadProduct(productPath)
		if err == nil {
			emitHints(checkBuildDefaultsHints(req, product.Defaults))
			req = applyBuildDefaults(req, product.Defaults)
		}
	}

	outputDir := strings.TrimSpace(req.OutputDir)
	if outputDir == "" {
		return BuildResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("output directory is required")
	}

	resolveNeeded := productPath != "" ||
		strings.TrimSpace(req.RepoIndex) != "" ||
		strings.TrimSpace(req.TargetUbuntu) != ""

	if resolveNeeded {
		_, err := s.Resolve(ctx, ResolveRequest{
			ProductPath:          productPath,
			Profiles:             req.Profiles,
			Workspace:            req.Workspace,
			RepoIndex:            req.RepoIndex,
			OutputDir:            outputDir,
			TargetUbuntu:         req.TargetUbuntu,
			SchemaFiles:          req.SchemaFiles,
			CompatGet:            true,
			EmitAptPreferences:   req.EmitAptPreferences,
			EmitAptInstallList:   req.EmitAptInstallList,
			EmitSnapshotSources:  req.EmitSnapshotSources,
			SnapshotAptBaseURL:   req.SnapshotAptBaseURL,
			SnapshotAptComponent: req.SnapshotAptComponent,
			SnapshotAptArchs:     req.SnapshotAptArchs,
			AptSatSolver:         req.AptSatSolver,
		})
		if err != nil {
			return BuildResult{}, err
		}
	}
	debsDir := strings.TrimSpace(req.DebsDir)
	if debsDir == "" {
		debsDir = filepath.Join(outputDir, "debs")
	}
	if err := os.MkdirAll(debsDir, 0o750); err != nil {
		return BuildResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create debs directory").
			WithCause(err)
	}

	if internalDir := strings.TrimSpace(req.InternalDebDir); internalDir != "" {
		if err := s.InternalDebs.CopyDebs(internalDir, debsDir); err != nil {
			return BuildResult{}, err
		}
	}
	if len(req.InternalSrc) > 0 {
		if err := s.InternalDebs.BuildInternalDebs(req.InternalSrc, debsDir); err != nil {
			return BuildResult{}, err
		}
	}

	builder := adapters.NewPackageBuildAdapter(strings.TrimSpace(req.PipIndexURL))
	if err := builder.BuildDebs(outputDir, debsDir); err != nil {
		return BuildResult{}, err
	}
	return BuildResult{DebsDir: debsDir}, nil
}

// applyBuildDefaults fills in BuildRequest fields from the product
// spec's defaults section when the request field is empty.  It covers
// both the shared resolve fields and build-specific ones.
func applyBuildDefaults(req BuildRequest, defaults types.SpecDefaults) BuildRequest {
	if strings.TrimSpace(req.TargetUbuntu) == "" && defaults.TargetUbuntu != "" {
		req.TargetUbuntu = defaults.TargetUbuntu
	}
	if len(req.Workspace) == 0 && len(defaults.Workspace) > 0 {
		req.Workspace = defaults.Workspace
	}
	if strings.TrimSpace(req.RepoIndex) == "" && defaults.RepoIndex != "" {
		req.RepoIndex = defaults.RepoIndex
	}
	if strings.TrimSpace(req.OutputDir) == "" && defaults.Output != "" {
		req.OutputDir = defaults.Output
	}
	if strings.TrimSpace(req.PipIndexURL) == "" && defaults.PipIndexURL != "" {
		req.PipIndexURL = defaults.PipIndexURL
	}
	if strings.TrimSpace(req.InternalDebDir) == "" && defaults.InternalDebDir != "" {
		req.InternalDebDir = defaults.InternalDebDir
	}
	if len(req.InternalSrc) == 0 && len(defaults.InternalSrc) > 0 {
		req.InternalSrc = defaults.InternalSrc
	}
	return req
}
