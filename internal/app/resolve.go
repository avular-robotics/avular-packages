package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/adapters"
	"avular-packages/internal/core"
	"avular-packages/internal/policies"
	"avular-packages/internal/types"
)

func (s Service) Resolve(ctx context.Context, req ResolveRequest) (ResolveResult, error) {
	productPath := strings.TrimSpace(req.ProductPath)
	if productPath == "" {
		productPath = discoverProduct()
	}
	if productPath == "" {
		return ResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("product spec path is required (provide --product or place product.yaml in current directory)")
	}

	product, err := s.SpecLoader.LoadProduct(productPath)
	if err != nil {
		return ResolveResult{}, err
	}

	// Emit hints about flags that duplicate spec defaults (before applying).
	emitHints(checkResolveDefaultsHints(req, product.Defaults))

	// Apply spec defaults for values not provided by the caller
	req = applySpecDefaults(req, product.Defaults)

	repoIndex := strings.TrimSpace(req.RepoIndex)
	if repoIndex == "" {
		return ResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("repo index file is required (provide --repo-index or set defaults.repo_index in product spec)")
	}
	outputDir := strings.TrimSpace(req.OutputDir)
	if outputDir == "" {
		outputDir = "out"
	}
	targetUbuntu := strings.TrimSpace(req.TargetUbuntu)
	if targetUbuntu == "" {
		return ResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("target Ubuntu release is required (provide --target-ubuntu or set defaults.target_ubuntu in product spec)")
	}
	targetUbuntu = normalizeTargetUbuntu(targetUbuntu)

	profiles, err := s.ProfileSource.LoadProfiles(product, req.Profiles)
	if err != nil {
		return ResolveResult{}, err
	}
	composer := core.NewProductComposer()
	compiler := core.NewSpecCompiler()
	composed, err := composer.Compose(ctx, product, profiles)
	if err != nil {
		return ResolveResult{}, err
	}
	if err := compiler.ValidateSpec(ctx, composed); err != nil {
		return ResolveResult{}, err
	}

	// Auto-discover schemas from a schemas/ directory next to the product spec.
	// These sit between inline schemas (lowest) and explicit schema_files (higher).
	discoveredSchemas := discoverSchemaFiles(productPath)

	// Build the final schema files list with correct precedence:
	//   1. Inline schema  (handled by builder, lowest)
	//   2. Auto-discovered ./schemas/*.yaml
	//   3. Explicit schema_files from spec
	//   4. CLI --schema flags (highest)
	resolveInputs := composed.Inputs
	if len(discoveredSchemas) > 0 {
		resolveInputs.PackageXML.SchemaFiles = append(
			discoveredSchemas,
			resolveInputs.PackageXML.SchemaFiles...,
		)
	}
	if len(req.SchemaFiles) > 0 {
		resolveInputs.PackageXML.SchemaFiles = append(
			resolveInputs.PackageXML.SchemaFiles,
			req.SchemaFiles...,
		)
	}

	// Collect inline schema from the composed spec.  The composer
	// merges schemas from all profiles and the product (product wins
	// per key), so this captures the fully-merged inline schema.
	var inlineSchema *types.SchemaFile
	if composed.Schema != nil && len(composed.Schema.Mappings) > 0 {
		inlineSchema = composed.Schema
	}

	builder := core.NewDependencyBuilder(s.Workspace, s.PackageXML)
	if s.SchemaResolver != nil {
		builder = builder.WithSchemaResolver(s.SchemaResolver)
	}
	deps, err := builder.BuildFromSpecsWithSchema(ctx, product, profiles, resolveInputs, req.Workspace, inlineSchema)
	if err != nil {
		return ResolveResult{}, err
	}

	policy := policies.NewPackagingPolicy(composed.Packaging.Groups, targetUbuntu)
	resolver := core.NewResolverCore(adapters.NewRepoIndexFileAdapter(repoIndex), policy)
	resolver.UseAptSolver = req.AptSatSolver
	result, err := resolver.Resolve(ctx, deps, composed.Resolutions)
	if err != nil {
		return ResolveResult{}, err
	}

	snapshotID := strings.TrimSpace(req.SnapshotID)
	if snapshotID == "" {
		snapshotID = buildSnapshotID(composed.Publish.Repository, targetUbuntu, result.AptLocks)
	}
	intent := buildSnapshotIntent(composed.Publish.Repository, snapshotID, s.Clock)

	if err := writeResolveOutputs(outputDir, req, result, intent); err != nil {
		return ResolveResult{}, err
	}
	return ResolveResult{
		ProductName: composed.Metadata.Name,
		SnapshotID:  snapshotID,
		OutputDir:   outputDir,
	}, nil
}

