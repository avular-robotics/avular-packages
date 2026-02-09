package cli

import "github.com/spf13/cobra"

type lockOptions = resolveOptions

func newLockCommand() *cobra.Command {
	opts := lockOptions{}
	cmd := &cobra.Command{
		Use:   "lock",
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

	return cmd
}
