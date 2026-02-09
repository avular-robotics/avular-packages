//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"avular-packages/internal/adapters"
	"avular-packages/internal/app"
	"avular-packages/internal/core"
	"avular-packages/internal/types"
	"avular-packages/tests/testutil"
)

type progetRequest struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	User   string `json:"user"`
	Pass   string `json:"pass"`
}

func TestE2EProGetPublishWithTestcontainers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers e2e in short mode")
	}

	ctx := t.Context()
	endpoint, cleanup := startProGetMock(ctx, t)
	t.Cleanup(cleanup)
	artifactEndpoint, artifactCleanup := startArtifactServer(ctx, t)
	t.Cleanup(artifactCleanup)
	pipIndexURL, pipCleanup := startLocalPipIndex(ctx, t)
	t.Cleanup(pipCleanup)
	pipSimpleURL := strings.TrimRight(pipIndexURL, "/") + "/simple"

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "workspace")
	require.NoError(t, os.MkdirAll(workspaceRoot, 0755))

	pkgPath := filepath.Join(workspaceRoot, "sample_pkg", "package.xml")
	require.NoError(t, os.MkdirAll(filepath.Dir(pkgPath), 0755))
	require.NoError(t, os.WriteFile(pkgPath, []byte(testPackageXML), 0644))

	profilePath := filepath.Join(root, "profile.yaml")
	productPath := filepath.Join(root, "product.yaml")
	require.NoError(t, os.WriteFile(profilePath, []byte(buildProfileSpec()), 0644))
	require.NoError(t, os.WriteFile(productPath, []byte(buildProductSpec(profilePath)), 0644))

	repoIndexPath := filepath.Join(root, "repo-index.yaml")

	outputDir := filepath.Join(root, "out")

	service := app.NewService()
	_, err := service.RepoIndex(ctx, app.RepoIndexRequest{
		Output:           repoIndexPath,
		AptEndpoint:      artifactEndpoint,
		AptDistribution:  "dev",
		AptComponent:     "main",
		AptArch:          "amd64",
		PipIndex:         pipIndexURL,
		PipPackages:      []string{pipPackageName},
		HTTPTimeoutSec:   10,
		HTTPRetries:      1,
		HTTPRetryDelayMs: 100,
	})
	require.NoError(t, err)
	buildResult, err := service.Build(ctx, app.BuildRequest{
		OutputDir:    outputDir,
		ProductPath:  productPath,
		RepoIndex:    repoIndexPath,
		TargetUbuntu: "22.04",
		Workspace:    []string{workspaceRoot},
		PipIndexURL:  pipSimpleURL,
	})
	require.NoError(t, err)

	_, err = service.Publish(ctx, app.PublishRequest{
		OutputDir:          outputDir,
		RepoBackend:        "proget",
		DebsDir:            buildResult.DebsDir,
		ProGetEndpoint:     endpoint,
		ProGetAPIKey:       "secret",
		ProGetWorkers:      1,
		ProGetTimeoutSec:   60,
		ProGetRetries:      3,
		ProGetRetryDelayMs: 200,
	})
	require.NoError(t, err)

	intent, err := adapters.NewOutputReaderAdapter().ReadSnapshotIntent(filepath.Join(outputDir, "snapshot.intent"))
	require.NoError(t, err)

	requests, err := fetchProGetRequests(endpoint)
	require.NoError(t, err)
	debCount, err := countDebs(buildResult.DebsDir)
	require.NoError(t, err)
	require.Greater(t, debCount, 0)

	counts := map[string]int{}
	for _, req := range requests {
		require.Equal(t, "PUT", req.Method)
		require.Equal(t, "api", req.User)
		require.Equal(t, "secret", req.Pass)
		counts[req.Path]++
	}
	snapshotPath := fmt.Sprintf("/debian/avular/upload/%s/main", intent.SnapshotID)
	devPath := "/debian/avular/upload/dev/main"
	require.Equal(t, debCount, counts[snapshotPath])
	require.Equal(t, debCount, counts[devPath])
	require.Len(t, counts, 2)
}

