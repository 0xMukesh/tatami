package cmd

import (
	"log/slog"

	"github.com/0xmukesh/tatami/internal/config"
	"github.com/0xmukesh/tatami/internal/wm"
	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "launch tatami wm",
	Run:   runLaunchCmd,
}

func init() {
	rootCmd.AddCommand(launchCmd)
}

func runLaunchCmd(cmd *cobra.Command, args []string) {
	wmConfig := config.Parse()

	wm, err := wm.New(wmConfig)
	if err != nil {
		slog.Error("failed to setup wm", slog.String("error", err.Error()))
		return
	}
	defer wm.Close()

	wm.Run()
}
