package adapters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

func TestReadSnapshotIntent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    types.SnapshotIntent
	}{
		{
			name:    "basic",
			content: "repository=avular\nchannel=dev\nsnapshot_prefix=test\nsnapshot_id=test-123\ncreated_at=1970-01-01T00:00:00Z\n",
			want: types.SnapshotIntent{
				Repository:     "avular",
				Channel:        "dev",
				SnapshotPrefix: "test",
				SnapshotID:     "test-123",
				CreatedAt:      "1970-01-01T00:00:00Z",
			},
		},
		{
			name:    "includes signing key",
			content: "repository=avular\nchannel=\nsnapshot_prefix=prod\nsnapshot_id=prod-999\ncreated_at=2026-02-03T00:00:00Z\nsigning_key=key\n",
			want: types.SnapshotIntent{
				Repository:     "avular",
				Channel:        "",
				SnapshotPrefix: "prod",
				SnapshotID:     "prod-999",
				CreatedAt:      "2026-02-03T00:00:00Z",
				SigningKey:     "key",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			intentPath := filepath.Join(dir, "snapshot.intent")
			require.NoError(t, os.WriteFile(intentPath, []byte(tt.content), 0644))

			reader := NewOutputReaderAdapter()
			intent, err := reader.ReadSnapshotIntent(intentPath)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, intent); diff != "" {
				t.Fatalf("unexpected snapshot intent (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWriteSBOM(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "repo")
	locks := []types.AptLockEntry{
		{Package: "libfoo", Version: "1.0.0"},
		{Package: "libbar", Version: "2.0.0"},
	}

	writer := NewSBOMWriterAdapter()
	require.NoError(t, writer.WriteSBOM(repoDir, "snap-1", "1970-01-01T00:00:00Z", locks))
	sbomPath := filepath.Join(repoDir, "snapshots", "snap-1.sbom.json")
	content, err := os.ReadFile(sbomPath)
	require.NoError(t, err)
	var doc struct {
		SPDXVersion       string `json:"SPDXVersion"`
		DataLicense       string `json:"DataLicense"`
		SPDXID            string `json:"SPDXID"`
		Name              string `json:"name"`
		DocumentNamespace string `json:"documentNamespace"`
		CreationInfo      struct {
			Created  string   `json:"created"`
			Creators []string `json:"creators"`
		} `json:"creationInfo"`
		Packages []struct {
			SPDXID           string `json:"SPDXID"`
			Name             string `json:"name"`
			VersionInfo      string `json:"versionInfo"`
			DownloadLocation string `json:"downloadLocation"`
		} `json:"packages"`
		Relationships []struct {
			SpdxElementID      string `json:"spdxElementId"`
			RelationshipType   string `json:"relationshipType"`
			RelatedSpdxElement string `json:"relatedSpdxElement"`
		} `json:"relationships"`
		DocumentDescribes []string `json:"documentDescribes"`
	}
	require.NoError(t, json.Unmarshal(content, &doc))
	fieldChecks := []struct {
		name string
		got  string
		want string
	}{
		{name: "spdx version", got: doc.SPDXVersion, want: "SPDX-2.3"},
		{name: "data license", got: doc.DataLicense, want: "CC0-1.0"},
		{name: "spdx id", got: doc.SPDXID, want: "SPDXRef-DOCUMENT"},
		{name: "created", got: doc.CreationInfo.Created, want: "1970-01-01T00:00:00Z"},
	}
	for _, tt := range fieldChecks {
		if diff := cmp.Diff(tt.want, tt.got); diff != "" {
			t.Fatalf("unexpected %s (-want +got):\n%s", tt.name, diff)
		}
	}

	boolChecks := []struct {
		name string
		got  bool
		want bool
	}{
		{name: "name contains snapshot", got: strings.Contains(doc.Name, "snap-1"), want: true},
		{name: "namespace contains snapshot", got: strings.Contains(doc.DocumentNamespace, "snap-1"), want: true},
		{name: "creators not empty", got: len(doc.CreationInfo.Creators) > 0, want: true},
	}
	for _, tt := range boolChecks {
		if diff := cmp.Diff(tt.want, tt.got); diff != "" {
			t.Fatalf("unexpected %s (-want +got):\n%s", tt.name, diff)
		}
	}

	countChecks := []struct {
		name string
		got  int
		want int
	}{
		{name: "packages", got: len(doc.Packages), want: 2},
		{name: "document describes", got: len(doc.DocumentDescribes), want: 2},
		{name: "relationships", got: len(doc.Relationships), want: 2},
	}
	for _, tt := range countChecks {
		if diff := cmp.Diff(tt.want, tt.got); diff != "" {
			t.Fatalf("unexpected %s count (-want +got):\n%s", tt.name, diff)
		}
	}
}