func TestE2EFixturesWithTestcontainers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers e2e in short mode")
	}

	ctx := t.Context()
	endpoint, cleanup := startProGetMock(ctx, t)
	t.Cleanup(cleanup)
	artifactEndpoint, artifactCleanup := startArtifactServer(ctx, t)
	t.Cleanup(artifactCleanup)
	pipIndexURL, pipCleanup := startLocalPipIndex(ctx, t)
	t.Cleanup(pipCleanup)
	pipSimpleURL := strings.TrimRight(pipIndexURL, "/") + "/simple"

	workspaceRoot := t.TempDir()
	pkgPath := filepath.Join(workspaceRoot, "fixture_pkg", "package.xml")
	require.NoError(t, os.MkdirAll(filepath.Dir(pkgPath), 0755))
	require.NoError(t, os.WriteFile(pkgPath, []byte(fixturePackageXML), 0644))

	repoRoot := testutil.RepoRoot(t)

	productPath := filepath.Join(repoRoot, "fixtures", "e2e-product.yaml")
	profilePath := filepath.Join(repoRoot, "fixtures", "e2e-profile.yaml")
	outputDir := filepath.Join(repoRoot, "out", "e2e-mock")
	repoIndexPath := filepath.Join(outputDir, "repo-index.yaml")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	service := app.NewService()
	_, err = service.RepoIndex(ctx, app.RepoIndexRequest{
		Output:           repoIndexPath,
		AptEndpoint:      artifactEndpoint,
		AptDistribution:  "dev",
		AptComponent:     "main",
		AptArch:          "amd64",
		PipIndex:         pipIndexURL,
		PipPackages:      []string{pipPackageName},
		HTTPTimeoutSec:   10,
		HTTPRetries:      1,
		HTTPRetryDelayMs: 100,
	})
	require.NoError(t, err)

	buildResult, err := service.Build(ctx, app.BuildRequest{
		OutputDir:    outputDir,
		ProductPath:  productPath,
		Profiles:     []string{profilePath},
		RepoIndex:    repoIndexPath,
		TargetUbuntu: "22.04",
		Workspace:    []string{workspaceRoot},
		PipIndexURL:  pipSimpleURL,
	})
	require.NoError(t, err)

	_, err = service.Publish(ctx, app.PublishRequest{
		OutputDir:          outputDir,
		RepoBackend:        "proget",
		DebsDir:            buildResult.DebsDir,
		ProGetEndpoint:     endpoint,
		ProGetAPIKey:       "secret",
		ProGetWorkers:      1,
		ProGetTimeoutSec:   10,
		ProGetRetries:      1,
		ProGetRetryDelayMs: 100,
	})
	require.NoError(t, err)

	intent, err := adapters.NewOutputReaderAdapter().ReadSnapshotIntent(filepath.Join(outputDir, "snapshot.intent"))
	require.NoError(t, err)

	requests, err := fetchProGetRequests(endpoint)
	require.NoError(t, err)
	debCount, err := countDebs(buildResult.DebsDir)
	require.NoError(t, err)
	require.Greater(t, debCount, 0)

	counts := map[string]int{}
	for _, req := range requests {
		require.Equal(t, "PUT", req.Method)
		require.Equal(t, "api", req.User)
		require.Equal(t, "secret", req.Pass)
		counts[req.Path]++
	}
	snapshotPath := fmt.Sprintf("/debian/avular/upload/%s/main", intent.SnapshotID)
	devPath := "/debian/avular/upload/dev/main"
	require.Equal(t, debCount, counts[snapshotPath])
	require.Equal(t, debCount, counts[devPath])
	require.Len(t, counts, 2)
}

