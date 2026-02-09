package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/adapters"
)

func (s Service) Build(ctx context.Context, req BuildRequest) (BuildResult, error) {
	outputDir := strings.TrimSpace(req.OutputDir)
	if outputDir == "" {
		return BuildResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("output directory is required")
	}

	// Determine whether the resolve phase should run.
	// With auto-discovery and spec defaults the user no longer needs to
	// supply --product, --repo-index, and --target-ubuntu explicitly;
	// the product spec can provide all of them.
	productPath := strings.TrimSpace(req.ProductPath)
	if productPath == "" {
		productPath = discoverProduct()
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
	if err := os.MkdirAll(debsDir, 0755); err != nil {
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
