package adapters

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
	"avular-packages/internal/shared"
	"avular-packages/internal/types"
)

type PackageBuildAdapter struct {
	PipIndexURL string
}

func NewPackageBuildAdapter(pipIndexURL string) PackageBuildAdapter {
	return PackageBuildAdapter{PipIndexURL: pipIndexURL}
}

func (a PackageBuildAdapter) BuildDebs(inputDir string, outputDir string) error {
	if strings.TrimSpace(inputDir) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("input directory is empty")
	}
	if strings.TrimSpace(outputDir) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("output directory is empty")
	}
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create output directory").
			WithCause(err)
	}

	internalDebs := filepath.Join(inputDir, "internal-debs")
	if _, err := os.Stat(internalDebs); err == nil {
		if err := copyDebs(internalDebs, outputDir); err != nil {
			return err
		}
	}

	manifest, err := loadBundleManifest(filepath.Join(inputDir, "bundle.manifest"))
	if err != nil {
		return err
	}
	pipDeps, err := loadGetDependenciesPip(filepath.Join(inputDir, "get-dependencies.pip"))
	if err != nil {
		return err
	}
	return buildPythonDebsFromManifest(manifest, pipDeps, outputDir, a.PipIndexURL)
}

// groupDeps pairs a packaging group with its resolved pip dependencies.
type groupDeps struct {
	group types.PackagingGroup
	deps  []types.ResolvedDependency
}

func buildPythonDebsFromManifest(manifest []types.BundleManifestEntry, pipDeps []types.ResolvedDependency, debsDir string, pipIndexURL string) error {
	grouped, err := groupManifestByPip(manifest, pipDeps)
	if err != nil {
		return err
	}
	built := map[string]string{}
	for _, entry := range grouped {
		sort.Slice(entry.deps, func(i, j int) bool {
			return entry.deps[i].Package < entry.deps[j].Package
		})
		switch entry.group.Mode {
		case types.PackagingModeIndividual:
			if err := buildResolvedPipDebs(entry.deps, pipIndexURL, debsDir, built); err != nil {
				return err
			}
		case types.PackagingModeMetaBundle:
			if err := buildResolvedPipDebs(entry.deps, pipIndexURL, debsDir, built); err != nil {
				return err
			}
			if err := buildMetaBundleDeb(entry.group.Name, entry.deps, debsDir); err != nil {
				return err
			}
		case types.PackagingModeFatBundle:
			if err := buildFatBundleDeb(entry.group.Name, entry.deps, debsDir, pipIndexURL); err != nil {
				return err
			}
		default:
			return errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg(fmt.Sprintf("unsupported packaging mode: %s", entry.group.Mode))
		}
	}
	return nil
}

// groupManifestByPip filters and groups manifest entries that match pip
// dependencies, returning them sorted by group name.
func groupManifestByPip(manifest []types.BundleManifestEntry, pipDeps []types.ResolvedDependency) ([]groupDeps, error) {
	pipSet := map[string]struct{}{}
	for _, dep := range pipDeps {
		if dep.Type != types.DependencyTypePip {
			continue
		}
		pipSet[dep.Package] = struct{}{}
	}
	grouped := map[string]*groupDeps{}
	for _, entry := range manifest {
		if _, ok := pipSet[entry.Package]; !ok {
			continue
		}
		ge, ok := grouped[entry.Group]
		if !ok {
			ge = &groupDeps{
				group: types.PackagingGroup{Name: entry.Group, Mode: entry.Mode},
			}
			grouped[entry.Group] = ge
		}
		if ge.group.Mode != entry.Mode {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg(fmt.Sprintf("bundle manifest mode mismatch for group %s", entry.Group))
		}
		ge.deps = append(ge.deps, types.ResolvedDependency{
			Type:    types.DependencyTypePip,
			Package: entry.Package,
			Version: entry.Version,
		})
	}
	var names []string
	for name := range grouped {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]groupDeps, 0, len(names))
	for _, name := range names {
		result = append(result, *grouped[name])
	}
	return result, nil
}

// buildResolvedPipDebs resolves pip dependencies, builds individual .deb
// packages, and tracks built versions to detect mismatches.
func buildResolvedPipDebs(deps []types.ResolvedDependency, pipIndexURL string, debsDir string, built map[string]string) error {
	resolved, err := resolvePipDependencies(deps, pipIndexURL)
	if err != nil {
		return err
	}
	for _, dep := range resolved.Packages {
		if existing, ok := built[dep.Package]; ok {
			if existing != dep.Version {
				return errbuilder.New().
					WithCode(errbuilder.CodeInvalidArgument).
					WithMsg(fmt.Sprintf("pip dependency version mismatch for %s: %s vs %s", dep.Package, existing, dep.Version))
			}
			continue
		}
		debDepends := pipDebDepends(dep.Package, resolved)
		if err := buildPythonPackageDeb(dep.Package, dep.Version, debsDir, pipIndexURL, debDepends); err != nil {
			return err
		}
		built[dep.Package] = dep.Version
	}
	return nil
}