func TestE2ERealWorkspaceWithTestcontainers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers e2e in short mode")
	}
	workspaceRoots := parseWorkspaceRootsEnv()
	if len(workspaceRoots) == 0 {
		t.Skip("set AVULAR_E2E_WORKSPACE_ROOTS to run real workspace e2e")
	}

	ctx := t.Context()
	endpoint, cleanup := startProGetMock(ctx, t)
	t.Cleanup(cleanup)

	repoRoot, err := repoRootPath()
	require.NoError(t, err)
	productPath := filepath.Join(repoRoot, "fixtures", "e2e-product.yaml")
	profilePath := filepath.Join(repoRoot, "fixtures", "e2e-profile.yaml")

	aptDeps, pipDeps, err := collectWorkspaceDependencies(workspaceRoots)
	require.NoError(t, err)
	if len(aptDeps) == 0 && len(pipDeps) == 0 {
		t.Skip("no package.xml dependencies discovered in workspace roots")
	}

	composed, err := loadComposedSpec(productPath, profilePath)
	require.NoError(t, err)
	replaceMap := buildReplaceMap(composed.Resolutions)

	aptIndex, err := buildVersionIndex(aptDeps, types.DependencyTypeApt, replaceMap, "1.0.0")
	require.NoError(t, err)
	pipIndex, err := buildVersionIndex(pipDeps, types.DependencyTypePip, replaceMap, "0.1.0")
	require.NoError(t, err)

	aptLimit := parseLimitEnv("AVULAR_E2E_APT_LIMIT")
	pipLimit := parseLimitEnv("AVULAR_E2E_PIP_LIMIT")
	aptIndex = limitVersionMap(aptIndex, aptLimit)
	pipIndex = limitVersionMap(pipIndex, pipLimit)

	artifactEndpoint, artifactCleanup := startArtifactServerWithPackages(ctx, t, aptIndex)
	t.Cleanup(artifactCleanup)

	pipIndexURL, pipCleanup := startLocalPipIndexFromMap(ctx, t, pipIndex)
	t.Cleanup(pipCleanup)
	pipSimpleURL := strings.TrimRight(pipIndexURL, "/") + "/simple"

	outputDir := filepath.Join(repoRoot, "out", "e2e-real")
	repoIndexPath := filepath.Join(outputDir, "repo-index.yaml")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	service := app.NewService()
	_, err = service.RepoIndex(ctx, app.RepoIndexRequest{
		Output:           repoIndexPath,
		AptEndpoint:      artifactEndpoint,
		AptDistribution:  "dev",
		AptComponent:     "main",
		AptArch:          "amd64",
		PipIndex:         pipIndexURL,
		PipPackages:      mapKeys(pipIndex),
		HTTPTimeoutSec:   30,
		HTTPRetries:      1,
		HTTPRetryDelayMs: 100,
	})
	require.NoError(t, err)

	buildResult, err := service.Build(ctx, app.BuildRequest{
		OutputDir:    outputDir,
		ProductPath:  productPath,
		Profiles:     []string{profilePath},
		RepoIndex:    repoIndexPath,
		TargetUbuntu: "22.04",
		Workspace:    workspaceRoots,
		PipIndexURL:  pipSimpleURL,
	})
	require.NoError(t, err)

	_, err = service.Publish(ctx, app.PublishRequest{
		OutputDir:          outputDir,
		RepoBackend:        "proget",
		DebsDir:            buildResult.DebsDir,
		ProGetEndpoint:     endpoint,
		ProGetAPIKey:       "secret",
		ProGetWorkers:      1,
		ProGetTimeoutSec:   10,
		ProGetRetries:      1,
		ProGetRetryDelayMs: 100,
	})
	require.NoError(t, err)

	intent, err := adapters.NewOutputReaderAdapter().ReadSnapshotIntent(filepath.Join(outputDir, "snapshot.intent"))
	require.NoError(t, err)
	requests, err := fetchProGetRequests(endpoint)
	require.NoError(t, err)

	debCount, err := countDebs(buildResult.DebsDir)
	require.NoError(t, err)
	require.Greater(t, debCount, 0)

	counts := map[string]int{}
	for _, req := range requests {
		require.Equal(t, "PUT", req.Method)
		require.Equal(t, "api", req.User)
		require.Equal(t, "secret", req.Pass)
		counts[req.Path]++
	}
	snapshotPath := fmt.Sprintf("/debian/avular/upload/%s/main", intent.SnapshotID)
	devPath := "/debian/avular/upload/dev/main"
	require.Equal(t, debCount, counts[snapshotPath])
	require.Equal(t, debCount, counts[devPath])
	require.Len(t, counts, 2)
}

func startProGetMock(ctx context.Context, t *testing.T) (string, func()) {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        "python:3.12-alpine",
		ExposedPorts: []string{"8080/tcp"},
		Cmd:          []string{"python", "-c", progetMockScript},
		WaitingFor:   wait.ForListeningPort("8080/tcp").WithStartupTimeout(30 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err)

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())
	cleanup := func() {
		_ = container.Terminate(ctx)
	}
	return endpoint, cleanup
}

