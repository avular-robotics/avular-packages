package adapters

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/errbuilder-go"
	pep440 "github.com/aquasecurity/go-pep440-version"
	debversion "github.com/knqyf263/go-deb-version"
	"gopkg.in/yaml.v3"

	"avular-packages/internal/ports"
	"avular-packages/internal/types"
)

type RepoIndexBuilderAdapter struct{}

type RepoIndexWriterAdapter struct{}

type aptSource struct {
	Endpoint     string
	Distribution string
	Component    string
	Arch         string
}

const defaultAptFetchWorkers = 4
const defaultHTTPTimeout = 60 * time.Second
const defaultHTTPRetries = 3
const defaultHTTPRetryDelay = 200 * time.Millisecond
const maxHTTPRetryDelay = 2 * time.Second

type httpRetryConfig struct {
	timeout   time.Duration
	retries   int
	baseDelay time.Duration
}

type cacheConfig struct {
	dir string
	ttl time.Duration
}

func normalizeHTTPConfig(timeoutSec int, retries int, delayMs int) httpRetryConfig {
	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	retryCount := retries
	if retryCount <= 0 {
		retryCount = defaultHTTPRetries
	}
	baseDelay := time.Duration(delayMs) * time.Millisecond
	if baseDelay <= 0 {
		baseDelay = defaultHTTPRetryDelay
	}
	return httpRetryConfig{
		timeout:   timeout,
		retries:   retryCount,
		baseDelay: baseDelay,
	}
}

func normalizeCacheConfig(dir string, ttlMinutes int) cacheConfig {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" || ttlMinutes <= 0 {
		return cacheConfig{}
	}
	return cacheConfig{
		dir: trimmed,
		ttl: time.Duration(ttlMinutes) * time.Minute,
	}
}

func NewRepoIndexBuilderAdapter() RepoIndexBuilderAdapter {
	return RepoIndexBuilderAdapter{}
}

func NewRepoIndexWriterAdapter() RepoIndexWriterAdapter {
	return RepoIndexWriterAdapter{}
}