func buildPythonPackageDeb(name string, version string, debsDir string, pipIndexURL string, debDepends []string) error {
	packageName := buildDebPackageNameParts("python3", name)
	staging, err := os.MkdirTemp("", "avular-python-")
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create staging directory").
			WithCause(err)
	}
	defer os.RemoveAll(staging)

	controlDir := filepath.Join(staging, "DEBIAN")
	if err := os.MkdirAll(controlDir, 0o750); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create control directory").
			WithCause(err)
	}
	sitePackages := filepath.Join(staging, "usr", "lib", "python3", "dist-packages")
	if err := os.MkdirAll(sitePackages, 0o750); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create site-packages directory").
			WithCause(err)
	}

	if err := pipInstall(sitePackages, []types.ResolvedDependency{{Package: name, Version: version}}, pipIndexURL, true); err != nil {
		return err
	}

	depends := formatDebDepends("python3", debDepends)
	control := buildControl(packageName, version, depends, fmt.Sprintf("Python package %s", name))
	if err := os.WriteFile(filepath.Join(controlDir, "control"), []byte(control), 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write control file").
			WithCause(err)
	}
	return buildDeb(staging, filepath.Join(debsDir, fmt.Sprintf("%s_%s_all.deb", packageName, version)))
}

func buildMetaBundleDeb(groupName string, deps []types.ResolvedDependency, debsDir string) error {
	packageName := buildDebPackageNameParts("python3", groupName, "meta")
	version := hashVersion(deps)
	staging, err := os.MkdirTemp("", "avular-meta-")
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create staging directory").
			WithCause(err)
	}
	defer os.RemoveAll(staging)

	controlDir := filepath.Join(staging, "DEBIAN")
	if err := os.MkdirAll(controlDir, 0o750); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create control directory").
			WithCause(err)
	}
	var depends []string
	for _, dep := range deps {
		pkgName := buildDebPackageNameParts("python3", dep.Package)
		depends = append(depends, fmt.Sprintf("%s (= %s)", pkgName, dep.Version))
	}
	control := buildControl(packageName, version, strings.Join(depends, ", "), fmt.Sprintf("Meta bundle for %s", groupName))
	if err := os.WriteFile(filepath.Join(controlDir, "control"), []byte(control), 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write control file").
			WithCause(err)
	}
	return buildDeb(staging, filepath.Join(debsDir, fmt.Sprintf("%s_%s_all.deb", packageName, version)))
}

func buildFatBundleDeb(groupName string, deps []types.ResolvedDependency, debsDir string, pipIndexURL string) error {
	packageName := buildDebPackageNameParts("python3", groupName, "fat")
	version := hashVersion(deps)
	staging, err := os.MkdirTemp("", "avular-fat-")
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create staging directory").
			WithCause(err)
	}
	defer os.RemoveAll(staging)

	controlDir := filepath.Join(staging, "DEBIAN")
	if err := os.MkdirAll(controlDir, 0o750); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create control directory").
			WithCause(err)
	}
	sitePackages := filepath.Join(staging, "usr", "lib", "python3", "dist-packages")
	if err := os.MkdirAll(sitePackages, 0o750); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create site-packages directory").
			WithCause(err)
	}
	if err := pipInstall(sitePackages, deps, pipIndexURL, false); err != nil {
		return err
	}

	control := buildControl(packageName, version, "python3", fmt.Sprintf("Fat bundle for %s", groupName))
	if err := os.WriteFile(filepath.Join(controlDir, "control"), []byte(control), 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write control file").
			WithCause(err)
	}
	return buildDeb(staging, filepath.Join(debsDir, fmt.Sprintf("%s_%s_all.deb", packageName, version)))
}

func pipInstall(targetDir string, deps []types.ResolvedDependency, pipIndexURL string, noDeps bool) error {
	var args []string
	args = append(args, "-m", "pip", "install", "--target", targetDir)
	if noDeps {
		args = append(args, "--no-deps")
	}
	if strings.TrimSpace(pipIndexURL) != "" {
		args = append(args, "--index-url", pipIndexURL)
	}
	for _, dep := range deps {
		args = append(args, fmt.Sprintf("%s==%s", dep.Package, dep.Version))
	}
	cmd := exec.Command("python3", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("pip install failed").
			WithCause(shared.CommandError(output, err))
	}
	return nil
}