func startArtifactServer(ctx context.Context, t *testing.T) (string, func()) {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        "python:3.12-alpine",
		ExposedPorts: []string{"8081/tcp"},
		Cmd:          []string{"python", "-c", artifactServerScript},
		WaitingFor:   wait.ForListeningPort("8081/tcp").WithStartupTimeout(30 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "8081/tcp")
	require.NoError(t, err)

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())
	cleanup := func() {
		_ = container.Terminate(ctx)
	}
	return endpoint, cleanup
}

func startLocalPipIndex(ctx context.Context, t *testing.T) (string, func()) {
	t.Helper()
	root := t.TempDir()
	filesDir := filepath.Join(root, "files")
	simpleDir := filepath.Join(root, "simple", pipPackageName)
	require.NoError(t, os.MkdirAll(filesDir, 0755))
	require.NoError(t, os.MkdirAll(simpleDir, 0755))

	pkgRoot := filepath.Join(root, "src")
	pkgDir := filepath.Join(pkgRoot, pipPackageName)
	require.NoError(t, os.MkdirAll(pkgDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgRoot, "setup.py"), []byte(fmt.Sprintf(
		"from setuptools import setup\nsetup(name='%s', version='%s', packages=['%s'])\n",
		pipPackageName,
		pipPackageVersion,
		pipPackageName,
	)), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "__init__.py"), []byte(fmt.Sprintf(
		"__version__ = '%s'\n",
		pipPackageVersion,
	)), 0644))

	cmd := exec.CommandContext(ctx, "python3", "-m", "pip", "wheel", "--no-deps", "--no-build-isolation", "-w", filesDir, pkgRoot)
	cmd.Env = append(os.Environ(), "PIP_DISABLE_PIP_VERSION_CHECK=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "pip wheel failed: %s", strings.TrimSpace(string(output)))

	wheelName := fmt.Sprintf("%s-%s-py3-none-any.whl", pipPackageName, pipPackageVersion)
	if _, err := os.Stat(filepath.Join(filesDir, wheelName)); err != nil {
		entries, err := os.ReadDir(filesDir)
		require.NoError(t, err)
		require.NotEmpty(t, entries, "no wheels found in %s", filesDir)
		wheelName = entries[0].Name()
	}

	indexPath := filepath.Join(root, "simple", "index.html")
	require.NoError(t, os.WriteFile(indexPath, []byte(fmt.Sprintf(`<a href="/simple/%s/">%s</a>`, pipPackageName, pipPackageName)), 0644))
	pkgIndexPath := filepath.Join(simpleDir, "index.html")
	require.NoError(t, os.WriteFile(pkgIndexPath, []byte(fmt.Sprintf(`<a href="/files/%s">%s</a>`, wheelName, wheelName)), 0644))

	server := httptest.NewServer(http.FileServer(http.Dir(root)))
	cleanup := func() {
		server.Close()
	}
	return server.URL, cleanup
}

func fetchProGetRequests(endpoint string) ([]progetRequest, error) {
	resp, err := http.Get(endpoint + "/requests")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var requests []progetRequest
	if err := json.NewDecoder(resp.Body).Decode(&requests); err != nil {
		return nil, err
	}
	return requests, nil
}

const (
	pipPackageName    = "offlinepkg"
	pipPackageVersion = "0.1.0"
	aptPackageName    = "libexample"
	aptPackageVersion = "1.2.3"
)

const progetMockScript = `
import base64
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

requests = []

def parse_basic_auth(header_value):
    if not header_value:
        return "", ""
    if not header_value.startswith("Basic "):
        return "", ""
    try:
        raw = header_value.split(" ", 1)[1]
        decoded = base64.b64decode(raw).decode("utf-8")
        user, _, password = decoded.partition(":")
        return user, password
    except Exception:
        return "", ""

class Handler(BaseHTTPRequestHandler):
    def do_PUT(self):
        length = int(self.headers.get("Content-Length", "0"))
        if length > 0:
            _ = self.rfile.read(length)
        user, password = parse_basic_auth(self.headers.get("Authorization", ""))
        requests.append(
            {"method": "PUT", "path": self.path, "user": user, "pass": password}
        )
        self.send_response(201)
        self.end_headers()

    def do_GET(self):
        if self.path == "/requests":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps(requests).encode("utf-8"))
            return
        self.send_response(404)
        self.end_headers()

    def log_message(self, format, *args):
        return

def main():
    server = ThreadingHTTPServer(("0.0.0.0", 8080), Handler)
    server.serve_forever()

if __name__ == "__main__":
    main()
`

