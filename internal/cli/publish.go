package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"avular-packages/internal/app"
)

type publishOptions struct {
	OutputDir          string
	RepoDir            string
	SBOM               bool
	RepoBackend        string
	DebsDir            string
	AptlyRepo          string
	AptlyComponent     string
	AptlyPrefix        string
	AptlyEndpoint      string
	GpgKey             string
	ProGetEndpoint     string
	ProGetFeed         string
	ProGetComponent    string
	ProGetUser         string
	ProGetAPIKey       string
	ProGetWorkers      int
	ProGetTimeoutSec   int
	ProGetRetries      int
	ProGetRetryDelayMs int
}

func newPublishCommand() *cobra.Command {
	opts := publishOptions{}
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish deb artifacts and create a snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPublish(cmd.Context(), cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.OutputDir, "output", "out", "Output directory containing snapshot.intent")
	cmd.Flags().StringVar(&opts.RepoDir, "repo-dir", "", "Repository directory for snapshot metadata")
	cmd.Flags().BoolVar(&opts.SBOM, "sbom", true, "Generate SBOM alongside snapshot metadata")
	cmd.Flags().StringVar(&opts.RepoBackend, "repo-backend", "file", "Repository backend (file, aptly, or proget)")
	cmd.Flags().StringVar(&opts.DebsDir, "debs-dir", "", "Directory with deb artifacts (aptly/proget backends)")
	cmd.Flags().StringVar(&opts.AptlyRepo, "aptly-repo", "", "Aptly repo name (defaults to snapshot intent repository)")
	cmd.Flags().StringVar(&opts.AptlyComponent, "aptly-component", "main", "Aptly component name")
	cmd.Flags().StringVar(&opts.AptlyPrefix, "aptly-prefix", ".", "Aptly publish prefix")
	cmd.Flags().StringVar(&opts.AptlyEndpoint, "aptly-endpoint", "", "Aptly publish endpoint (e.g., s3:repo)")
	cmd.Flags().StringVar(&opts.GpgKey, "gpg-key", "", "GPG key ID for signing")
	cmd.Flags().StringVar(&opts.ProGetEndpoint, "proget-endpoint", "", "ProGet base URL (e.g., https://packages.example.com)")
	cmd.Flags().StringVar(&opts.ProGetFeed, "proget-feed", "", "ProGet Debian feed name (defaults to snapshot intent repository)")
	cmd.Flags().StringVar(&opts.ProGetComponent, "proget-component", "main", "ProGet Debian component name")
	cmd.Flags().StringVar(&opts.ProGetUser, "proget-user", "", "ProGet username for basic auth (defaults to api)")
	cmd.Flags().StringVar(&opts.ProGetAPIKey, "proget-api-key", "", "ProGet API key or password for basic auth")
	cmd.Flags().IntVar(&opts.ProGetWorkers, "proget-workers", 4, "Concurrent ProGet upload workers (0 = default)")
	cmd.Flags().IntVar(&opts.ProGetTimeoutSec, "proget-timeout", 60, "ProGet HTTP timeout in seconds (0 = default)")
	cmd.Flags().IntVar(&opts.ProGetRetries, "proget-retries", 3, "ProGet upload retries (0 = default)")
	cmd.Flags().IntVar(&opts.ProGetRetryDelayMs, "proget-retry-delay-ms", 200, "ProGet retry base delay in ms (0 = default)")
	_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("repo_dir", cmd.Flags().Lookup("repo-dir"))
	_ = viper.BindPFlag("sbom", cmd.Flags().Lookup("sbom"))
	_ = viper.BindPFlag("repo_backend", cmd.Flags().Lookup("repo-backend"))
	_ = viper.BindPFlag("debs_dir", cmd.Flags().Lookup("debs-dir"))
	_ = viper.BindPFlag("aptly_repo", cmd.Flags().Lookup("aptly-repo"))
	_ = viper.BindPFlag("aptly_component", cmd.Flags().Lookup("aptly-component"))
	_ = viper.BindPFlag("aptly_prefix", cmd.Flags().Lookup("aptly-prefix"))
	_ = viper.BindPFlag("aptly_endpoint", cmd.Flags().Lookup("aptly-endpoint"))
	_ = viper.BindPFlag("gpg_key", cmd.Flags().Lookup("gpg-key"))
	_ = viper.BindPFlag("proget_endpoint", cmd.Flags().Lookup("proget-endpoint"))
	_ = viper.BindPFlag("proget_feed", cmd.Flags().Lookup("proget-feed"))
	_ = viper.BindPFlag("proget_component", cmd.Flags().Lookup("proget-component"))
	_ = viper.BindPFlag("proget_user", cmd.Flags().Lookup("proget-user"))
	_ = viper.BindPFlag("proget_api_key", cmd.Flags().Lookup("proget-api-key"))
	_ = viper.BindPFlag("proget_workers", cmd.Flags().Lookup("proget-workers"))
	_ = viper.BindPFlag("proget_timeout_sec", cmd.Flags().Lookup("proget-timeout"))
	_ = viper.BindPFlag("proget_retries", cmd.Flags().Lookup("proget-retries"))
	_ = viper.BindPFlag("proget_retry_delay_ms", cmd.Flags().Lookup("proget-retry-delay-ms"))
	return cmd
}

