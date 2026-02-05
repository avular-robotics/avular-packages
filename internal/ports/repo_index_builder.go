package ports

import (
	"context"

	"avular-packages/internal/types"
)

type RepoIndexBuildRequest struct {
	AptSources       []string
	AptEndpoint      string
	AptDistribution  string
	AptComponent     string
	AptArch          string
	AptUser          string
	AptAPIKey        string
	AptWorkers       int
	PipIndex         string
	PipUser          string
	PipAPIKey        string
	PipPackages      []string
	PipMax           int
	PipWorkers       int
	HTTPTimeoutSec   int
	HTTPRetries      int
	HTTPRetryDelayMs int
}

type RepoIndexBuilderPort interface {
	Build(ctx context.Context, request RepoIndexBuildRequest) (types.RepoIndexFile, error)
}

type RepoIndexWriterPort interface {
	Write(path string, index types.RepoIndexFile) error
}
