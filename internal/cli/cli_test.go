package cli

import (
	"testing"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- Command tree tests ----------

func TestRootCommandHasSubcommands(t *testing.T) {
	root := newRootCommand()
	names := make([]string, 0, len(root.Commands()))
	for _, cmd := range root.Commands() {
		names = append(names, cmd.Name())
	}
	expected := []string{
		"validate", "resolve", "lock", "build",
		"publish", "inspect", "repo-index", "prune",
	}
	for _, name := range expected {
		assert.Contains(t, names, name, "missing subcommand: %s", name)
	}
}

func TestRootCommandVersion(t *testing.T) {
	root := newRootCommand()
	assert.Equal(t, "dev", root.Version)
}

func TestResolveCommandFlags(t *testing.T) {
	cmd := newResolveCommand()
	flags := []string{
		"product", "profile", "workspace", "repo-index",
		"output", "snapshot-id", "target-ubuntu", "schema",
		"compat-get-dependencies", "compat-rosdep",
		"apt-preferences", "apt-install-list",
		"snapshot-apt-sources", "snapshot-apt-base-url",
		"snapshot-apt-component", "snapshot-apt-arch",
		"apt-sat-solver",
	}
	for _, name := range flags {
		flag := cmd.Flags().Lookup(name)
		assert.NotNil(t, flag, "missing flag: %s", name)
	}
}

func TestPublishCommandFlags(t *testing.T) {
	cmd := newPublishCommand()
	flags := []string{
		"output", "repo-dir", "sbom", "repo-backend",
		"debs-dir", "aptly-repo", "aptly-component",
		"aptly-prefix", "aptly-endpoint", "gpg-key",
		"proget-endpoint", "proget-feed", "proget-component",
		"proget-user", "proget-api-key", "proget-workers",
		"proget-timeout", "proget-retries", "proget-retry-delay-ms",
	}
	for _, name := range flags {
		flag := cmd.Flags().Lookup(name)
		assert.NotNil(t, flag, "missing flag: %s", name)
	}
}

func TestValidateCommandFlags(t *testing.T) {
	cmd := newValidateCommand()
	assert.NotNil(t, cmd.Flags().Lookup("product"))
	assert.NotNil(t, cmd.Flags().Lookup("profile"))
}

// ---------- Helper function tests ----------

func TestResolveString(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *cobra.Command
		value    string
		expected string
	}{
		{
			name:     "nil cmd with value returns value",
			cmd:      nil,
			value:    "explicit",
			expected: "explicit",
		},
		{
			name:     "nil cmd empty value returns empty",
			cmd:      nil,
			value:    "",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveString(tt.cmd, tt.value, "test_key", "test-flag")
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveStrings(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *cobra.Command
		values   []string
		expected []string
	}{
		{
			name:     "nil cmd with values returns values",
			cmd:      nil,
			values:   []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "nil cmd empty returns nil",
			cmd:      nil,
			values:   nil,
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveStrings(tt.cmd, tt.values, "test_key", "test-flag")
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveBool(t *testing.T) {
	got := resolveBool(nil, true, "test_key", "test-flag")
	assert.True(t, got)

	got = resolveBool(nil, false, "test_key", "test-flag")
	assert.False(t, got)
}

func TestResolveInt(t *testing.T) {
	got := resolveInt(nil, 42, "test_key", "test-flag")
	assert.Equal(t, 42, got)
}

func TestFlagChanged(t *testing.T) {
	assert.False(t, flagChanged(nil, "anything"), "nil cmd should return false")
	assert.False(t, flagChanged(nil, ""), "nil cmd with empty name")

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("myflag", "", "test flag")
	assert.False(t, flagChanged(cmd, "myflag"), "unchanged flag")
	assert.False(t, flagChanged(cmd, "nonexistent"), "nonexistent flag")
}

func TestFlagChangedAfterSet(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("myflag", "", "test flag")
	require.NoError(t, cmd.Flags().Set("myflag", "val"))
	assert.True(t, flagChanged(cmd, "myflag"))
}

// ---------- Exit code tests ----------

func TestExitCodeForError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name: "invalid argument",
			err: errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("bad input"),
			expected: 2,
		},
		{
			name: "already exists",
			err: errbuilder.New().
				WithCode(errbuilder.CodeAlreadyExists).
				WithMsg("dup"),
			expected: 2,
		},
		{
			name: "conflict without resolution",
			err: errbuilder.New().
				WithCode(errbuilder.CodeFailedPrecondition).
				WithMsg("conflict without resolution directive: libfoo"),
			expected: 3,
		},
		{
			name: "no compatible version",
			err: errbuilder.New().
				WithCode(errbuilder.CodeFailedPrecondition).
				WithMsg("no compatible version for libfoo"),
			expected: 4,
		},
		{
			name: "generic failed precondition",
			err: errbuilder.New().
				WithCode(errbuilder.CodeFailedPrecondition).
				WithMsg("something else failed"),
			expected: 4,
		},
		{
			name: "permission denied",
			err: errbuilder.New().
				WithCode(errbuilder.CodePermissionDenied).
				WithMsg("nope"),
			expected: 3,
		},
		{
			name: "not found no available versions",
			err: errbuilder.New().
				WithCode(errbuilder.CodeNotFound).
				WithMsg("no available versions for libbar"),
			expected: 4,
		},
		{
			name: "not found generic",
			err: errbuilder.New().
				WithCode(errbuilder.CodeNotFound).
				WithMsg("file missing"),
			expected: 5,
		},
		{
			name: "internal error",
			err: errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("boom"),
			expected: 5,
		},
		{
			name:     "unknown error",
			err:      assert.AnError,
			expected: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exitCodeForError(tt.err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name: "errbuilder with msg",
			err: errbuilder.New().
				WithCode(errbuilder.CodeInternal).
				WithMsg("something broke"),
			expected: "something broke",
		},
		{
			name:     "plain error",
			err:      assert.AnError,
			expected: assert.AnError.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorMessage(tt.err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
