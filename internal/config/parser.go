package config

import (
	"flag"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/0xmukesh/tatami/internal/constants"
)

type WmConfig struct {
	Modifier uint16
	Launcher string
	Terminal string
}

var (
	modifier string
	launcher string
	terminal string
)

var modifiersMap = map[string]uint16{
	"mod1":  constants.KB_MOD1,
	"mod2":  constants.KB_MOD2,
	"mod3":  constants.KB_MOD3,
	"mod4":  constants.KB_MOD4,
	"mod5":  constants.KB_MOD5,
	"ctrl":  constants.KB_MODCTRL,
	"shift": constants.KB_MODSHIFT,
}

func Parse() WmConfig {
	flag.StringVar(&modifier, "mod", "mod1", "modifier key which would be used in key bindings")
	flag.StringVar(&launcher, "launcher", "dmenu_run", "program which would act like an app launcher")
	flag.StringVar(&terminal, "terminal", "kitty", "default terminal app")

	flag.Parse()

	isValidModifier := slices.Contains(slices.Collect(maps.Keys(modifiersMap)), modifier)
	if !isValidModifier {
		slog.Error("invalid modifiers", slog.String("valid modifiers", strings.Join(slices.Collect(maps.Keys(modifiersMap)), ", ")))
	}

	return WmConfig{
		Modifier: modifiersMap[modifier],
		Launcher: launcher,
		Terminal: terminal,
	}
}
