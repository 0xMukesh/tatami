package main

import (
	"log/slog"

	"github.com/0xmukesh/tatami/internal/wm"
	"github.com/jezek/xgb/xproto"
)

func main() {
	wm, err := wm.New(xproto.ModMask1)
	if err != nil {
		slog.Error("failed to setup wm", slog.String("error", err.Error()))
		return
	}
	defer wm.Close()

	wm.Run()
}