const artifactServerScript = `
import os

root = "/srv/repo"
apt_name = "` + aptPackageName + `"
apt_version = "` + aptPackageVersion + `"

apt_path = os.path.join(root, "dists", "dev", "main", "binary-amd64")
os.makedirs(apt_path, exist_ok=True)
with open(os.path.join(apt_path, "Packages"), "w") as f:
    f.write("Package: %s\nVersion: %s\n\n" % (apt_name, apt_version))

os.execvp("python", ["python", "-m", "http.server", "8081", "--directory", root])
`

const testPackageXML = `<package format="3">
  <name>sample_pkg</name>
  <version>0.0.1</version>
  <description>test package</description>
  <maintainer email="test@example.com">Test</maintainer>
  <license>MIT</license>
  <export>
    <pip_depend version="0.1.0">offlinepkg</pip_depend>
  </export>
</package>
`

const fixturePackageXML = `<package format="3">
  <name>fixture_pkg</name>
  <version>0.0.1</version>
  <description>fixture package</description>
  <maintainer email="fixture@example.com">Fixture</maintainer>
  <license>MIT</license>
  <export>
    <debian_depend>` + aptPackageName + `</debian_depend>
    <pip_depend version="` + pipPackageVersion + `">` + pipPackageName + `</pip_depend>
  </export>
</package>
`

func buildProfileSpec() string {
	return `api_version: "v1"
kind: "profile"
metadata:
  name: "test-profile"
  version: "2026.01.01"
  owners: ["platform"]
inputs:
  package_xml:
    enabled: true
    tags: ["pip_depend"]
packaging:
  groups:
    - name: "apt-runtime"
      mode: "individual"
      scope: "runtime"
      matches: ["apt:*"]
      targets: ["ubuntu-22.04"]
    - name: "python-runtime"
      mode: "individual"
      scope: "runtime"
      matches: ["pip:*"]
      targets: ["ubuntu-22.04"]
`
}

func buildProductSpec(profilePath string) string {
	return fmt.Sprintf(`api_version: "v1"
kind: "product"
metadata:
  name: "e2e-product"
  version: "2026.01.01"
  owners: ["platform"]
compose:
  - name: "%s"
    version: "2026.01.01"
    source: "local"
    path: "%s"
inputs:
  manual:
    apt:
      - "libexample=1.2.3"
publish:
  repository:
    name: "avular"
    channel: "dev"
    snapshot_prefix: "e2e"
    signing_key: "test-key"
`, filepath.Base(profilePath), profilePath)
}

func repoRootPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(cwd, "..", "..")), nil
}

func countDebs(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".deb") {
			count++
		}
	}
	return count, nil
}

func parseWorkspaceRootsEnv() []string {
	raw := strings.TrimSpace(os.Getenv("AVULAR_E2E_WORKSPACE_ROOTS"))
	if raw == "" {
		return nil
	}
	var roots []string
	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		roots = append(roots, value)
	}
	return roots
}

