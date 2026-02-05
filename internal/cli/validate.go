package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"avular-packages/internal/app"
)

type validateOptions struct {
	Product  string
	Profiles []string
}

func newValidateCommand() *cobra.Command {
	opts := validateOptions{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate product and profile specs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidate(cmd.Context(), cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Product, "product", "", "Product spec path")
	cmd.Flags().StringSliceVar(&opts.Profiles, "profile", nil, "Profile spec paths")
	_ = viper.BindPFlag("product", cmd.Flags().Lookup("product"))
	_ = viper.BindPFlag("profiles", cmd.Flags().Lookup("profile"))
	return cmd
}

func runValidate(ctx context.Context, cmd *cobra.Command, opts validateOptions) error {
	service := newAppService()
	result, err := service.Validate(ctx, app.ValidateRequest{
		ProductPath: resolveString(cmd, opts.Product, "product", "product"),
		Profiles:    resolveStrings(cmd, opts.Profiles, "profiles", "profile"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("validated: %s\n", result.ProductName)
	return nil
}

func resolveString(cmd *cobra.Command, value string, key string, flagName string) string {
	if cmd == nil {
		if value != "" {
			return value
		}
		return viper.GetString(key)
	}
	if flagChanged(cmd, flagName) {
		return value
	}
	return viper.GetString(key)
}

func resolveStrings(cmd *cobra.Command, values []string, key string, flagName string) []string {
	if cmd == nil {
		if len(values) > 0 {
			return values
		}
		return viper.GetStringSlice(key)
	}
	if flagChanged(cmd, flagName) {
		return values
	}
	return viper.GetStringSlice(key)
}

func resolveBool(cmd *cobra.Command, value bool, key string, flagName string) bool {
	if cmd == nil {
		return value
	}
	if flagChanged(cmd, flagName) {
		return value
	}
	return viper.GetBool(key)
}

func flagChanged(cmd *cobra.Command, name string) bool {
	if cmd == nil || strings.TrimSpace(name) == "" {
		return false
	}
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag.Changed
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		return flag.Changed
	}
	return false
}
