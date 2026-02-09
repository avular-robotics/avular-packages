package integration

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/adapters"
)

func TestProGetUploadIntegration(t *testing.T) {
	debsDir := t.TempDir()
	debPath := filepath.Join(debsDir, "test.deb")
	require.NoError(t, os.WriteFile(debPath, []byte("payload"), 0644))

	t.Run("uploads via Debian endpoint", func(t *testing.T) {
		ctx := t.Context()
		var requests []requestInfo
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, _ := r.BasicAuth()
			requests = append(requests, requestInfo{
				Method: r.Method,
				Path:   r.URL.Path,
				User:   user,
				Pass:   pass,
			})
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		adapter := adapters.NewRepoSnapshotProGetAdapter(adapters.ProGetConfig{
			Endpoint:       server.URL,
			Feed:           "avular",
			DebsDir:        debsDir,
			APIKey:         "secret",
			SnapshotPrefix: "snap",
			Workers:        1,
			TimeoutSec:     1,
			Retries:        1,
			RetryDelayMs:   1,
		})
		require.NoError(t, adapter.Publish(ctx, "abc"))
		require.NoError(t, adapter.Promote(ctx, "abc", "dev"))

		expected := []requestInfo{
			{
				Method: "PUT",
				Path:   "/debian/avular/upload/snap-abc/main",
				User:   "api",
				Pass:   "secret",
			},
			{
				Method: "PUT",
				Path:   "/debian/avular/upload/dev/main",
				User:   "api",
				Pass:   "secret",
			},
		}
		if diff := cmp.Diff(expected, requests); diff != "" {
			t.Fatalf("unexpected requests (-want +got):\n%s", diff)
		}
	})

	t.Run("ignores already exists responses", func(t *testing.T) {
		ctx := t.Context()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("already exists"))
		}))
		defer server.Close()

		adapter := adapters.NewRepoSnapshotProGetAdapter(adapters.ProGetConfig{
			Endpoint:       server.URL,
			Feed:           "avular",
			DebsDir:        debsDir,
			APIKey:         "secret",
			SnapshotPrefix: "snap",
			Workers:        1,
			TimeoutSec:     1,
			Retries:        1,
			RetryDelayMs:   1,
		})
		require.NoError(t, adapter.Publish(ctx, "abc"))
	})
}

type requestInfo struct {
	Method string
	Path   string
	User   string
	Pass   string
}