func parseLimitEnv(key string) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func collectWorkspaceDependencies(roots []string) ([]string, []string, error) {
	workspace := adapters.NewWorkspaceAdapter()
	pkgXML := adapters.NewPackageXMLAdapter()
	var paths []string
	for _, root := range roots {
		found, err := workspace.FindPackageXML(root)
		if err != nil {
			return nil, nil, err
		}
		for _, path := range found {
			if shouldSkipWorkspacePath(path) {
				continue
			}
			if _, err := os.Stat(path); err != nil {
				continue
			}
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return nil, nil, nil
	}
	return pkgXML.ParseDependencies(paths, []string{"debian_depend", "pip_depend"})
}

func shouldSkipWorkspacePath(path string) bool {
	ignored := []string{
		"/install/",
		"/build/",
		"/log/",
		"/devel/",
		"/.git/",
		"/.colcon/",
		"/.ros/",
	}
	for _, marker := range ignored {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

func loadComposedSpec(productPath string, profilePath string) (types.Spec, error) {
	specAdapter := adapters.NewSpecFileAdapter()
	product, err := specAdapter.LoadProduct(productPath)
	if err != nil {
		return types.Spec{}, err
	}
	profile, err := specAdapter.LoadProfile(profilePath)
	if err != nil {
		return types.Spec{}, err
	}
	composer := core.NewProductComposer()
	return composer.Compose(context.Background(), product, []types.Spec{profile})
}

func buildReplaceMap(directives []types.ResolutionDirective) map[string]string {
	out := map[string]string{}
	for _, directive := range directives {
		if strings.ToLower(strings.TrimSpace(directive.Action)) != "replace" {
			continue
		}
		parts := strings.SplitN(strings.TrimSpace(directive.Dependency), ":", 2)
		if len(parts) != 2 {
			continue
		}
		depType := strings.ToLower(strings.TrimSpace(parts[0]))
		name := strings.TrimSpace(parts[1])
		if depType == "pip" {
			name = normalizePipName(name)
		}
		replacement := strings.TrimSpace(directive.Value)
		if replacement == "" {
			continue
		}
		if depType == "pip" {
			replacement = normalizePipName(replacement)
		}
		out[depType+":"+name] = replacement
	}
	return out
}

func buildVersionIndex(entries []string, depType types.DependencyType, replaceMap map[string]string, defaultVersion string) (map[string][]string, error) {
	versions := map[string]map[string]struct{}{}
	for _, entry := range entries {
		constraint, err := core.ParseConstraint(entry, "workspace")
		if err != nil {
			return nil, err
		}
		name := strings.TrimSpace(constraint.Name)
		if name == "" {
			continue
		}
		if depType == types.DependencyTypePip {
			name = normalizePipName(name)
		}
		key := depTypeKey(depType, name)
		if replacement, ok := replaceMap[key]; ok {
			name = replacement
		}
		version := versionForConstraint(constraint, defaultVersion)
		if version == "" {
			continue
		}
		if versions[name] == nil {
			versions[name] = map[string]struct{}{}
		}
		versions[name][version] = struct{}{}
	}
	index := map[string][]string{}
	for name, set := range versions {
		var list []string
		for version := range set {
			list = append(list, version)
		}
		sort.Strings(list)
		index[name] = list
	}
	return index, nil
}

func versionForConstraint(constraint types.Constraint, fallback string) string {
	if constraint.Op == types.ConstraintOpNone {
		return fallback
	}
	if constraint.Op == types.ConstraintOpNe {
		version := fallback
		if version == constraint.Version && version != "" {
			return version + ".1"
		}
		return version
	}
	if strings.TrimSpace(constraint.Version) != "" {
		return strings.TrimSpace(constraint.Version)
	}
	return fallback
}

func depTypeKey(depType types.DependencyType, name string) string {
	switch depType {
	case types.DependencyTypeApt:
		return "apt:" + name
	case types.DependencyTypePip:
		return "pip:" + name
	default:
		return name
	}
}

func limitVersionMap(values map[string][]string, limit int) map[string][]string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	var names []string
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	limited := map[string][]string{}
	for i, name := range names {
		if i >= limit {
			break
		}
		limited[name] = values[name]
	}
	return limited
}

func startArtifactServerWithPackages(ctx context.Context, t *testing.T, packages map[string][]string) (string, func()) {
	t.Helper()
	script := buildArtifactServerScript(packages)
	req := testcontainers.ContainerRequest{
		Image:        "python:3.12-alpine",
		ExposedPorts: []string{"8081/tcp"},
		Cmd:          []string{"python", "-c", script},
		WaitingFor:   wait.ForListeningPort("8081/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "8081/tcp")
	require.NoError(t, err)

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())
	cleanup := func() {
		_ = container.Terminate(ctx)
	}
	return endpoint, cleanup
}

func buildArtifactServerScript(packages map[string][]string) string {
	var builder strings.Builder
	builder.WriteString("import os\n")
	builder.WriteString("root = \"/srv/repo\"\n")
	builder.WriteString("apt_path = os.path.join(root, \"dists\", \"dev\", \"main\", \"binary-amd64\")\n")
	builder.WriteString("os.makedirs(apt_path, exist_ok=True)\n")
	builder.WriteString("with open(os.path.join(apt_path, \"Packages\"), \"w\") as f:\n")
	var names []string
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		versions := packages[name]
		sort.Strings(versions)
		for _, version := range versions {
			builder.WriteString(fmt.Sprintf("    f.write(\"Package: %s\\nVersion: %s\\n\\n\")\n", name, version))
		}
	}
	builder.WriteString("os.execvp(\"python\", [\"python\", \"-m\", \"http.server\", \"8081\", \"--directory\", root])\n")
	return builder.String()
}

func startLocalPipIndexFromMap(ctx context.Context, t *testing.T, packages map[string][]string) (string, func()) {
	t.Helper()
	if len(packages) == 0 {
		t.Skip("no pip dependencies available to build local index")
	}
	root := t.TempDir()
	filesDir := filepath.Join(root, "files")
	simpleRoot := filepath.Join(root, "simple")
	require.NoError(t, os.MkdirAll(filesDir, 0755))
	require.NoError(t, os.MkdirAll(simpleRoot, 0755))

	wheels := map[string][]string{}
	var names []string
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		versions := packages[name]
		sort.Strings(versions)
		for _, version := range versions {
			wheel, err := buildDummyWheel(ctx, root, filesDir, name, version)
			require.NoError(t, err)
			wheels[name] = append(wheels[name], wheel)
		}
	}

	var indexBuilder strings.Builder
	for _, name := range names {
		indexBuilder.WriteString(fmt.Sprintf(`<a href="/simple/%s/">%s</a>`, name, name))
	}
	require.NoError(t, os.WriteFile(filepath.Join(simpleRoot, "index.html"), []byte(indexBuilder.String()), 0644))

	for _, name := range names {
		entries := wheels[name]
		sort.Strings(entries)
		pkgDir := filepath.Join(simpleRoot, name)
		require.NoError(t, os.MkdirAll(pkgDir, 0755))
		var pkgIndex strings.Builder
		for _, wheel := range entries {
			pkgIndex.WriteString(fmt.Sprintf(`<a href="/files/%s">%s</a>`, wheel, wheel))
		}
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "index.html"), []byte(pkgIndex.String()), 0644))
	}

	server := httptest.NewServer(http.FileServer(http.Dir(root)))
	cleanup := func() {
		server.Close()
	}
	return server.URL, cleanup
}

