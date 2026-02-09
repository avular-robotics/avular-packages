package cli

import (
	"errors"
	"os"
	"strings"

	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// version is set at build time via ldflags.
var version = "dev"

const envPrefix = "AVULAR_PACKAGES"

type RootConfig struct {
	ConfigFile string
	LogLevel   string
}

func Execute() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		os.Exit(exitCodeForError(err))
	}
}

func newRootCommand() *cobra.Command {
	cfg := RootConfig{}
	cmd := &cobra.Command{
		Use:     "avular-packages",
		Short:   "Unified dependency resolver and packager",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := initConfig(cfg.ConfigFile); err != nil {
				return err
			}
			setupLogging(viper.GetString("log_level"))
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&cfg.ConfigFile, "config", "", "Config file path")
	cmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "Log level")
	_ = viper.BindPFlag("log_level", cmd.PersistentFlags().Lookup("log-level"))

	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newResolveCommand())
	cmd.AddCommand(newLockCommand())
	cmd.AddCommand(newBuildCommand())
	cmd.AddCommand(newPublishCommand())
	cmd.AddCommand(newInspectCommand())
	cmd.AddCommand(newRepoIndexCommand())
	cmd.AddCommand(newPruneCommand())
	return cmd
}

func initConfig(configFile string) error {
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return errbuilder.New().
				WithCode(errbuilder.CodeInvalidArgument).
				WithMsg("failed to read config file").
				WithCause(err)
		}
		return nil
	}

	viper.SetConfigName("avular-packages")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/avular-packages")
	if err := viper.ReadInConfig(); err != nil {
		return nil
	}
	return nil
}

func setupLogging(level string) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func exitCodeForError(err error) int {
	code := errbuilder.CodeOf(err)
	message := errorMessage(err)
	switch code {
	case errbuilder.CodeInvalidArgument, errbuilder.CodeAlreadyExists:
		return 2
	case errbuilder.CodeFailedPrecondition:
		if strings.HasPrefix(message, "conflict without resolution directive") {
			return 3
		}
		if strings.HasPrefix(message, "no compatible version") {
			return 4
		}
		return 4
	case errbuilder.CodePermissionDenied:
		return 3
	case errbuilder.CodeNotFound:
		if strings.HasPrefix(message, "no available versions") {
			return 4
		}
		return 5
	case errbuilder.CodeInternal:
		return 5
	default:
		return 1
	}
}

func errorMessage(err error) string {
	var builder *errbuilder.ErrBuilder
	if errors.As(err, &builder) && strings.TrimSpace(builder.Msg) != "" {
		return builder.Msg
	}
	return err.Error()
}
