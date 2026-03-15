package main

import (
	"log/slog"

	"github.com/0xmukesh/tatami/internal/config"
	"github.com/0xmukesh/tatami/internal/wm"
)

func main() {
	wmConfig := config.Parse()

	wm, err := wm.New(wmConfig)
	if err != nil {
		slog.Error("failed to setup wm", slog.String("error", err.Error()))
		return
	}
	defer wm.Close()

	wm.Run()
}