func buildDummyWheel(ctx context.Context, root string, filesDir string, name string, version string) (string, error) {
	moduleName := moduleNameFromPackageName(name)
	pkgRoot := filepath.Join(root, "src", fmt.Sprintf("%s-%s", name, version))
	if err := os.MkdirAll(filepath.Join(pkgRoot, moduleName), 0755); err != nil {
		return "", err
	}
	setup := fmt.Sprintf("from setuptools import setup\nsetup(name='%s', version='%s', packages=['%s'])\n", name, version, moduleName)
	if err := os.WriteFile(filepath.Join(pkgRoot, "setup.py"), []byte(setup), 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(pkgRoot, moduleName, "__init__.py"), []byte("__version__ = '"+version+"'\n"), 0644); err != nil {
		return "", err
	}

	before, err := listDirFiles(filesDir)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "python3", "-m", "pip", "wheel", "--no-deps", "--no-build-isolation", "-w", filesDir, pkgRoot)
	cmd.Env = append(os.Environ(), "PIP_DISABLE_PIP_VERSION_CHECK=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pip wheel failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	after, err := listDirFiles(filesDir)
	if err != nil {
		return "", err
	}
	for name := range after {
		if !strings.HasSuffix(name, ".whl") {
			continue
		}
		if _, existed := before[name]; existed {
			continue
		}
		return name, nil
	}
	return "", fmt.Errorf("no wheel produced for %s", name)
}

func listDirFiles(dir string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files[entry.Name()] = struct{}{}
	}
	return files, nil
}

func moduleNameFromPackageName(value string) string {
	normalized := normalizePipName(value)
	replacer := strings.NewReplacer("-", "_", ".", "_")
	return replacer.Replace(normalized)
}

func normalizePipName(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "-", ".", "-")
	return replacer.Replace(lower)
}

func mapKeys(values map[string][]string) []string {
	var keys []string
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