func (a RepoIndexBuilderAdapter) Build(ctx context.Context, request ports.RepoIndexBuildRequest) (types.RepoIndexFile, error) {
	pipIndex := strings.TrimSpace(request.PipIndex)
	if pipIndex == "" {
		return types.RepoIndexFile{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("pip index is required")
	}
	aptSources := resolveAptSources(
		request.AptSources,
		request.AptEndpoint,
		request.AptDistribution,
		request.AptComponent,
		request.AptArch,
	)
	httpCfg := normalizeHTTPConfig(request.HTTPTimeoutSec, request.HTTPRetries, request.HTTPRetryDelayMs)
	cacheCfg := normalizeCacheConfig(request.CacheDir, request.CacheTTLMinutes)
	aptVersions, aptPackages, err := buildAptIndex(ctx, aptSources, request.AptUser, request.AptAPIKey, request.AptWorkers, httpCfg, cacheCfg)
	if err != nil {
		return types.RepoIndexFile{}, err
	}
	pipIndexMap, err := buildPipIndex(
		ctx,
		pipIndex,
		request.PipUser,
		request.PipAPIKey,
		request.PipPackages,
		request.PipMax,
		request.PipWorkers,
		httpCfg,
		cacheCfg,
	)
	if err != nil {
		return types.RepoIndexFile{}, err
	}
	return types.RepoIndexFile{
		Apt:         aptVersions,
		AptPackages: aptPackages,
		Pip:         pipIndexMap,
	}, nil
}

func (a RepoIndexWriterAdapter) Write(path string, index types.RepoIndexFile) error {
	if strings.TrimSpace(path) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("output path is required")
	}
	data, err := yaml.Marshal(index)
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to marshal repo index").
			WithCause(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create repo index directory").
			WithCause(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write repo index").
			WithCause(err)
	}
	return nil
}

func buildAptIndex(ctx context.Context, sources []aptSource, user string, apiKey string, workerCount int, httpCfg httpRetryConfig, cacheCfg cacheConfig) (map[string][]string, map[string][]types.AptPackageVersion, error) {
	if len(sources) == 0 {
		return nil, nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("apt sources are required")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	merged := map[string]map[string]types.AptPackageVersion{}
	var mu sync.Mutex
	var errMu sync.Mutex
	var firstErr error
	if workerCount <= 0 {
		workerCount = defaultAptFetchWorkers
	}
	if len(sources) < workerCount {
		workerCount = len(sources)
	}
	sem := make(chan struct{}, workerCount)
	var wg sync.WaitGroup
	for _, source := range sources {
		source := source
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			index, err := buildAptIndexSingle(ctx, source, user, apiKey, httpCfg, cacheCfg)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
					cancel()
				}
				errMu.Unlock()
				return
			}
			mu.Lock()
			for name, versions := range index {
				if merged[name] == nil {
					merged[name] = map[string]types.AptPackageVersion{}
				}
				for version, metadata := range versions {
					if _, ok := merged[name][version]; ok {
						continue
					}
					merged[name][version] = metadata
				}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, nil, firstErr
	}
	versions, packages := finalizeAptPackages(merged)
	return versions, packages, nil
}

func buildAptIndexSingle(ctx context.Context, source aptSource, user string, apiKey string, httpCfg httpRetryConfig, cacheCfg cacheConfig) (map[string]map[string]types.AptPackageVersion, error) {
	base := strings.TrimRight(strings.TrimSpace(source.Endpoint), "/")
	component := strings.TrimSpace(source.Component)
	if component == "" {
		component = "main"
	}
	arch := strings.TrimSpace(source.Arch)
	if arch == "" {
		arch = "amd64"
	}
	distribution := strings.TrimSpace(source.Distribution)
	if distribution == "" {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("apt distribution is required")
	}
	gzURL := fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages.gz", base, distribution, component, arch)
	index, notFound, err := fetchAptPackages(ctx, gzURL, user, apiKey, httpCfg, cacheCfg)
	if err != nil {
		return nil, err
	}
	if notFound {
		plainURL := fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", base, distribution, component, arch)
		index, _, err = fetchAptPackages(ctx, plainURL, user, apiKey, httpCfg, cacheCfg)
		if err != nil {
			return nil, err
		}
	}
	return index, nil
}

func fetchAptPackages(ctx context.Context, url string, user string, apiKey string, httpCfg httpRetryConfig, cacheCfg cacheConfig) (map[string]map[string]types.AptPackageVersion, bool, error) {
	status, body, header, err := fetchURL(ctx, url, user, apiKey, httpCfg, cacheCfg)
	if err != nil {
		return nil, false, err
	}
	if status == http.StatusNotFound {
		return nil, true, nil
	}
	if status < 200 || status >= 300 {
		return nil, false, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to fetch apt packages").
			WithCause(fmt.Errorf("status=%d url=%s", status, url))
	}
	var reader io.Reader = bytes.NewReader(body)
	if isGzipContent(url, body, header) {
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, false, errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("failed to read gzipped apt packages").
				WithCause(err)
		}
		defer gz.Close()
		reader = gz
	}
	index, err := parseAptPackages(reader)
	if err != nil {
		return nil, false, err
	}
	return index, false, nil
}

func parseAptPackages(reader io.Reader) (map[string]map[string]types.AptPackageVersion, error) {
	packages := map[string]map[string]types.AptPackageVersion{}
	buffered := bufio.NewReader(reader)
	var name string
	var version string
	var dependsRaw string
	var preDependsRaw string
	var providesRaw string
	var lastField string
	flush := func() {
		if name == "" || version == "" {
			return
		}
		if packages[name] == nil {
			packages[name] = map[string]types.AptPackageVersion{}
		}
		packages[name][version] = types.AptPackageVersion{
			Version:    version,
			Depends:    parseAptDependencyField(dependsRaw),
			PreDepends: parseAptDependencyField(preDependsRaw),
			Provides:   parseAptDependencyField(providesRaw),
		}
	}
	for {
		line, err := buffered.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("failed to read apt packages").
				WithCause(err)
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) == "" {
			flush()
			name = ""
			version = ""
			dependsRaw = ""
			preDependsRaw = ""
			providesRaw = ""
			lastField = ""
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continued := strings.TrimSpace(line)
			switch lastField {
			case "Depends":
				dependsRaw = joinField(dependsRaw, continued)
			case "Pre-Depends":
				preDependsRaw = joinField(preDependsRaw, continued)
			case "Provides":
				providesRaw = joinField(providesRaw, continued)
			}
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "Package:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Package:"))
			lastField = "Package"
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "Version:") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
			lastField = "Version"
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "Depends:") {
			dependsRaw = strings.TrimSpace(strings.TrimPrefix(line, "Depends:"))
			lastField = "Depends"
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "Pre-Depends:") {
			preDependsRaw = strings.TrimSpace(strings.TrimPrefix(line, "Pre-Depends:"))
			lastField = "Pre-Depends"
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "Provides:") {
			providesRaw = strings.TrimSpace(strings.TrimPrefix(line, "Provides:"))
			lastField = "Provides"
			if err == io.EOF {
				break
			}
			continue
		}
		if err == io.EOF {
			break
		}
	}
	flush()
	return packages, nil
}

