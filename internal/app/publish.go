package app

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/adapters"
	"avular-packages/internal/types"
)

// Publish creates a repository snapshot using the configured backend
// (file, aptly, or proget), optionally generates an SBOM, and returns
// the snapshot identifier.
func (s Service) Publish(ctx context.Context, req PublishRequest) (PublishResult, error) {
	outputDir := strings.TrimSpace(req.OutputDir)
	if outputDir == "" {
		return PublishResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("output directory is required")
	}
	repoDir := strings.TrimSpace(req.RepoDir)
	if repoDir == "" {
		repoDir = filepath.Join(outputDir, "repo")
	}
	intent, err := s.OutputReader.ReadSnapshotIntent(filepath.Join(outputDir, "snapshot.intent"))
	if err != nil {
		return PublishResult{}, err
	}
	repoBackend := strings.ToLower(strings.TrimSpace(req.RepoBackend))
	if repoBackend == "" {
		repoBackend = "file"
	}

	switch repoBackend {
	case "file":
		if err := publishFile(ctx, repoDir, intent); err != nil {
			return PublishResult{}, err
		}
	case "aptly":
		if err := publishAptly(ctx, outputDir, req, intent); err != nil {
			return PublishResult{}, err
		}
	case "proget":
		if err := publishProGet(ctx, outputDir, req, intent); err != nil {
			return PublishResult{}, err
		}
	default:
		return PublishResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("unsupported repo backend")
	}

	if req.SBOM {
		locks, err := s.OutputReader.ReadAptLock(filepath.Join(outputDir, "apt.lock"))
		if err != nil {
			return PublishResult{}, err
		}
		if err := s.SBOMWriter.WriteSBOM(repoDir, intent.SnapshotID, intent.CreatedAt, locks); err != nil {
			return PublishResult{}, err
		}
	}
	return PublishResult{SnapshotID: intent.SnapshotID}, nil
}

// publishFile creates a file-backed snapshot and promotes it to a
// channel if one is configured.
func publishFile(ctx context.Context, repoDir string, intent types.SnapshotIntent) error {
	adapter := adapters.NewRepoSnapshotFileAdapter(repoDir)
	if err := adapter.Publish(ctx, intent.SnapshotID); err != nil {
		return err
	}
	if strings.TrimSpace(intent.Channel) != "" {
		return adapter.Promote(ctx, intent.SnapshotID, intent.Channel)
	}
	return nil
}

// publishAptly creates a snapshot via the Aptly CLI adapter, uploading
// debs and publishing to a prefix/endpoint with GPG signing.
func publishAptly(ctx context.Context, outputDir string, req PublishRequest, intent types.SnapshotIntent) error {
	debsDir := strings.TrimSpace(req.DebsDir)
	if debsDir == "" {
		debsDir = filepath.Join(outputDir, "debs")
	}
	repoName := strings.TrimSpace(req.AptlyRepo)
	if repoName == "" {
		repoName = intent.Repository
	}
	component := strings.TrimSpace(req.AptlyComponent)
	prefix := strings.TrimSpace(req.AptlyPrefix)
	endpoint := strings.TrimSpace(req.AptlyEndpoint)
	gpgKey := strings.TrimSpace(req.GpgKey)
	if gpgKey == "" {
		gpgKey = intent.SigningKey
	}
	if strings.TrimSpace(gpgKey) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("gpg key is required for aptly backend")
	}

	adapter := adapters.NewRepoSnapshotAptlyAdapter(repoName, intent.Channel, component, debsDir, prefix, endpoint, gpgKey)
	if err := adapter.Publish(ctx, intent.SnapshotID); err != nil {
		return err
	}
	return adapter.Promote(ctx, intent.SnapshotID, intent.Channel)
}

// publishProGet creates a snapshot via the ProGet HTTP API adapter,
// uploading debs and optionally promoting to a channel.
func publishProGet(ctx context.Context, outputDir string, req PublishRequest, intent types.SnapshotIntent) error {
	debsDir := strings.TrimSpace(req.DebsDir)
	if debsDir == "" {
		debsDir = filepath.Join(outputDir, "debs")
	}
	endpoint := strings.TrimSpace(req.ProGetEndpoint)
	feed := strings.TrimSpace(req.ProGetFeed)
	if feed == "" {
		feed = intent.Repository
	}
	component := strings.TrimSpace(req.ProGetComponent)
	user := strings.TrimSpace(req.ProGetUser)
	apiKey := strings.TrimSpace(req.ProGetAPIKey)
	workers := req.ProGetWorkers
	if strings.TrimSpace(apiKey) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("proget api key is required for proget backend")
	}

	adapter := adapters.NewRepoSnapshotProGetAdapter(endpoint, feed, component, debsDir, user, apiKey, intent.SnapshotPrefix, workers, req.ProGetTimeoutSec, req.ProGetRetries, req.ProGetRetryDelayMs)
	if err := adapter.Publish(ctx, intent.SnapshotID); err != nil {
		return err
	}
	if strings.TrimSpace(intent.Channel) != "" {
		return adapter.Promote(ctx, intent.SnapshotID, intent.Channel)
	}
	return nil
}
