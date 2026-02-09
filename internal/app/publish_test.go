package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

// stubOutputReader satisfies ports.OutputReaderPort for testing Publish
// validation paths that occur *before* the backend is invoked.
type stubOutputReader struct {
	intent types.SnapshotIntent
	err    error
}

func (s stubOutputReader) ReadAptLock(_ string) ([]types.AptLockEntry, error) {
	return nil, nil
}
func (s stubOutputReader) ReadBundleManifest(_ string) ([]types.BundleManifestEntry, error) {
	return nil, nil
}
func (s stubOutputReader) ReadResolutionReport(_ string) (types.ResolutionReport, error) {
	return types.ResolutionReport{}, nil
}
func (s stubOutputReader) ReadSnapshotIntent(_ string) (types.SnapshotIntent, error) {
	return s.intent, s.err
}

// stubSBOMWriter satisfies ports.SBOMPort.
type stubSBOMWriter struct{}

func (stubSBOMWriter) WriteSBOM(_, _, _ string, _ []types.AptLockEntry) error { return nil }

func TestPublish_EmptyOutputDir(t *testing.T) {
	svc := Service{}
	_, err := svc.Publish(context.Background(), PublishRequest{OutputDir: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "output directory is required")
}

func TestPublish_UnsupportedBackend(t *testing.T) {
	svc := Service{
		OutputReader: stubOutputReader{intent: types.SnapshotIntent{SnapshotID: "test-snap"}},
	}
	_, err := svc.Publish(context.Background(), PublishRequest{
		OutputDir:   "/tmp/test-publish",
		RepoBackend: "unsupported",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported repo backend")
}

func TestPublish_AptlyMissingGPGKey(t *testing.T) {
	svc := Service{
		OutputReader: stubOutputReader{
			intent: types.SnapshotIntent{
				SnapshotID: "test-snap",
				Repository: "testrepo",
				Channel:    "stable",
				SigningKey: "", // no key in intent
			},
		},
	}
	_, err := svc.Publish(context.Background(), PublishRequest{
		OutputDir:   "/tmp/test-publish",
		RepoBackend: "aptly",
		GpgKey:      "", // no key in request either
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gpg key is required for aptly backend")
}

func TestPublish_ProGetMissingAPIKey(t *testing.T) {
	svc := Service{
		OutputReader: stubOutputReader{
			intent: types.SnapshotIntent{
				SnapshotID: "test-snap",
				Repository: "testrepo",
			},
		},
	}
	_, err := svc.Publish(context.Background(), PublishRequest{
		OutputDir:    "/tmp/test-publish",
		RepoBackend:  "proget",
		ProGetAPIKey: "", // no key
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proget api key is required for proget backend")
}

func TestPublish_BackendDefaultsToFile(t *testing.T) {
	// When RepoBackend is empty the code defaults to "file".
	// With a valid intent this will attempt to create the snapshot
	// directory, which will fail because the repo dir doesn't exist.
	// We just verify it does NOT fail on "unsupported backend".
	svc := Service{
		OutputReader: stubOutputReader{
			intent: types.SnapshotIntent{
				SnapshotID: "snap-123",
				Repository: "testrepo",
			},
		},
	}
	_, err := svc.Publish(context.Background(), PublishRequest{
		OutputDir:   "/tmp/test-publish-default-backend",
		RepoBackend: "", // should default to "file"
	})
	// It may error because /tmp/test-publish-default-backend/repo
	// doesn't exist, but the error should NOT be "unsupported repo backend".
	if err != nil {
		assert.NotContains(t, err.Error(), "unsupported repo backend")
	}
}