func buildPipIndex(ctx context.Context, base string, user string, apiKey string, packages []string, maxPackages int, workerCount int, httpCfg httpRetryConfig, cacheCfg cacheConfig) (map[string][]string, error) {
	simpleBase := normalizePipSimpleIndex(base)
	names := uniqueStrings(normalizePipNames(packages))
	if len(names) == 0 {
		list, err := fetchPipPackageNames(ctx, simpleBase, user, apiKey, httpCfg, cacheCfg)
		if err != nil {
			return nil, err
		}
		names = list
	}
	if maxPackages > 0 && len(names) > maxPackages {
		names = names[:maxPackages]
	}
	index := map[string][]string{}
	if len(names) == 0 {
		return index, nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if workerCount <= 0 {
		workerCount = 8
	}
	if len(names) < workerCount {
		workerCount = len(names)
	}
	type pipResult struct {
		name     string
		versions []string
		err      error
	}
	tasks := make(chan string)
	results := make(chan pipResult, len(names))
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range tasks {
				if ctx.Err() != nil {
					results <- pipResult{name: name, versions: nil, err: ctx.Err()}
					continue
				}
				versions, err := fetchPipPackageVersions(ctx, simpleBase, name, user, apiKey, httpCfg, cacheCfg)
				results <- pipResult{name: name, versions: versions, err: err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	for _, name := range names {
		if ctx.Err() != nil {
			break
		}
		tasks <- name
	}
	close(tasks)

	var firstErr error
	for result := range results {
		if result.err != nil && firstErr == nil {
			firstErr = result.err
			cancel()
		}
		if result.err == nil && len(result.versions) > 0 {
			index[result.name] = result.versions
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return index, nil
}

func fetchPipPackageNames(ctx context.Context, simpleBase string, user string, apiKey string, httpCfg httpRetryConfig, cacheCfg cacheConfig) ([]string, error) {
	status, body, _, err := fetchURL(ctx, simpleBase, user, apiKey, httpCfg, cacheCfg)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to fetch pip index").
			WithCause(fmt.Errorf("status=%d url=%s", status, simpleBase))
	}
	names := parsePipSimpleNames(string(body))
	if len(names) == 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("pip index returned no packages")
	}
	return names, nil
}

func fetchPipPackageVersions(ctx context.Context, simpleBase string, name string, user string, apiKey string, httpCfg httpRetryConfig, cacheCfg cacheConfig) ([]string, error) {
	url := strings.TrimRight(simpleBase, "/") + "/" + name + "/"
	status, body, _, err := fetchURL(ctx, url, user, apiKey, httpCfg, cacheCfg)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status < 200 || status >= 300 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to fetch pip package").
			WithCause(fmt.Errorf("status=%d url=%s", status, url))
	}
	versions := parsePipVersionsFromSimple(string(body))
	return sortPep440Versions(versions), nil
}

func fetchURL(ctx context.Context, url string, user string, apiKey string, httpCfg httpRetryConfig, cacheCfg cacheConfig) (int, []byte, http.Header, error) {
	if cacheCfg.dir != "" && cacheCfg.ttl > 0 {
		key := cacheKey(url, user, apiKey)
		if payload, ok, err := readCache(cacheCfg, key); err != nil {
			return 0, nil, nil, err
		} else if ok {
			return http.StatusOK, payload, http.Header{}, nil
		}
	}
	resp, err := doRequest(ctx, url, user, apiKey, httpCfg)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to read response body").
			WithCause(err)
	}
	if cacheCfg.dir != "" && cacheCfg.ttl > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		key := cacheKey(url, user, apiKey)
		_ = writeCache(cacheCfg, key, payload)
	}
	return resp.StatusCode, payload, resp.Header, nil
}

