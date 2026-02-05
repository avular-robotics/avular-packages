package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
)

type RepoSnapshotProGetAdapter struct {
	Endpoint       string
	Feed           string
	Component      string
	DebsDir        string
	Username       string
	APIKey         string
	SnapshotPrefix string
	Workers        int
	Timeout        time.Duration
	Retries        int
	RetryDelay     time.Duration
}

const defaultProgetUploadWorkers = 4
const defaultProgetUploadRetries = 3
const defaultProgetRetryDelay = 200 * time.Millisecond
const defaultProgetTimeout = 60 * time.Second
const maxProgetRetryDelay = 2 * time.Second

func NewRepoSnapshotProGetAdapter(endpoint string, feed string, component string, debsDir string, username string, apiKey string, snapshotPrefix string, workers int, timeoutSec int, retries int, retryDelayMs int) RepoSnapshotProGetAdapter {
	if component == "" {
		component = "main"
	}
	workers = normalizeProgetWorkers(workers)
	timeout := normalizeProgetTimeout(timeoutSec)
	retryCount := normalizeProgetRetries(retries)
	retryDelay := normalizeProgetRetryDelay(retryDelayMs)
	return RepoSnapshotProGetAdapter{
		Endpoint:       endpoint,
		Feed:           feed,
		Component:      component,
		DebsDir:        debsDir,
		Username:       username,
		APIKey:         apiKey,
		SnapshotPrefix: snapshotPrefix,
		Workers:        workers,
		Timeout:        timeout,
		Retries:        retryCount,
		RetryDelay:     retryDelay,
	}
}

func (a RepoSnapshotProGetAdapter) Publish(ctx context.Context, snapshotID string) error {
	if strings.TrimSpace(snapshotID) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("snapshot id is empty")
	}
	distribution := a.snapshotDistribution(snapshotID)
	return a.uploadDistribution(ctx, distribution)
}

func (a RepoSnapshotProGetAdapter) Promote(ctx context.Context, snapshotID string, channel string) error {
	target := strings.TrimSpace(channel)
	if target == "" {
		return nil
	}
	return a.uploadDistribution(ctx, target)
}

func (a RepoSnapshotProGetAdapter) uploadDistribution(ctx context.Context, distribution string) error {
	if strings.TrimSpace(a.Endpoint) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("proget endpoint is empty")
	}
	if strings.TrimSpace(a.Feed) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("proget feed is empty")
	}
	if strings.TrimSpace(distribution) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("proget distribution is empty")
	}
	if strings.TrimSpace(a.DebsDir) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("debs directory is empty")
	}
	debs, err := listDebs(a.DebsDir)
	if err != nil {
		return err
	}
	if len(debs) == 0 {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("no deb artifacts found")
	}
	return a.uploadDebsParallel(ctx, debs, distribution)
}

func (a RepoSnapshotProGetAdapter) uploadDebsParallel(ctx context.Context, debs []string, distribution string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var firstErr error
	workerCount := a.Workers
	if len(debs) < workerCount {
		workerCount = len(debs)
	}
	if workerCount == 0 {
		return nil
	}
	tasks := make(chan string)
	results := make(chan error, len(debs))
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for deb := range tasks {
				if ctx.Err() != nil {
					results <- ctx.Err()
					continue
				}
				results <- a.uploadDeb(ctx, deb, distribution)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	for _, deb := range debs {
		tasks <- deb
	}
	close(tasks)

	for err := range results {
		if err != nil && firstErr == nil {
			firstErr = err
			cancel()
		}
	}
	return firstErr
}

func (a RepoSnapshotProGetAdapter) uploadDeb(ctx context.Context, path string, distribution string) error {
	var lastErr error
	for attempt := 0; attempt < a.Retries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		retry, err := a.uploadDebOnce(ctx, path, distribution)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retry || attempt == a.Retries-1 {
			return err
		}
		time.Sleep(a.progetRetryDelay(attempt))
	}
	if lastErr == nil {
		lastErr = errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("proget upload failed")
	}
	return lastErr
}

func (a RepoSnapshotProGetAdapter) uploadDebOnce(ctx context.Context, path string, distribution string) (bool, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(a.Endpoint), "/")
	url := fmt.Sprintf("%s/debian/%s/upload/%s/%s", endpoint, a.Feed, distribution, a.Component)
	file, err := os.Open(path)
	if err != nil {
		return false, errbuilder.New().
			WithCode(errbuilder.CodeNotFound).
			WithMsg("failed to open deb artifact").
			WithCause(err)
	}
	defer file.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, file)
	if err != nil {
		return false, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to create proget request").
			WithCause(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if strings.TrimSpace(a.APIKey) != "" {
		user := strings.TrimSpace(a.Username)
		if user == "" {
			user = "api"
		}
		req.SetBasicAuth(user, a.APIKey)
	}
	client := &http.Client{Timeout: a.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return true, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("proget upload failed").
			WithCause(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	body, _ := io.ReadAll(resp.Body)
	message := strings.TrimSpace(string(body))
	lower := strings.ToLower(message)
	if (resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusBadRequest) && strings.Contains(lower, "already") {
		return false, nil
	}
	retry := resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests
	return retry, errbuilder.New().
		WithCode(errbuilder.CodeInternal).
		WithMsg("proget upload failed").
		WithCause(fmt.Errorf("status=%d url=%s response=%s", resp.StatusCode, url, message))
}

func (a RepoSnapshotProGetAdapter) progetRetryDelay(attempt int) time.Duration {
	delay := a.RetryDelay * time.Duration(1<<attempt)
	if delay > maxProgetRetryDelay {
		delay = maxProgetRetryDelay
	}
	jitter := time.Duration(time.Now().UnixNano() % int64(delay/2+1))
	return delay + jitter
}

func normalizeProgetWorkers(value int) int {
	if value <= 0 {
		return defaultProgetUploadWorkers
	}
	return value
}

func normalizeProgetTimeout(value int) time.Duration {
	timeout := time.Duration(value) * time.Second
	if timeout <= 0 {
		return defaultProgetTimeout
	}
	return timeout
}

func normalizeProgetRetries(value int) int {
	if value <= 0 {
		return defaultProgetUploadRetries
	}
	return value
}

func normalizeProgetRetryDelay(value int) time.Duration {
	delay := time.Duration(value) * time.Millisecond
	if delay <= 0 {
		return defaultProgetRetryDelay
	}
	return delay
}

func (a RepoSnapshotProGetAdapter) snapshotDistribution(snapshotID string) string {
	prefix := strings.TrimSpace(a.SnapshotPrefix)
	if prefix == "" {
		return snapshotID
	}
	trimmedPrefix := strings.TrimSuffix(prefix, "-")
	if trimmedPrefix != "" {
		if snapshotID == trimmedPrefix || strings.HasPrefix(snapshotID, trimmedPrefix+"-") {
			return snapshotID
		}
	}
	if strings.HasSuffix(prefix, "-") {
		return prefix + snapshotID
	}
	return fmt.Sprintf("%s-%s", prefix, snapshotID)
}

func listDebs(root string) ([]string, error) {
	var debs []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".deb") {
			debs = append(debs, path)
		}
		return nil
	}); err != nil {
		return nil, errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("failed to scan deb artifacts").
			WithCause(err)
	}
	return debs, nil
}

var _ ports.RepoSnapshotPort = RepoSnapshotProGetAdapter{}
