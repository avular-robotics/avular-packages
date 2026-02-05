package app

import "avular-packages/internal/types"

type ValidateRequest struct {
	ProductPath string
	Profiles    []string
}

type ValidateResult struct {
	ProductName string
}

type ResolveRequest struct {
	ProductPath  string
	Profiles     []string
	Workspace    []string
	RepoIndex    string
	OutputDir    string
	SnapshotID   string
	TargetUbuntu string
	CompatGet    bool
	CompatRosdep bool
}

type ResolveResult struct {
	ProductName string
	SnapshotID  string
	OutputDir   string
}

type BuildRequest struct {
	ProductPath    string
	Profiles       []string
	Workspace      []string
	RepoIndex      string
	OutputDir      string
	DebsDir        string
	TargetUbuntu   string
	PipIndexURL    string
	InternalDebDir string
	InternalSrc    []string
}

type BuildResult struct {
	DebsDir string
}

type PublishRequest struct {
	OutputDir          string
	RepoDir            string
	SBOM               bool
	RepoBackend        string
	DebsDir            string
	AptlyRepo          string
	AptlyComponent     string
	AptlyPrefix        string
	AptlyEndpoint      string
	GpgKey             string
	ProGetEndpoint     string
	ProGetFeed         string
	ProGetComponent    string
	ProGetUser         string
	ProGetAPIKey       string
	ProGetWorkers      int
	ProGetTimeoutSec   int
	ProGetRetries      int
	ProGetRetryDelayMs int
}

type PublishResult struct {
	SnapshotID string
}

type RepoIndexRequest struct {
	Output           string
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

type RepoIndexResult struct {
	OutputPath string
	AptCount   int
	PipCount   int
}

type InspectRequest struct {
	OutputDir string
}

type InspectGroupSummary struct {
	Name     string
	Mode     types.PackagingMode
	Count    int
	Packages []string
}

type InspectResult struct {
	AptLockCount      int
	Groups            []InspectGroupSummary
	ResolutionRecords []types.ResolutionRecord
}