func isGzipContent(url string, data []byte, header http.Header) bool {
	if strings.HasSuffix(url, ".gz") {
		return true
	}
	if header != nil && strings.EqualFold(header.Get("Content-Encoding"), "gzip") {
		return true
	}
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}

func cacheKey(url string, user string, apiKey string) string {
	sum := sha256.Sum256([]byte(url + "|" + user + "|" + apiKey))
	return hex.EncodeToString(sum[:])
}

func readCache(cfg cacheConfig, key string) ([]byte, bool, error) {
	if cfg.dir == "" || cfg.ttl <= 0 {
		return nil, false, nil
	}
	path := filepath.Join(cfg.dir, key+".cache")
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to stat cache file").
			WithCause(err)
	}
	if time.Since(info.ModTime()) > cfg.ttl {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to read cache file").
			WithCause(err)
	}
	return data, true, nil
}

func writeCache(cfg cacheConfig, key string, data []byte) error {
	if cfg.dir == "" || cfg.ttl <= 0 {
		return nil
	}
	if err := os.MkdirAll(cfg.dir, 0755); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create cache directory").
			WithCause(err)
	}
	path := filepath.Join(cfg.dir, key+".cache")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to write cache file").
			WithCause(err)
	}
	return nil
}

func normalizePipSimpleIndex(base string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(trimmed, "/simple") {
		return trimmed + "/"
	}
	return trimmed + "/simple/"
}

func parsePipSimpleNames(content string) []string {
	re := regexp.MustCompile(`(?is)<a[^>]*>([^<]+)</a>`)
	matches := re.FindAllStringSubmatch(content, -1)
	var names []string
	for _, match := range matches {
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		names = append(names, normalizePipName(name))
	}
	sort.Strings(names)
	return uniqueStrings(names)
}

