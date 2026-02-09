package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"avular-packages/internal/app"
)

type pruneOptions struct {
	RepoBackend      string
	RepoDir          string
	KeepLast         int
	KeepDays         int
	ProtectChannels  []string
	ProtectPrefixes  []string
	DryRun           bool
	ProGetEndpoint   string
	ProGetFeed       string
	ProGetComponent  string
	ProGetUser       string
	ProGetAPIKey     string
	ProGetTimeoutSec int
	ProGetRetries    int
	ProGetRetryDelay int
}

func newPruneCommand() *cobra.Command {
	opts := pruneOptions{}
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune snapshot distributions based on retention policy",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPrune(cmd.Context(), cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.RepoBackend, "repo-backend", "file", "Repository backend (file, aptly, or proget)")
	cmd.Flags().StringVar(&opts.RepoDir, "repo-dir", "", "Repository directory for file backend")
	cmd.Flags().IntVar(&opts.KeepLast, "keep-last", 0, "Keep last N snapshots per group")
	cmd.Flags().IntVar(&opts.KeepDays, "keep-days", 0, "Keep snapshots newer than N days")
	cmd.Flags().StringSliceVar(&opts.ProtectChannels, "protect-channel", nil, "Protect channel distributions from pruning")
	cmd.Flags().StringSliceVar(&opts.ProtectPrefixes, "protect-prefix", nil, "Protect snapshot prefixes from pruning")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", true, "Only report prune actions without deleting")
	cmd.Flags().StringVar(&opts.ProGetEndpoint, "proget-endpoint", "", "ProGet base URL (e.g., https://packages.example.com)")
	cmd.Flags().StringVar(&opts.ProGetFeed, "proget-feed", "", "ProGet Debian feed name")
	cmd.Flags().StringVar(&opts.ProGetComponent, "proget-component", "main", "ProGet Debian component name")
	cmd.Flags().StringVar(&opts.ProGetUser, "proget-user", "", "ProGet username for basic auth (defaults to api)")
	cmd.Flags().StringVar(&opts.ProGetAPIKey, "proget-api-key", "", "ProGet API key or password for basic auth")
	cmd.Flags().IntVar(&opts.ProGetTimeoutSec, "proget-timeout", 60, "ProGet HTTP timeout in seconds (0 = default)")
	cmd.Flags().IntVar(&opts.ProGetRetries, "proget-retries", 3, "ProGet API retries (0 = default)")
	cmd.Flags().IntVar(&opts.ProGetRetryDelay, "proget-retry-delay-ms", 200, "ProGet retry base delay in ms (0 = default)")

	_ = viper.BindPFlag("repo_backend", cmd.Flags().Lookup("repo-backend"))
	_ = viper.BindPFlag("repo_dir", cmd.Flags().Lookup("repo-dir"))
	_ = viper.BindPFlag("keep_last", cmd.Flags().Lookup("keep-last"))
	_ = viper.BindPFlag("keep_days", cmd.Flags().Lookup("keep-days"))
	_ = viper.BindPFlag("protect_channels", cmd.Flags().Lookup("protect-channel"))
	_ = viper.BindPFlag("protect_prefixes", cmd.Flags().Lookup("protect-prefix"))
	_ = viper.BindPFlag("dry_run", cmd.Flags().Lookup("dry-run"))
	_ = viper.BindPFlag("proget_endpoint", cmd.Flags().Lookup("proget-endpoint"))
	_ = viper.BindPFlag("proget_feed", cmd.Flags().Lookup("proget-feed"))
	_ = viper.BindPFlag("proget_component", cmd.Flags().Lookup("proget-component"))
	_ = viper.BindPFlag("proget_user", cmd.Flags().Lookup("proget-user"))
	_ = viper.BindPFlag("proget_api_key", cmd.Flags().Lookup("proget-api-key"))
	_ = viper.BindPFlag("proget_timeout_sec", cmd.Flags().Lookup("proget-timeout"))
	_ = viper.BindPFlag("proget_retries", cmd.Flags().Lookup("proget-retries"))
	_ = viper.BindPFlag("proget_retry_delay_ms", cmd.Flags().Lookup("proget-retry-delay-ms"))

	return cmd
}

func runPrune(ctx context.Context, cmd *cobra.Command, opts pruneOptions) error {
	service := newAppService()
	result, err := service.PruneSnapshots(ctx, app.PruneRequest{
		RepoBackend:        resolveString(cmd, opts.RepoBackend, "repo_backend", "repo-backend"),
		RepoDir:            resolveString(cmd, opts.RepoDir, "repo_dir", "repo-dir"),
		KeepLast:           resolveInt(cmd, opts.KeepLast, "keep_last", "keep-last"),
		KeepDays:           resolveInt(cmd, opts.KeepDays, "keep_days", "keep-days"),
		ProtectChannels:    resolveStrings(cmd, opts.ProtectChannels, "protect_channels", "protect-channel"),
		ProtectPrefixes:    resolveStrings(cmd, opts.ProtectPrefixes, "protect_prefixes", "protect-prefix"),
		DryRun:             resolveBool(cmd, opts.DryRun, "dry_run", "dry-run"),
		ProGetEndpoint:     resolveString(cmd, opts.ProGetEndpoint, "proget_endpoint", "proget-endpoint"),
		ProGetFeed:         resolveString(cmd, opts.ProGetFeed, "proget_feed", "proget-feed"),
		ProGetComponent:    resolveString(cmd, opts.ProGetComponent, "proget_component", "proget-component"),
		ProGetUser:         resolveString(cmd, opts.ProGetUser, "proget_user", "proget-user"),
		ProGetAPIKey:       resolveString(cmd, opts.ProGetAPIKey, "proget_api_key", "proget-api-key"),
		ProGetTimeoutSec:   resolveInt(cmd, opts.ProGetTimeoutSec, "proget_timeout_sec", "proget-timeout"),
		ProGetRetries:      resolveInt(cmd, opts.ProGetRetries, "proget_retries", "proget-retries"),
		ProGetRetryDelayMs: resolveInt(cmd, opts.ProGetRetryDelay, "proget_retry_delay_ms", "proget-retry-delay-ms"),
	})
	if err != nil {
		return err
	}
	if result.DryRun {
		fmt.Printf("dry-run: keep=%d delete=%d\n", result.KeepCount, result.DeleteCount)
		return nil
	}
	fmt.Printf("pruned snapshots: %d\n", result.DeleteCount)
	return nil
}
