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
	productPath := strings.TrimSpace(req.ProductPath)
	repoIndex := strings.TrimSpace(req.RepoIndex)
	targetUbuntu := strings.TrimSpace(req.TargetUbuntu)
	if productPath != "" || repoIndex != "" || targetUbuntu != "" {
		if productPath == "" || repoIndex == "" || targetUbuntu == "" {
			return BuildResult{}, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("product, repo_index, and target_ubuntu are required when running resolve in build")
		}
		_, err := s.Resolve(ctx, ResolveRequest{
			ProductPath:  productPath,
			Profiles:     req.Profiles,
			Workspace:    req.Workspace,
			RepoIndex:    repoIndex,
			OutputDir:    outputDir,
			TargetUbuntu: targetUbuntu,
			CompatGet:    true,
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
