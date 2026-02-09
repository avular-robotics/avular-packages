package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
		return ResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("product spec path is required")
	}
	repoIndex := strings.TrimSpace(req.RepoIndex)
	if repoIndex == "" {
		return ResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("repo index file is required")
	}
	outputDir := strings.TrimSpace(req.OutputDir)
	if outputDir == "" {
		return ResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("output directory is required")
	}
	targetUbuntu := strings.TrimSpace(req.TargetUbuntu)
	if targetUbuntu == "" {
		return ResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("target Ubuntu release is required")
	}
	targetUbuntu = normalizeTargetUbuntu(targetUbuntu)

	product, err := s.SpecLoader.LoadProduct(productPath)
	if err != nil {
		return ResolveResult{}, err
	}
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

	// Merge CLI schema files with spec-level ones (CLI appended last = highest precedence)
	resolveInputs := composed.Inputs
	if len(req.SchemaFiles) > 0 {
		resolveInputs.PackageXML.SchemaFiles = append(
			resolveInputs.PackageXML.SchemaFiles,
			req.SchemaFiles...,
		)
	}

	builder := core.NewDependencyBuilder(s.Workspace, s.PackageXML)
	if s.SchemaResolver != nil {
		builder = builder.WithSchemaResolver(s.SchemaResolver)
	}
	deps, err := builder.BuildFromSpecs(ctx, product, profiles, resolveInputs, req.Workspace)
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

	output := adapters.NewOutputFileAdapter(outputDir)
	if err := output.WriteAptLock(result.AptLocks); err != nil {
		return ResolveResult{}, err
	}
	if err := output.WriteBundleManifest(result.BundleManifest); err != nil {
		return ResolveResult{}, err
	}
	snapshotID := strings.TrimSpace(req.SnapshotID)
	if snapshotID == "" {
		snapshotID = buildSnapshotID(composed.Publish.Repository, targetUbuntu, result.AptLocks)
	}
	intent := buildSnapshotIntent(composed.Publish.Repository, snapshotID, s.Clock)
	if err := output.WriteSnapshotIntent(intent); err != nil {
		return ResolveResult{}, err
	}
	if err := output.WriteResolutionReport(result.Resolution); err != nil {
		return ResolveResult{}, err
	}
	if req.EmitAptPreferences {
		if err := output.WriteAptPreferences(result.AptLocks); err != nil {
			return ResolveResult{}, err
		}
	}
	if req.EmitAptInstallList {
		if err := output.WriteAptInstallList(result.AptLocks); err != nil {
			return ResolveResult{}, err
		}
	}
	if req.EmitSnapshotSources {
		if err := output.WriteSnapshotSources(intent, req.SnapshotAptBaseURL, req.SnapshotAptComponent, req.SnapshotAptArchs); err != nil {
			return ResolveResult{}, err
		}
	}
	if req.CompatGet {
		compat := adapters.NewCompatibilityOutputAdapter(outputDir)
		if err := compat.WriteGetDependencies(result.ResolvedDeps); err != nil {
			return ResolveResult{}, err
		}
	}
	if req.CompatRosdep {
		compat := adapters.NewCompatibilityOutputAdapter(outputDir)
		if err := compat.WriteRosdepMapping(result.ResolvedDeps); err != nil {
			return ResolveResult{}, err
		}
	}
	return ResolveResult{
		ProductName: composed.Metadata.Name,
		SnapshotID:  snapshotID,
		OutputDir:   outputDir,
	}, nil
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