// writeResolveOutputs persists all resolver artifacts to the output
// directory: lock files, manifests, snapshot intent, and optional
// compatibility outputs.
func writeResolveOutputs(outputDir string, req ResolveRequest, result core.ResolveResult, intent types.SnapshotIntent) error {
	output := adapters.NewOutputFileAdapter(outputDir)
	if err := output.WriteAptLock(result.AptLocks); err != nil {
		return err
	}
	if err := output.WriteBundleManifest(result.BundleManifest); err != nil {
		return err
	}
	if err := output.WriteSnapshotIntent(intent); err != nil {
		return err
	}
	if err := output.WriteResolutionReport(result.Resolution); err != nil {
		return err
	}
	if req.EmitAptPreferences {
		if err := output.WriteAptPreferences(result.AptLocks); err != nil {
			return err
		}
	}
	if req.EmitAptInstallList {
		if err := output.WriteAptInstallList(result.AptLocks); err != nil {
			return err
		}
	}
	if req.EmitSnapshotSources {
		if err := output.WriteSnapshotSources(intent, req.SnapshotAptBaseURL, req.SnapshotAptComponent, req.SnapshotAptArchs); err != nil {
			return err
		}
	}
	if req.CompatGet {
		compat := adapters.NewCompatibilityOutputAdapter(outputDir)
		if err := compat.WriteGetDependencies(result.ResolvedDeps); err != nil {
			return err
		}
	}
	if req.CompatRosdep {
		compat := adapters.NewCompatibilityOutputAdapter(outputDir)
		if err := compat.WriteRosdepMapping(result.ResolvedDeps); err != nil {
			return err
		}
	}
	return nil
}

// applySpecDefaults fills in ResolveRequest fields from the product
// spec's defaults section when the request field is empty.
func applySpecDefaults(req ResolveRequest, defaults types.SpecDefaults) ResolveRequest {
	if strings.TrimSpace(req.TargetUbuntu) == "" && defaults.TargetUbuntu != "" {
		req.TargetUbuntu = defaults.TargetUbuntu
	}
	if len(req.Workspace) == 0 && len(defaults.Workspace) > 0 {
		req.Workspace = defaults.Workspace
	}
	if strings.TrimSpace(req.RepoIndex) == "" && defaults.RepoIndex != "" {
		req.RepoIndex = defaults.RepoIndex
	}
	if strings.TrimSpace(req.OutputDir) == "" && defaults.Output != "" {
		req.OutputDir = defaults.Output
	}
	return req
}

// discoverProduct attempts to find a product spec in conventional
// locations.  Returns the path or empty string if none found.
func discoverProduct() string {
	candidates := []string{
		"product.yaml",
		"product.yml",
		"avular-product.yaml",
		"avular-product.yml",
	}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// discoverSchemaFiles looks for a schemas/ directory next to the
// product spec and returns sorted paths to any .yaml/.yml files found.
// Returns nil if no schemas directory exists or it contains no files.
func discoverSchemaFiles(productPath string) []string {
	if productPath == "" {
		return nil
	}
	dir := filepath.Dir(productPath)
	schemasDir := filepath.Join(dir, "schemas")
	info, err := os.Stat(schemasDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			paths = append(paths, filepath.Join(schemasDir, name))
		}
	}
	sort.Strings(paths)
	return paths
}

func buildSnapshotIntent(repo types.PublishRepository, snapshotID string, clock func() time.Time) types.SnapshotIntent {
	now := time.Now().UTC()
	if clock != nil {
		now = clock().UTC()
	}
	return types.SnapshotIntent{
		Repository:     repo.Name,
		Channel:        repo.Channel,
		SnapshotPrefix: repo.SnapshotPrefix,
		SnapshotID:     snapshotID,
		CreatedAt:      now.Format(time.RFC3339),
		SigningKey:     repo.SigningKey,
	}
}

func buildSnapshotID(repo types.PublishRepository, targetUbuntu string, locks []types.AptLockEntry) string {
	ordered := append([]types.AptLockEntry(nil), locks...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Package < ordered[j].Package
	})
	var builder strings.Builder
	builder.WriteString(repo.Name)
	builder.WriteString("\n")
	builder.WriteString(repo.Channel)
	builder.WriteString("\n")
	builder.WriteString(repo.SnapshotPrefix)
	builder.WriteString("\n")
	builder.WriteString(targetUbuntu)
	builder.WriteString("\n")
	for _, entry := range ordered {
		builder.WriteString(entry.Package)
		builder.WriteString("=")
		builder.WriteString(entry.Version)
		builder.WriteString("\n")
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return fmt.Sprintf("%s-%s", repo.SnapshotPrefix, hex.EncodeToString(sum[:])[:12])
}

func normalizeTargetUbuntu(value string) string {
	normalized := strings.TrimSpace(value)
	lower := strings.ToLower(normalized)
	if strings.HasPrefix(lower, "ubuntu-") {
		return strings.TrimSpace(normalized[len("ubuntu-"):])
	}
	return normalized
}
