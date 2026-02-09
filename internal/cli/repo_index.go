package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"avular-packages/internal/app"
)

type repoIndexOptions struct {
	Output           string
	AptSources       []string
	AptEndpoint      string
	AptDistribution  string
	AptComponent     string
	AptArch          string
	AptUser          string
	AptAPIKey        string
	AptWorkers       int
	PipIndex         string
	PipUser          string
	PipAPIKey        string
	PipPackages      []string
	PipMax           int
	PipWorkers       int
	HTTPTimeoutSec   int
	HTTPRetries      int
	HTTPRetryDelayMs int
	CacheDir         string
	CacheTTLMinutes  int
}

func newRepoIndexCommand() *cobra.Command {
	opts := repoIndexOptions{}
	cmd := &cobra.Command{
		Use:   "repo-index",
		Short: "Generate a repository index from APT and PyPI feeds",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRepoIndex(cmd.Context(), cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Output, "output", "repo-index.yaml", "Output path for repo index YAML")
	cmd.Flags().StringSliceVar(&opts.AptSources, "apt-source", nil, "APT source entry: endpoint|distribution|component|arch")
	cmd.Flags().StringVar(&opts.AptEndpoint, "apt-endpoint", "", "APT feed base URL (e.g., https://packages.avular.dev/debian/avular)")
	cmd.Flags().StringVar(&opts.AptDistribution, "apt-distribution", "", "APT distribution (e.g., dev, staging, snapshot)")
	cmd.Flags().StringVar(&opts.AptComponent, "apt-component", "main", "APT component")
	cmd.Flags().StringVar(&opts.AptArch, "apt-arch", "amd64", "APT architecture")
	cmd.Flags().StringVar(&opts.AptUser, "apt-user", "", "APT basic auth user (defaults to api)")
	cmd.Flags().StringVar(&opts.AptAPIKey, "apt-api-key", "", "APT basic auth password/API key")
	cmd.Flags().IntVar(&opts.AptWorkers, "apt-workers", 4, "Concurrent APT fetch workers (0 = default)")
	cmd.Flags().StringVar(&opts.PipIndex, "pip-index", "", "PyPI simple index base URL (e.g., https://packages.avular.dev/pypi/avular)")
	cmd.Flags().StringVar(&opts.PipUser, "pip-user", "", "PyPI basic auth user (defaults to api)")
	cmd.Flags().StringVar(&opts.PipAPIKey, "pip-api-key", "", "PyPI basic auth password/API key")
	cmd.Flags().StringSliceVar(&opts.PipPackages, "pip-package", nil, "Limit indexing to specified package(s)")
	cmd.Flags().IntVar(&opts.PipMax, "pip-max", 0, "Maximum number of PyPI packages to index (0 = all)")
	cmd.Flags().IntVar(&opts.PipWorkers, "pip-workers", 8, "Concurrent PyPI fetch workers (0 = default)")
	cmd.Flags().IntVar(&opts.HTTPTimeoutSec, "http-timeout", 60, "HTTP timeout in seconds (0 = default)")
	cmd.Flags().IntVar(&opts.HTTPRetries, "http-retries", 3, "HTTP retries (0 = default)")
	cmd.Flags().IntVar(&opts.HTTPRetryDelayMs, "http-retry-delay-ms", 200, "HTTP retry base delay in ms (0 = default)")
	cmd.Flags().StringVar(&opts.CacheDir, "cache-dir", "", "Optional cache directory for repo-index fetches")
	cmd.Flags().IntVar(&opts.CacheTTLMinutes, "cache-ttl-minutes", 60, "Cache TTL in minutes (0 = no caching)")

	_ = viper.BindPFlag("repo_index_output", cmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("apt_sources", cmd.Flags().Lookup("apt-source"))
	_ = viper.BindPFlag("apt_endpoint", cmd.Flags().Lookup("apt-endpoint"))
	_ = viper.BindPFlag("apt_distribution", cmd.Flags().Lookup("apt-distribution"))
	_ = viper.BindPFlag("apt_component", cmd.Flags().Lookup("apt-component"))
	_ = viper.BindPFlag("apt_arch", cmd.Flags().Lookup("apt-arch"))
	_ = viper.BindPFlag("apt_user", cmd.Flags().Lookup("apt-user"))
	_ = viper.BindPFlag("apt_api_key", cmd.Flags().Lookup("apt-api-key"))
	_ = viper.BindPFlag("apt_workers", cmd.Flags().Lookup("apt-workers"))
	_ = viper.BindPFlag("pip_index", cmd.Flags().Lookup("pip-index"))
	_ = viper.BindPFlag("pip_user", cmd.Flags().Lookup("pip-user"))
	_ = viper.BindPFlag("pip_api_key", cmd.Flags().Lookup("pip-api-key"))
	_ = viper.BindPFlag("pip_packages", cmd.Flags().Lookup("pip-package"))
	_ = viper.BindPFlag("pip_max", cmd.Flags().Lookup("pip-max"))
	_ = viper.BindPFlag("pip_workers", cmd.Flags().Lookup("pip-workers"))
	_ = viper.BindPFlag("http_timeout_sec", cmd.Flags().Lookup("http-timeout"))
	_ = viper.BindPFlag("http_retries", cmd.Flags().Lookup("http-retries"))
	_ = viper.BindPFlag("http_retry_delay_ms", cmd.Flags().Lookup("http-retry-delay-ms"))
	_ = viper.BindPFlag("repo_index_cache_dir", cmd.Flags().Lookup("cache-dir"))
	_ = viper.BindPFlag("repo_index_cache_ttl_minutes", cmd.Flags().Lookup("cache-ttl-minutes"))

	return cmd
}

func runRepoIndex(ctx context.Context, cmd *cobra.Command, opts repoIndexOptions) error {
	service := newAppService()
	result, err := service.RepoIndex(ctx, app.RepoIndexRequest{
		Output:           resolveString(cmd, opts.Output, "repo_index_output", "output"),
		AptSources:       resolveStrings(cmd, opts.AptSources, "apt_sources", "apt-source"),
		AptEndpoint:      resolveString(cmd, opts.AptEndpoint, "apt_endpoint", "apt-endpoint"),
		AptDistribution:  resolveString(cmd, opts.AptDistribution, "apt_distribution", "apt-distribution"),
		AptComponent:     resolveString(cmd, opts.AptComponent, "apt_component", "apt-component"),
		AptArch:          resolveString(cmd, opts.AptArch, "apt_arch", "apt-arch"),
		AptUser:          resolveString(cmd, opts.AptUser, "apt_user", "apt-user"),
		AptAPIKey:        resolveString(cmd, opts.AptAPIKey, "apt_api_key", "apt-api-key"),
		AptWorkers:       resolveInt(cmd, opts.AptWorkers, "apt_workers", "apt-workers"),
		PipIndex:         resolveString(cmd, opts.PipIndex, "pip_index", "pip-index"),
		PipUser:          resolveString(cmd, opts.PipUser, "pip_user", "pip-user"),
		PipAPIKey:        resolveString(cmd, opts.PipAPIKey, "pip_api_key", "pip-api-key"),
		PipPackages:      resolveStrings(cmd, opts.PipPackages, "pip_packages", "pip-package"),
		PipMax:           resolveInt(cmd, opts.PipMax, "pip_max", "pip-max"),
		PipWorkers:       resolveInt(cmd, opts.PipWorkers, "pip_workers", "pip-workers"),
		HTTPTimeoutSec:   resolveInt(cmd, opts.HTTPTimeoutSec, "http_timeout_sec", "http-timeout"),
		HTTPRetries:      resolveInt(cmd, opts.HTTPRetries, "http_retries", "http-retries"),
		HTTPRetryDelayMs: resolveInt(cmd, opts.HTTPRetryDelayMs, "http_retry_delay_ms", "http-retry-delay-ms"),
		CacheDir:         resolveString(cmd, opts.CacheDir, "repo_index_cache_dir", "cache-dir"),
		CacheTTLMinutes:  resolveInt(cmd, opts.CacheTTLMinutes, "repo_index_cache_ttl_minutes", "cache-ttl-minutes"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("wrote repo index: %s\n", result.OutputPath)
	return nil
}

func resolveInt(cmd *cobra.Command, value int, key string, flagName string) int {
	if cmd == nil {
		return value
	}
	if flagChanged(cmd, flagName) {
		return value
	}
	return viper.GetInt(key)
}