type pipResolveResult struct {
	Packages []types.ResolvedDependency
	Versions map[string]string
	Requires map[string][]string
}

type pipListEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type pipMetadata struct {
	Name     string
	Version  string
	Requires []string
}

func resolvePipDependencies(deps []types.ResolvedDependency, pipIndexURL string) (pipResolveResult, error) {
	result := pipResolveResult{
		Packages: []types.ResolvedDependency{},
		Versions: map[string]string{},
		Requires: map[string][]string{},
	}
	if len(deps) == 0 {
		return result, nil
	}
	staging, err := os.MkdirTemp("", "avular-pip-resolve-")
	if err != nil {
		return pipResolveResult{}, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create pip resolve directory").
			WithCause(err)
	}
	defer os.RemoveAll(staging)

	if err := pipInstall(staging, deps, pipIndexURL, false); err != nil {
		return pipResolveResult{}, err
	}

	versions, err := pipList(staging)
	if err != nil {
		return pipResolveResult{}, err
	}
	metadata, err := readPipMetadata(staging)
	if err != nil {
		return pipResolveResult{}, err
	}

	requires := map[string][]string{}
	for name, meta := range metadata {
		normalized := name
		if _, ok := versions[normalized]; !ok {
			continue
		}
		var deps []string
		for _, req := range meta.Requires {
			reqName := parseRequiresDistName(req)
			if strings.TrimSpace(reqName) == "" {
				continue
			}
			reqName = shared.NormalizePipName(reqName)
			if reqName == normalized {
				continue
			}
			if _, ok := versions[reqName]; !ok {
				continue
			}
			deps = append(deps, reqName)
		}
		requires[normalized] = uniqueSortedStrings(deps)
	}

	packages := make([]types.ResolvedDependency, 0, len(versions))
	for name, version := range versions {
		packages = append(packages, types.ResolvedDependency{
			Type:    types.DependencyTypePip,
			Package: name,
			Version: version,
		})
	}
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Package < packages[j].Package
	})

	result.Packages = packages
	result.Versions = versions
	result.Requires = requires
	return result, nil
}

func pipList(targetDir string) (map[string]string, error) {
	cmd := exec.Command("python3", "-m", "pip", "list", "--format=json", "--path", targetDir)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("pip list failed").
			WithCause(fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err))
	}
	var entries []pipListEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("pip list output is invalid").
			WithCause(err)
	}
	versions := map[string]string{}
	for _, entry := range entries {
		name := shared.NormalizePipName(entry.Name)
		if strings.TrimSpace(name) == "" {
			continue
		}
		versions[name] = strings.TrimSpace(entry.Version)
	}
	return versions, nil
}

func readPipMetadata(targetDir string) (map[string]pipMetadata, error) {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to read pip metadata directory").
			WithCause(err)
	}
	metadata := map[string]pipMetadata{}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dist-info") {
			continue
		}
		metadataPath := filepath.Join(targetDir, entry.Name(), "METADATA")
		content, err := os.ReadFile(metadataPath)
		if err != nil {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("failed to read pip metadata").
				WithCause(err)
		}
		var name string
		var version string
		var requires []string
		for _, line := range strings.Split(string(content), "\n") {
			switch {
			case strings.HasPrefix(line, "Name:"):
				name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
			case strings.HasPrefix(line, "Version:"):
				version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
			case strings.HasPrefix(line, "Requires-Dist:"):
				requires = append(requires, strings.TrimSpace(strings.TrimPrefix(line, "Requires-Dist:")))
			}
		}
		if strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
			continue
		}
		normalized := shared.NormalizePipName(name)
		metadata[normalized] = pipMetadata{
			Name:     name,
			Version:  version,
			Requires: requires,
		}
	}
	return metadata, nil
}

func parseRequiresDistName(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return ""
	}
	if idx := strings.Index(cleaned, ";"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}
	if idx := strings.Index(cleaned, "["); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}
	var builder strings.Builder
	for _, r := range cleaned {
		if isPipNameRune(r) {
			builder.WriteRune(r)
			continue
		}
		break
	}
	return builder.String()
}

func isPipNameRune(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	return r == '-' || r == '_' || r == '.'
}

