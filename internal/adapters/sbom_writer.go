package adapters

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

type SBOMWriterAdapter struct{}

func NewSBOMWriterAdapter() SBOMWriterAdapter {
	return SBOMWriterAdapter{}
}

func (a SBOMWriterAdapter) WriteSBOM(repoDir string, snapshotID string, createdAt string, locks []types.AptLockEntry) error {
	if strings.TrimSpace(repoDir) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("repo directory is empty")
	}
	if strings.TrimSpace(snapshotID) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("snapshot id is empty")
	}
	snapshotsDir := filepath.Join(repoDir, "snapshots")
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create snapshots directory").
			WithCause(err)
	}
	ordered := append([]types.AptLockEntry(nil), locks...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Package < ordered[j].Package
	})
	type spdxCreationInfo struct {
		Created  string   `json:"created"`
		Creators []string `json:"creators"`
	}
	type spdxPackage struct {
		SPDXID           string `json:"SPDXID"`
		Name             string `json:"name"`
		VersionInfo      string `json:"versionInfo"`
		DownloadLocation string `json:"downloadLocation"`
		LicenseConcluded string `json:"licenseConcluded"`
		LicenseDeclared  string `json:"licenseDeclared"`
		Supplier         string `json:"supplier"`
	}
	type spdxRelationship struct {
		SpdxElementID      string `json:"spdxElementId"`
		RelationshipType   string `json:"relationshipType"`
		RelatedSpdxElement string `json:"relatedSpdxElement"`
	}
	created := strings.TrimSpace(createdAt)
	if created == "" {
		created = time.Now().UTC().Format(time.RFC3339)
	}
	payload := struct {
		SPDXVersion       string             `json:"SPDXVersion"`
		DataLicense       string             `json:"DataLicense"`
		SPDXID            string             `json:"SPDXID"`
		Name              string             `json:"name"`
		DocumentNamespace string             `json:"documentNamespace"`
		CreationInfo      spdxCreationInfo   `json:"creationInfo"`
		Packages          []spdxPackage      `json:"packages"`
		Relationships     []spdxRelationship `json:"relationships"`
		DocumentDescribes []string           `json:"documentDescribes"`
	}{
		SPDXVersion:       "SPDX-2.3",
		DataLicense:       "CC0-1.0",
		SPDXID:            "SPDXRef-DOCUMENT",
		Name:              fmt.Sprintf("avular-packages snapshot %s", snapshotID),
		DocumentNamespace: fmt.Sprintf("https://avular.dev/spdx/snapshots/%s", snapshotID),
		CreationInfo: spdxCreationInfo{
			Created:  created,
			Creators: []string{"Tool: avular-packages"},
		},
	}
	for _, entry := range ordered {
		spdxID := spdxPackageID(entry.Package, entry.Version)
		payload.Packages = append(payload.Packages, spdxPackage{
			SPDXID:           spdxID,
			Name:             entry.Package,
			VersionInfo:      entry.Version,
			DownloadLocation: "NOASSERTION",
			LicenseConcluded: "NOASSERTION",
			LicenseDeclared:  "NOASSERTION",
			Supplier:         "NOASSERTION",
		})
		payload.DocumentDescribes = append(payload.DocumentDescribes, spdxID)
		payload.Relationships = append(payload.Relationships, spdxRelationship{
			SpdxElementID:      "SPDXRef-DOCUMENT",
			RelationshipType:   "DESCRIBES",
			RelatedSpdxElement: spdxID,
		})
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to marshal sbom payload").
			WithCause(err)
	}
	path := filepath.Join(snapshotsDir, snapshotID+".sbom.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write sbom file").
			WithCause(err)
	}
	return nil
}

func spdxPackageID(name string, version string) string {
	seed := fmt.Sprintf("%s@%s", name, version)
	hash := sha256.Sum256([]byte(seed))
	return "SPDXRef-Package-" + hex.EncodeToString(hash[:8])
}

var _ ports.SBOMPort = SBOMWriterAdapter{}