func parsePipVersionsFromSimple(content string) []string {
	re := regexp.MustCompile(`href=["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(content, -1)
	versions := map[string]struct{}{}
	for _, match := range matches {
		raw := strings.Split(match[1], "#")[0]
		raw = strings.Split(raw, "?")[0]
		filename := filepath.Base(raw)
		version := parsePipVersionFromFilename(filename)
		if version == "" {
			continue
		}
		if _, err := pep440.Parse(version); err != nil {
			continue
		}
		versions[version] = struct{}{}
	}
	return mapKeys(versions)
}

func parsePipVersionFromFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return ""
	}
	wheel := regexp.MustCompile(`^(.+?)-([0-9][^-]*)(?:-[^-]+)?-[^-]+-[^-]+-[^-]+\.whl$`)
	if match := wheel.FindStringSubmatch(filename); len(match) == 3 {
		return match[2]
	}
	sdist := regexp.MustCompile(`^(.+?)-([0-9][^-]*)\.(?:tar\.gz|zip|tar\.bz2|tar\.xz|tgz)$`)
	if match := sdist.FindStringSubmatch(filename); len(match) == 3 {
		return match[2]
	}
	return ""
}

func normalizePipNames(values []string) []string {
	var out []string
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		out = append(out, normalizePipName(name))
	}
	return out
}

func sortDebVersions(versions []string) []string {
	sort.Slice(versions, func(i, j int) bool {
		vi, err := debversion.NewVersion(versions[i])
		if err != nil {
			return versions[i] < versions[j]
		}
		vj, err := debversion.NewVersion(versions[j])
		if err != nil {
			return versions[i] < versions[j]
		}
		return vi.Compare(vj) < 0
	})
	return versions
}

func sortPep440Versions(versions []string) []string {
	sort.Slice(versions, func(i, j int) bool {
		vi, err := pep440.Parse(versions[i])
		if err != nil {
			return versions[i] < versions[j]
		}
		vj, err := pep440.Parse(versions[j])
		if err != nil {
			return versions[i] < versions[j]
		}
		return vi.Compare(vj) < 0
	})
	return versions
}

func finalizeAptPackages(raw map[string]map[string]types.AptPackageVersion) (map[string][]string, map[string][]types.AptPackageVersion) {
	versionIndex := map[string][]string{}
	packageIndex := map[string][]types.AptPackageVersion{}
	for name, versions := range raw {
		keys := make([]string, 0, len(versions))
		for version := range versions {
			keys = append(keys, version)
		}
		keys = sortDebVersions(keys)
		versionIndex[name] = keys
		entries := make([]types.AptPackageVersion, 0, len(keys))
		for _, version := range keys {
			entry := versions[version]
			if entry.Version == "" {
				entry.Version = version
			}
			entries = append(entries, entry)
		}
		packageIndex[name] = entries
	}
	return versionIndex, packageIndex
}

func finalizeVersions(raw map[string]map[string]struct{}, sorter func([]string) []string) map[string][]string {
	out := map[string][]string{}
	for name, versions := range raw {
		list := mapKeys(versions)
		out[name] = sorter(list)
	}
	return out
}

func mapKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func parseAptDependencyField(value string) []string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var out []string
	for _, part := range parts {
		entry := strings.TrimSpace(part)
		if entry == "" {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func joinField(current string, next string) string {
	if current == "" {
		return next
	}
	if next == "" {
		return current
	}
	return current + " " + next
}

func resolveAptSources(values []string, endpoint string, distribution string, component string, arch string) []aptSource {
	var sources []aptSource
	for _, raw := range values {
		source, err := parseAptSource(raw)
		if err != nil {
			continue
		}
		sources = append(sources, source)
	}
	if len(sources) == 0 && strings.TrimSpace(endpoint) != "" {
		sources = append(sources, aptSource{
			Endpoint:     endpoint,
			Distribution: distribution,
			Component:    component,
			Arch:         arch,
		})
	}
	return sources
}

func parseAptSource(value string) (aptSource, error) {
	parts := strings.Split(value, "|")
	if len(parts) < 2 {
		return aptSource{}, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("invalid apt source entry")
	}
	source := aptSource{
		Endpoint:     strings.TrimSpace(parts[0]),
		Distribution: strings.TrimSpace(parts[1]),
	}
	if len(parts) > 2 {
		source.Component = strings.TrimSpace(parts[2])
	}
	if len(parts) > 3 {
		source.Arch = strings.TrimSpace(parts[3])
	}
	return source, nil
}

func doRequest(ctx context.Context, url string, user string, apiKey string, cfg httpRetryConfig) (*http.Response, error) {
	client := &http.Client{Timeout: cfg.timeout}
	var lastErr error
	for attempt := 0; attempt < cfg.retries; attempt++ {
		if ctx.Err() != nil {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("request canceled").
				WithCause(ctx.Err())
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("failed to create request").
				WithCause(err)
		}
		if strings.TrimSpace(apiKey) != "" {
			authUser := strings.TrimSpace(user)
			if authUser == "" {
				authUser = "api"
			}
			req.SetBasicAuth(authUser, apiKey)
		}
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, errbuilder.New().
					WithCode(errbuilder.CodeInternal).
					WithMsg("request canceled").
					WithCause(ctx.Err())
			}
			lastErr = err
			if attempt < cfg.retries-1 {
				time.Sleep(httpRetryDelay(attempt, cfg))
				continue
			}
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("request failed").
				WithCause(err)
		}
		if (resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests) && attempt < cfg.retries-1 {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			time.Sleep(httpRetryDelay(attempt, cfg))
			continue
		}
		return resp, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("request failed")
	}
	return nil, errbuilder.New().
		WithCode(errbuilder.CodeInternal).
		WithMsg("request failed").
		WithCause(lastErr)
}

func httpRetryDelay(attempt int, cfg httpRetryConfig) time.Duration {
	delay := cfg.baseDelay * time.Duration(1<<attempt)
	if delay > maxHTTPRetryDelay {
		delay = maxHTTPRetryDelay
	}
	jitter := time.Duration(time.Now().UnixNano() % int64(delay/2+1))
	return delay + jitter
}

var _ ports.RepoIndexBuilderPort = RepoIndexBuilderAdapter{}
var _ ports.RepoIndexWriterPort = RepoIndexWriterAdapter{}