func uniqueSortedStrings(values []string) []string {
	unique := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		unique[trimmed] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func formatDebDepends(base string, deps []string) string {
	cleaned := uniqueSortedStrings(deps)
	if strings.TrimSpace(base) == "" {
		return strings.Join(cleaned, ", ")
	}
	for i := 0; i < len(cleaned); i++ {
		if cleaned[i] == base {
			cleaned = append(cleaned[:i], cleaned[i+1:]...)
			break
		}
	}
	if len(cleaned) == 0 {
		return base
	}
	return strings.Join(append([]string{base}, cleaned...), ", ")
}

func pipDebDepends(name string, resolved pipResolveResult) []string {
	required := resolved.Requires[name]
	if len(required) == 0 {
		return nil
	}
	var depends []string
	for _, depName := range required {
		version, ok := resolved.Versions[depName]
		if !ok {
			continue
		}
		depPackage := buildDebPackageNameParts("python3", depName)
		depends = append(depends, fmt.Sprintf("%s (= %s)", depPackage, version))
	}
	sort.Strings(depends)
	return depends
}

func buildDeb(stagingDir string, outputPath string) error {
	cmd := exec.Command("dpkg-deb", "--build", stagingDir, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("dpkg-deb build failed").
			WithCause(shared.CommandError(output, err))
	}
	return nil
}

func buildControl(packageName string, version string, depends string, description string) string {
	var builder strings.Builder
	builder.WriteString("Package: ")
	builder.WriteString(packageName)
	builder.WriteString("\n")
	builder.WriteString("Version: ")
	builder.WriteString(version)
	builder.WriteString("\n")
	builder.WriteString("Architecture: all\n")
	builder.WriteString("Maintainer: avular\n")
	if strings.TrimSpace(depends) != "" {
		builder.WriteString("Depends: ")
		builder.WriteString(depends)
		builder.WriteString("\n")
	}
	builder.WriteString("Description: ")
	builder.WriteString(description)
	builder.WriteString("\n")
	return builder.String()
}

func normalizeDebPackageName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return normalized
}

func buildDebPackageNameParts(parts ...string) string {
	var tokens []string
	for _, part := range parts {
		normalized := normalizeDebPackageName(part)
		if normalized == "" {
			continue
		}
		for _, token := range strings.Split(normalized, "-") {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			tokens = append(tokens, token)
		}
	}
	tokens = dedupeTokens(tokens)
	return strings.Join(tokens, "-")
}

func dedupeTokens(tokens []string) []string {
	if len(tokens) == 0 {
		return tokens
	}
	seen := map[string]struct{}{}
	var deduped []string
	for _, token := range tokens {
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		deduped = append(deduped, token)
	}
	return deduped
}

func hashVersion(deps []types.ResolvedDependency) string {
	var builder strings.Builder
	for _, dep := range deps {
		builder.WriteString(dep.Package)
		builder.WriteString("=")
		builder.WriteString(dep.Version)
		builder.WriteString("\n")
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return "0.0.0+" + hex.EncodeToString(sum[:])[:8]
}

func loadBundleManifest(path string) ([]types.BundleManifestEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("bundle.manifest not found").
			WithCause(err)
	}
	var entries []types.BundleManifestEntry
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) != 4 {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("invalid bundle.manifest format")
		}
		entry := types.BundleManifestEntry{
			Group:   strings.TrimSpace(parts[0]),
			Mode:    types.PackagingMode(strings.TrimSpace(parts[1])),
			Package: strings.TrimSpace(parts[2]),
			Version: strings.TrimSpace(parts[3]),
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func loadGetDependenciesPip(path string) ([]types.ResolvedDependency, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("get-dependencies.pip not found").
			WithCause(err)
	}
	var deps []types.ResolvedDependency
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "==", 2)
		if len(parts) != 2 {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("invalid get-dependencies.pip format")
		}
		deps = append(deps, types.ResolvedDependency{
			Type:    types.DependencyTypePip,
			Package: strings.TrimSpace(parts[0]),
			Version: strings.TrimSpace(parts[1]),
		})
	}
	return deps, nil
}

func copyDebs(srcDir string, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("internal deb dir not found").
			WithCause(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".deb") {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())
		if err := copyFile(srcPath, destPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(srcPath string, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("failed to open source deb").
			WithCause(err)
	}
	defer srcFile.Close()
	destFile, err := os.Create(destPath)
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create destination deb").
			WithCause(err)
	}
	defer destFile.Close()
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to copy deb").
			WithCause(err)
	}
	return nil
}

var _ ports.PackageBuildPort = PackageBuildAdapter{}
