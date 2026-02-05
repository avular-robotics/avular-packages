package adapters

import (
	"bufio"
	"compress/gzip"
	"context"
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
	aptIndex, err := buildAptIndex(ctx, aptSources, request.AptUser, request.AptAPIKey, request.AptWorkers, httpCfg)
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
	)
	if err != nil {
		return types.RepoIndexFile{}, err
	}
	return types.RepoIndexFile{
		Apt: aptIndex,
		Pip: pipIndexMap,
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

func buildAptIndex(ctx context.Context, sources []aptSource, user string, apiKey string, workerCount int, httpCfg httpRetryConfig) (map[string][]string, error) {
	if len(sources) == 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("apt sources are required")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	merged := map[string]map[string]struct{}{}
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
			index, err := buildAptIndexSingle(ctx, source, user, apiKey, httpCfg)
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
					merged[name] = map[string]struct{}{}
				}
				for _, version := range versions {
					merged[name][version] = struct{}{}
				}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return finalizeVersions(merged, sortDebVersions), nil
}

func buildAptIndexSingle(ctx context.Context, source aptSource, user string, apiKey string, httpCfg httpRetryConfig) (map[string][]string, error) {
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
	index, notFound, err := fetchAptPackages(ctx, gzURL, user, apiKey, httpCfg)
	if err != nil {
		return nil, err
	}
	if notFound {
		plainURL := fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", base, distribution, component, arch)
		index, _, err = fetchAptPackages(ctx, plainURL, user, apiKey, httpCfg)
		if err != nil {
			return nil, err
		}
	}
	return index, nil
}

func fetchAptPackages(ctx context.Context, url string, user string, apiKey string, httpCfg httpRetryConfig) (map[string][]string, bool, error) {
	resp, err := doRequest(ctx, url, user, apiKey, httpCfg)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, true, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to fetch apt packages").
			WithCause(fmt.Errorf("status=%d url=%s", resp.StatusCode, url))
	}
	var reader io.Reader = resp.Body
	if strings.HasSuffix(url, ".gz") || strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
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

func parseAptPackages(reader io.Reader) (map[string][]string, error) {
	packages := map[string]map[string]struct{}{}
	buffered := bufio.NewReader(reader)
	var name string
	var version string
	flush := func() {
		if name == "" || version == "" {
			return
		}
		if packages[name] == nil {
			packages[name] = map[string]struct{}{}
		}
		packages[name][version] = struct{}{}
	}
	for {
		line, err := buffered.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("failed to read apt packages").
				WithCause(err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			name = ""
			version = ""
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "Package:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Package:"))
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "Version:") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
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
	return finalizeVersions(packages, sortDebVersions), nil
}

func buildPipIndex(ctx context.Context, base string, user string, apiKey string, packages []string, maxPackages int, workerCount int, httpCfg httpRetryConfig) (map[string][]string, error) {
	simpleBase := normalizePipSimpleIndex(base)
	names := uniqueStrings(normalizePipNames(packages))
	if len(names) == 0 {
		list, err := fetchPipPackageNames(ctx, simpleBase, user, apiKey, httpCfg)
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
				versions, err := fetchPipPackageVersions(ctx, simpleBase, name, user, apiKey, httpCfg)
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

func fetchPipPackageNames(ctx context.Context, simpleBase string, user string, apiKey string, httpCfg httpRetryConfig) ([]string, error) {
	resp, err := doRequest(ctx, simpleBase, user, apiKey, httpCfg)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to fetch pip index").
			WithCause(fmt.Errorf("status=%d url=%s", resp.StatusCode, simpleBase))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to read pip index").
			WithCause(err)
	}
	names := parsePipSimpleNames(string(body))
	if len(names) == 0 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("pip index returned no packages")
	}
	return names, nil
}

func fetchPipPackageVersions(ctx context.Context, simpleBase string, name string, user string, apiKey string, httpCfg httpRetryConfig) ([]string, error) {
	url := strings.TrimRight(simpleBase, "/") + "/" + name + "/"
	resp, err := doRequest(ctx, url, user, apiKey, httpCfg)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to fetch pip package").
			WithCause(fmt.Errorf("status=%d url=%s", resp.StatusCode, url))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to read pip package index").
			WithCause(err)
	}
	versions := parsePipVersionsFromSimple(string(body))
	return sortPep440Versions(versions), nil
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
