package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"avular-packages/internal/app"
)

type buildOptions struct {
	Product        string
	Profiles       []string
	Workspace      []string
	RepoIndex      string
	OutputDir      string
	DebsDir        string
	TargetUbuntu   string
	PipIndexURL    string
	InternalDebDir string
	InternalSrc    []string
}

func newBuildCommand() *cobra.Command {
	opts := buildOptions{}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build deb artifacts from resolved dependencies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBuild(cmd.Context(), cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Product, "product", "", "Product spec path")
	cmd.Flags().StringSliceVar(&opts.Profiles, "profile", nil, "Profile spec paths")
	cmd.Flags().StringSliceVar(&opts.Workspace, "workspace", nil, "Workspace root(s)")
	cmd.Flags().StringVar(&opts.RepoIndex, "repo-index", "", "Repository index file")
	cmd.Flags().StringVar(&opts.OutputDir, "output", "out", "Output directory")
	cmd.Flags().StringVar(&opts.DebsDir, "debs-dir", "", "Directory for built debs")
	cmd.Flags().StringVar(&opts.TargetUbuntu, "target-ubuntu", "", "Target Ubuntu release")
	cmd.Flags().StringVar(&opts.PipIndexURL, "pip-index-url", "", "Optional PIP index URL override")
	cmd.Flags().StringVar(&opts.InternalDebDir, "internal-deb-dir", "", "Directory containing prebuilt internal debs")
	cmd.Flags().StringSliceVar(&opts.InternalSrc, "internal-src", nil, "Internal package source directory (debian)")

	_ = viper.BindPFlag("product", cmd.Flags().Lookup("product"))
	_ = viper.BindPFlag("profiles", cmd.Flags().Lookup("profile"))
	_ = viper.BindPFlag("workspace", cmd.Flags().Lookup("workspace"))
	_ = viper.BindPFlag("repo_index", cmd.Flags().Lookup("repo-index"))
	_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("debs_dir", cmd.Flags().Lookup("debs-dir"))
	_ = viper.BindPFlag("target_ubuntu", cmd.Flags().Lookup("target-ubuntu"))
	_ = viper.BindPFlag("pip_index_url", cmd.Flags().Lookup("pip-index-url"))
	_ = viper.BindPFlag("internal_deb_dir", cmd.Flags().Lookup("internal-deb-dir"))
	_ = viper.BindPFlag("internal_src", cmd.Flags().Lookup("internal-src"))

	return cmd
}

func runBuild(ctx context.Context, cmd *cobra.Command, opts buildOptions) error {
	service := newAppService()
	result, err := service.Build(ctx, app.BuildRequest{
		ProductPath:    resolveString(cmd, opts.Product, "product", "product"),
		Profiles:       resolveStrings(cmd, opts.Profiles, "profiles", "profile"),
		Workspace:      resolveStrings(cmd, opts.Workspace, "workspace", "workspace"),
		RepoIndex:      resolveString(cmd, opts.RepoIndex, "repo_index", "repo-index"),
		OutputDir:      resolveString(cmd, opts.OutputDir, "output", "output"),
		DebsDir:        resolveString(cmd, opts.DebsDir, "debs_dir", "debs-dir"),
		TargetUbuntu:   resolveString(cmd, opts.TargetUbuntu, "target_ubuntu", "target-ubuntu"),
		PipIndexURL:    resolveString(cmd, opts.PipIndexURL, "pip_index_url", "pip-index-url"),
		InternalDebDir: resolveString(cmd, opts.InternalDebDir, "internal_deb_dir", "internal-deb-dir"),
		InternalSrc:    resolveStrings(cmd, opts.InternalSrc, "internal_src", "internal-src"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("built debs: %s\n", result.DebsDir)
	return nil
}
