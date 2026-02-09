package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"avular-packages/internal/app"
)

type resolveOptions struct {
	Product              string
	Profiles             []string
	Workspace            []string
	RepoIndex            string
	OutputDir            string
	SnapshotID           string
	TargetUbuntu         string
	SchemaFiles          []string
	CompatGetDeps        bool
	CompatRosdep         bool
	AptPreferences       bool
	AptInstallList       bool
	SnapshotSources      bool
	SnapshotAptBaseURL   string
	SnapshotAptComponent string
	SnapshotAptArchs     []string
	AptSatSolver         bool
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
	cmd.Flags().BoolVar(&opts.AptPreferences, "apt-preferences", false, "Emit apt preferences pin file from apt.lock")
	cmd.Flags().BoolVar(&opts.AptInstallList, "apt-install-list", false, "Emit apt-get install command from apt.lock")
	cmd.Flags().BoolVar(&opts.SnapshotSources, "snapshot-apt-sources", false, "Emit snapshot-locked sources.list snippet")
	cmd.Flags().StringVar(&opts.SnapshotAptBaseURL, "snapshot-apt-base-url", "", "Base URL for snapshot apt repo (e.g., https://packages.example.com/debian/feed)")
	cmd.Flags().StringVar(&opts.SnapshotAptComponent, "snapshot-apt-component", "main", "Component for snapshot apt source")
	cmd.Flags().StringSliceVar(&opts.SnapshotAptArchs, "snapshot-apt-arch", nil, "Optional arch list for snapshot apt source")
	cmd.Flags().BoolVar(&opts.AptSatSolver, "apt-sat-solver", false, "Resolve apt versions with SAT-based dependency closure")
	cmd.Flags().StringSliceVar(&opts.SchemaFiles, "schema", nil, "Schema mapping file(s) for ROS tag resolution (layered, last wins)")

	_ = viper.BindPFlag("product", cmd.Flags().Lookup("product"))
	_ = viper.BindPFlag("profiles", cmd.Flags().Lookup("profile"))
	_ = viper.BindPFlag("workspace", cmd.Flags().Lookup("workspace"))
	_ = viper.BindPFlag("repo_index", cmd.Flags().Lookup("repo-index"))
	_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("snapshot_id", cmd.Flags().Lookup("snapshot-id"))
	_ = viper.BindPFlag("target_ubuntu", cmd.Flags().Lookup("target-ubuntu"))
	_ = viper.BindPFlag("compat_get_dependencies", cmd.Flags().Lookup("compat-get-dependencies"))
	_ = viper.BindPFlag("compat_rosdep", cmd.Flags().Lookup("compat-rosdep"))
	_ = viper.BindPFlag("apt_preferences", cmd.Flags().Lookup("apt-preferences"))
	_ = viper.BindPFlag("apt_install_list", cmd.Flags().Lookup("apt-install-list"))
	_ = viper.BindPFlag("snapshot_apt_sources", cmd.Flags().Lookup("snapshot-apt-sources"))
	_ = viper.BindPFlag("snapshot_apt_base_url", cmd.Flags().Lookup("snapshot-apt-base-url"))
	_ = viper.BindPFlag("snapshot_apt_component", cmd.Flags().Lookup("snapshot-apt-component"))
	_ = viper.BindPFlag("snapshot_apt_arch", cmd.Flags().Lookup("snapshot-apt-arch"))
	_ = viper.BindPFlag("apt_sat_solver", cmd.Flags().Lookup("apt-sat-solver"))
	_ = viper.BindPFlag("schema_files", cmd.Flags().Lookup("schema"))

	return cmd
}

func runResolve(ctx context.Context, cmd *cobra.Command, opts resolveOptions) error {
	service := newAppService()
	result, err := service.Resolve(ctx, app.ResolveRequest{
		ProductPath:          resolveString(cmd, opts.Product, "product", "product"),
		Profiles:             resolveStrings(cmd, opts.Profiles, "profiles", "profile"),
		Workspace:            resolveStrings(cmd, opts.Workspace, "workspace", "workspace"),
		RepoIndex:            resolveString(cmd, opts.RepoIndex, "repo_index", "repo-index"),
		OutputDir:            resolveString(cmd, opts.OutputDir, "output", "output"),
		SnapshotID:           resolveString(cmd, opts.SnapshotID, "snapshot_id", "snapshot-id"),
		TargetUbuntu:         resolveString(cmd, opts.TargetUbuntu, "target_ubuntu", "target-ubuntu"),
		SchemaFiles:          resolveStrings(cmd, opts.SchemaFiles, "schema_files", "schema"),
		CompatGet:            resolveBool(cmd, opts.CompatGetDeps, "compat_get_dependencies", "compat-get-dependencies"),
		CompatRosdep:         resolveBool(cmd, opts.CompatRosdep, "compat_rosdep", "compat-rosdep"),
		EmitAptPreferences:   resolveBool(cmd, opts.AptPreferences, "apt_preferences", "apt-preferences"),
		EmitAptInstallList:   resolveBool(cmd, opts.AptInstallList, "apt_install_list", "apt-install-list"),
		EmitSnapshotSources:  resolveBool(cmd, opts.SnapshotSources, "snapshot_apt_sources", "snapshot-apt-sources"),
		SnapshotAptBaseURL:   resolveString(cmd, opts.SnapshotAptBaseURL, "snapshot_apt_base_url", "snapshot-apt-base-url"),
		SnapshotAptComponent: resolveString(cmd, opts.SnapshotAptComponent, "snapshot_apt_component", "snapshot-apt-component"),
		SnapshotAptArchs:     resolveStrings(cmd, opts.SnapshotAptArchs, "snapshot_apt_arch", "snapshot-apt-arch"),
		AptSatSolver:         resolveBool(cmd, opts.AptSatSolver, "apt_sat_solver", "apt-sat-solver"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("resolved: %s\n", result.ProductName)
	return nil
}
