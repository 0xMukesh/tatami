package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "tatami",
		Short: "tatami is a x11 tabbed window manager",
	}
)

func init() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		slog.Error("failed to get user config dir", slog.String("error", err.Error()))
		os.Exit(1)
	}

	cobra.OnInitialize(initConfig(userConfigDir))

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", fmt.Sprintf("config file path (default - %s/tatami/config.yaml)", userConfigDir))
}

func initConfig(userConfigDir string) func() {
	return func() {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			viper.AddConfigPath(fmt.Sprintf("%s/tatami", userConfigDir))
			viper.AddConfigPath(".")
			viper.SetConfigType("yaml")
			viper.SetConfigName("config")
		}

		viper.SetEnvPrefix("TATAMI")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			slog.Error("failed to read config", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("failed to execute command", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
