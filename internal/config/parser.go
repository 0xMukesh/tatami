package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/0xmukesh/tatami/internal/constants"
	"github.com/spf13/viper"
)

type WmConfig struct {
	Modifier uint16
	Launcher string
	Terminal string
}

var modifiersMap = map[string]uint16{
	"mod1":  constants.KB_MOD1,
	"mod2":  constants.KB_MOD2,
	"mod3":  constants.KB_MOD3,
	"mod4":  constants.KB_MOD4,
	"mod5":  constants.KB_MOD5,
	"ctrl":  constants.KB_MODCTRL,
	"shift": constants.KB_MODSHIFT,
}

func validModifiers() string {
	keys := make([]string, 0, len(modifiersMap))
	for k := range modifiersMap {
		keys = append(keys, k)
	}

	return strings.Join(keys, ", ")
}

func parse() (WmConfig, error) {
	var errs []error

	modifier := viper.GetString("keybindings.modifier")
	launcher := viper.GetString("general.launcher")
	terminal := viper.GetString("general.terminal")

	if modifier == "" {
		errs = append(errs, fmt.Errorf("missing `keybindings.modifier` (valid: %s)", validModifiers()))
	}
	if launcher == "" {
		errs = append(errs, errors.New("missing `general.launcher`"))
	}
	if terminal == "" {
		errs = append(errs, errors.New("missing `general.terminal`"))
	}

	if len(errs) > 0 {
		return WmConfig{}, errors.Join(errs...)
	}

	modifierVal, ok := modifiersMap[modifier]
	if !ok {
		return WmConfig{}, fmt.Errorf("invalid modifier %q (valid: %s)", modifier, validModifiers())
	}

	return WmConfig{
		Modifier: modifierVal,
		Launcher: launcher,
		Terminal: terminal,
	}, nil
}

func Parse() WmConfig {
	cfg, err := parse()
	if err != nil {
		slog.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	return cfg
}
