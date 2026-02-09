package app

import (
	"context"
	"strings"

	"avular-packages/internal/ports"
)

func (s Service) RepoIndex(ctx context.Context, req RepoIndexRequest) (RepoIndexResult, error) {
	buildRequest := ports.RepoIndexBuildRequest{
		AptSources:       req.AptSources,
		AptEndpoint:      strings.TrimSpace(req.AptEndpoint),
		AptDistribution:  strings.TrimSpace(req.AptDistribution),
		AptComponent:     strings.TrimSpace(req.AptComponent),
		AptArch:          strings.TrimSpace(req.AptArch),
		AptUser:          strings.TrimSpace(req.AptUser),
		AptAPIKey:        strings.TrimSpace(req.AptAPIKey),
		AptWorkers:       req.AptWorkers,
		PipIndex:         strings.TrimSpace(req.PipIndex),
		PipUser:          strings.TrimSpace(req.PipUser),
		PipAPIKey:        strings.TrimSpace(req.PipAPIKey),
		PipPackages:      req.PipPackages,
		PipMax:           req.PipMax,
		PipWorkers:       req.PipWorkers,
		HTTPTimeoutSec:   req.HTTPTimeoutSec,
		HTTPRetries:      req.HTTPRetries,
		HTTPRetryDelayMs: req.HTTPRetryDelayMs,
		CacheDir:         strings.TrimSpace(req.CacheDir),
		CacheTTLMinutes:  req.CacheTTLMinutes,
	}
	index, err := s.RepoIndexBuild.Build(ctx, buildRequest)
	if err != nil {
		return RepoIndexResult{}, err
	}
	if err := s.RepoIndexWriter.Write(strings.TrimSpace(req.Output), index); err != nil {
		return RepoIndexResult{}, err
	}
	return RepoIndexResult{
		OutputPath: strings.TrimSpace(req.Output),
		AptCount:   len(index.Apt),
		PipCount:   len(index.Pip),
	}, nil
}
