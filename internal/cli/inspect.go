package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"avular-packages/internal/app"
)

type inspectOptions struct {
	OutputDir string
}

func newInspectCommand() *cobra.Command {
	opts := inspectOptions{}
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect resolved outputs and bundle membership",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInspect(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.OutputDir, "output", "out", "Output directory")
	_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))
	return cmd
}

func runInspect(cmd *cobra.Command, opts inspectOptions) error {
	service := newAppService()
	result, err := service.Inspect(app.InspectRequest{
		OutputDir: resolveString(cmd, opts.OutputDir, "output", "output"),
	})
	if err != nil {
		return err
	}

	fmt.Printf("apt.lock entries: %d\n", result.AptLockCount)
	fmt.Println("bundle.manifest groups:")
	for _, summary := range result.Groups {
		fmt.Printf("- %s (%s): %d packages\n", summary.Name, summary.Mode, summary.Count)
		if len(summary.Packages) > 0 {
			fmt.Printf("  %s\n", strings.Join(summary.Packages, ", "))
		}
	}
	fmt.Printf("resolution.report records: %d\n", len(result.ResolutionRecords))
	for _, record := range result.ResolutionRecords {
		fmt.Printf("- %s %s %s (owner=%s)\n", record.Dependency, record.Action, record.Value, record.Owner)
	}
	return nil
}