func runPublish(_ context.Context, cmd *cobra.Command, opts publishOptions) error {
	service := newAppService()
	result, err := service.Publish(cmd.Context(), app.PublishRequest{
		OutputDir:          resolveString(cmd, opts.OutputDir, "output", "output"),
		RepoDir:            resolveString(cmd, opts.RepoDir, "repo_dir", "repo-dir"),
		SBOM:               resolveBool(cmd, opts.SBOM, "sbom", "sbom"),
		RepoBackend:        resolveString(cmd, opts.RepoBackend, "repo_backend", "repo-backend"),
		DebsDir:            resolveString(cmd, opts.DebsDir, "debs_dir", "debs-dir"),
		AptlyRepo:          resolveString(cmd, opts.AptlyRepo, "aptly_repo", "aptly-repo"),
		AptlyComponent:     resolveString(cmd, opts.AptlyComponent, "aptly_component", "aptly-component"),
		AptlyPrefix:        resolveString(cmd, opts.AptlyPrefix, "aptly_prefix", "aptly-prefix"),
		AptlyEndpoint:      resolveString(cmd, opts.AptlyEndpoint, "aptly_endpoint", "aptly-endpoint"),
		GpgKey:             resolveString(cmd, opts.GpgKey, "gpg_key", "gpg-key"),
		ProGetEndpoint:     resolveString(cmd, opts.ProGetEndpoint, "proget_endpoint", "proget-endpoint"),
		ProGetFeed:         resolveString(cmd, opts.ProGetFeed, "proget_feed", "proget-feed"),
		ProGetComponent:    resolveString(cmd, opts.ProGetComponent, "proget_component", "proget-component"),
		ProGetUser:         resolveString(cmd, opts.ProGetUser, "proget_user", "proget-user"),
		ProGetAPIKey:       resolveString(cmd, opts.ProGetAPIKey, "proget_api_key", "proget-api-key"),
		ProGetWorkers:      resolveInt(cmd, opts.ProGetWorkers, "proget_workers", "proget-workers"),
		ProGetTimeoutSec:   resolveInt(cmd, opts.ProGetTimeoutSec, "proget_timeout_sec", "proget-timeout"),
		ProGetRetries:      resolveInt(cmd, opts.ProGetRetries, "proget_retries", "proget-retries"),
		ProGetRetryDelayMs: resolveInt(cmd, opts.ProGetRetryDelayMs, "proget_retry_delay_ms", "proget-retry-delay-ms"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("published snapshot: %s\n", result.SnapshotID)
	return nil
}
