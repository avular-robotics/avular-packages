package adapters

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"avular-packages/internal/ports"
)

type RepoSnapshotAptlyAdapter struct {
	RepoName     string
	Distribution string
	Component    string
	DebsDir      string
	Prefix       string
	Endpoint     string
	GpgKey       string
}

func NewRepoSnapshotAptlyAdapter(repoName string, distribution string, component string, debsDir string, prefix string, endpoint string, gpgKey string) RepoSnapshotAptlyAdapter {
	if component == "" {
		component = "main"
	}
	if prefix == "" {
		prefix = "."
	}
	return RepoSnapshotAptlyAdapter{
		RepoName:     repoName,
		Distribution: distribution,
		Component:    component,
		DebsDir:      debsDir,
		Prefix:       prefix,
		Endpoint:     endpoint,
		GpgKey:       gpgKey,
	}
}

func (a RepoSnapshotAptlyAdapter) Publish(ctx context.Context, snapshotID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(a.RepoName) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("aptly repo name is empty")
	}
	if strings.TrimSpace(a.Distribution) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("aptly distribution is empty")
	}
	if strings.TrimSpace(a.DebsDir) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("debs directory is empty")
	}
	if strings.TrimSpace(snapshotID) == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("snapshot id is empty")
	}

	if err := a.ensureRepo(ctx); err != nil {
		return err
	}
	if err := a.runAptly(ctx, "repo", "add", a.RepoName, a.DebsDir); err != nil {
		return err
	}
	if err := a.runAptly(ctx, "snapshot", "create", snapshotID, "from", "repo", a.RepoName); err != nil {
		return err
	}
	return nil
}

func (a RepoSnapshotAptlyAdapter) Promote(ctx context.Context, snapshotID string, channel string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	distribution := strings.TrimSpace(channel)
	if distribution == "" {
		distribution = a.Distribution
	}
	if distribution == "" {
		return errbuilder.New().
			WithCode(errbuilder.CodeInvalidArgument).
			WithMsg("distribution is empty for publish")
	}
	prefix := a.Prefix
	if prefix == "" {
		prefix = "."
	}
	identifier := fmt.Sprintf("%s %s", prefix, distribution)
	listRaw, err := a.runAptlyOutput(ctx, "publish", "list", "-raw")
	if err != nil {
		return err
	}
	published := containsLine(listRaw, identifier)
	if published {
		args := []string{"publish", "switch", distribution}
		if a.Endpoint != "" {
			args = append(args, fmt.Sprintf("%s:%s", a.Endpoint, prefix))
		} else {
			args = append(args, prefix)
		}
		args = append(args, snapshotID)
		if a.GpgKey != "" {
			args = append(args, "-gpg-key", a.GpgKey)
		}
		return a.runAptly(ctx, args...)
	}

	args := []string{"publish", "snapshot", snapshotID}
	if a.Endpoint != "" {
		args = append(args, fmt.Sprintf("%s:%s", a.Endpoint, prefix))
	} else {
		args = append(args, prefix)
	}
	args = append(args, "-distribution", distribution, "-component", a.Component)
	if a.GpgKey != "" {
		args = append(args, "-gpg-key", a.GpgKey)
	}
	return a.runAptly(ctx, args...)
}

func (a RepoSnapshotAptlyAdapter) ensureRepo(ctx context.Context) error {
	if err := a.runAptly(ctx, "repo", "show", a.RepoName); err == nil {
		return nil
	}
	args := []string{"repo", "create", "-distribution", a.Distribution, "-component", a.Component, a.RepoName}
	return a.runAptly(ctx, args...)
}

func (a RepoSnapshotAptlyAdapter) runAptly(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "aptly", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("aptly command failed").
			WithCause(fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err))
	}
	return nil
}

func (a RepoSnapshotAptlyAdapter) runAptlyOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "aptly", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errbuilder.New().
			WithCode(errbuilder.CodeInternal).
			WithMsg("aptly command failed").
			WithCause(fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err))
	}
	return string(output), nil
}

func containsLine(content string, match string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == strings.TrimSpace(match) {
			return true
		}
	}
	return false
}

var _ ports.RepoSnapshotPort = RepoSnapshotAptlyAdapter{}
