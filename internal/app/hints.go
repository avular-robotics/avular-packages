package app

import (
	"fmt"
	"os"
	"strings"

	"avular-packages/internal/types"
)

// defaultsHint pairs a flag name with a spec defaults key for hint messages.
type defaultsHint struct {
	FlagName    string
	DefaultsKey string
}

// checkResolveDefaultsHints returns hints for resolve flags that could
// be replaced by spec defaults.  A hint is generated when the user
// explicitly provided a value that matches a non-empty default.
func checkResolveDefaultsHints(req ResolveRequest, defaults types.SpecDefaults) []string {
	checks := []struct {
		hint       defaultsHint
		provided   bool
		hasDefault bool
	}{
		{
			hint:       defaultsHint{"--target-ubuntu", "defaults.target_ubuntu"},
			provided:   strings.TrimSpace(req.TargetUbuntu) != "",
			hasDefault: defaults.TargetUbuntu != "",
		},
		{
			hint:       defaultsHint{"--workspace", "defaults.workspace"},
			provided:   len(req.Workspace) > 0,
			hasDefault: len(defaults.Workspace) > 0,
		},
		{
			hint:       defaultsHint{"--repo-index", "defaults.repo_index"},
			provided:   strings.TrimSpace(req.RepoIndex) != "",
			hasDefault: defaults.RepoIndex != "",
		},
		{
			hint:       defaultsHint{"--output", "defaults.output"},
			provided:   strings.TrimSpace(req.OutputDir) != "",
			hasDefault: defaults.Output != "",
		},
	}

	var hints []string
	for _, c := range checks {
		if c.provided && c.hasDefault {
			hints = append(hints, fmt.Sprintf(
				"hint: %s is also set in product spec (%s); you can omit the flag",
				c.hint.FlagName, c.hint.DefaultsKey,
			))
		}
	}
	return hints
}

// checkBuildDefaultsHints returns hints for build-specific flags that
// could be replaced by spec defaults.
func checkBuildDefaultsHints(req BuildRequest, defaults types.SpecDefaults) []string {
	// Start with the common resolve-level hints
	resolveReq := ResolveRequest{
		TargetUbuntu: req.TargetUbuntu,
		Workspace:    req.Workspace,
		RepoIndex:    req.RepoIndex,
		OutputDir:    req.OutputDir,
	}
	hints := checkResolveDefaultsHints(resolveReq, defaults)

	// Build-specific checks
	buildChecks := []struct {
		hint       defaultsHint
		provided   bool
		hasDefault bool
	}{
		{
			hint:       defaultsHint{"--pip-index-url", "defaults.pip_index_url"},
			provided:   strings.TrimSpace(req.PipIndexURL) != "",
			hasDefault: defaults.PipIndexURL != "",
		},
		{
			hint:       defaultsHint{"--internal-deb-dir", "defaults.internal_deb_dir"},
			provided:   strings.TrimSpace(req.InternalDebDir) != "",
			hasDefault: defaults.InternalDebDir != "",
		},
		{
			hint:       defaultsHint{"--internal-src", "defaults.internal_src"},
			provided:   len(req.InternalSrc) > 0,
			hasDefault: len(defaults.InternalSrc) > 0,
		},
	}

	for _, c := range buildChecks {
		if c.provided && c.hasDefault {
			hints = append(hints, fmt.Sprintf(
				"hint: %s is also set in product spec (%s); you can omit the flag",
				c.hint.FlagName, c.hint.DefaultsKey,
			))
		}
	}
	return hints
}

// emitHints writes hint messages to stderr.
func emitHints(hints []string) {
	for _, h := range hints {
		fmt.Fprintln(os.Stderr, h)
	}
}
