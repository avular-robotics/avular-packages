// Package shared provides common utility functions used across multiple
// packages in the avular-packages codebase.
package shared

import (
	"fmt"
	"strings"
)

// NormalizePipName lowercases a Python package name and replaces
// underscores and dots with hyphens, following PEP 503 normalization.
func NormalizePipName(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "-", ".", "-")
	return replacer.Replace(lower)
}

// HTTPStatusError creates a formatted error for non-2xx HTTP responses.
func HTTPStatusError(status int, url string) error {
	return fmt.Errorf("status=%d url=%s", status, url)
}

// HTTPStatusErrorWithBody creates a formatted error that includes the
// response body for non-2xx HTTP responses.
func HTTPStatusErrorWithBody(status int, url string, body string) error {
	return fmt.Errorf("status=%d url=%s response=%s", status, url, body)
}

// CommandError wraps a command execution error with its trimmed output
// for cleaner error messages.
func CommandError(output []byte, err error) error {
	return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
}
