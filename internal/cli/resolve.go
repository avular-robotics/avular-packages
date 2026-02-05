package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"avular-packages/internal/app"
)

type resolveOptions struct {
	Product       string
	Profiles      []string
	Workspace     []string
	RepoIndex     string
	OutputDir     string
	SnapshotID    string
	TargetUbuntu  string
	CompatGetDeps bool
	CompatRosdep  bool
}

func newResolveCommand() *cobra.Command {
	opts := resolveOptions{}
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve dependencies and produce lock outputs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runResolve(cmd.Context(), cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Product, "product", "", "Product spec path")
	cmd.Flags().StringSliceVar(&opts.Profiles, "profile", nil, "Profile spec paths")
	cmd.Flags().StringSliceVar(&opts.Workspace, "workspace", nil, "Workspace root(s)")
	cmd.Flags().StringVar(&opts.RepoIndex, "repo-index", "", "Repository index file")
	cmd.Flags().StringVar(&opts.OutputDir, "output", "out", "Output directory")
	cmd.Flags().StringVar(&opts.SnapshotID, "snapshot-id", "", "Snapshot ID (optional override)")
	cmd.Flags().StringVar(&opts.TargetUbuntu, "target-ubuntu", "", "Target Ubuntu release")
	cmd.Flags().BoolVar(&opts.CompatGetDeps, "compat-get-dependencies", false, "Emit get-dependencies compatible outputs")
	cmd.Flags().BoolVar(&opts.CompatRosdep, "compat-rosdep", false, "Emit rosdep-style mapping output")

	_ = viper.BindPFlag("product", cmd.Flags().Lookup("product"))
	_ = viper.BindPFlag("profiles", cmd.Flags().Lookup("profile"))
	_ = viper.BindPFlag("workspace", cmd.Flags().Lookup("workspace"))
	_ = viper.BindPFlag("repo_index", cmd.Flags().Lookup("repo-index"))
	_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("snapshot_id", cmd.Flags().Lookup("snapshot-id"))
	_ = viper.BindPFlag("target_ubuntu", cmd.Flags().Lookup("target-ubuntu"))
	_ = viper.BindPFlag("compat_get_dependencies", cmd.Flags().Lookup("compat-get-dependencies"))
	_ = viper.BindPFlag("compat_rosdep", cmd.Flags().Lookup("compat-rosdep"))

	return cmd
}

func runResolve(ctx context.Context, cmd *cobra.Command, opts resolveOptions) error {
	service := newAppService()
	result, err := service.Resolve(ctx, app.ResolveRequest{
		ProductPath:  resolveString(cmd, opts.Product, "product", "product"),
		Profiles:     resolveStrings(cmd, opts.Profiles, "profiles", "profile"),
		Workspace:    resolveStrings(cmd, opts.Workspace, "workspace", "workspace"),
		RepoIndex:    resolveString(cmd, opts.RepoIndex, "repo_index", "repo-index"),
		OutputDir:    resolveString(cmd, opts.OutputDir, "output", "output"),
		SnapshotID:   resolveString(cmd, opts.SnapshotID, "snapshot_id", "snapshot-id"),
		TargetUbuntu: resolveString(cmd, opts.TargetUbuntu, "target_ubuntu", "target-ubuntu"),
		CompatGet:    resolveBool(cmd, opts.CompatGetDeps, "compat_get_dependencies", "compat-get-dependencies"),
		CompatRosdep: resolveBool(cmd, opts.CompatRosdep, "compat_rosdep", "compat-rosdep"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("resolved: %s\n", result.ProductName)
	return nil
}
