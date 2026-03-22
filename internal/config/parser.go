package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/0xmukesh/tatami/internal/constants"
	"github.com/spf13/viper"
)

type WmConfig struct {
	Modifier     uint16
	Bg           uint32
	Keybindings  []Keybinding
	TabBarConfig TabBarConfig
}

type TabBarConfig struct {
	Height       uint16 `mapstructure:"height"`
	ActiveBg     uint32 `mapstructure:"active_bg"`
	InactiveBg   uint32 `mapstructure:"inactive_bg"`
	ActiveText   uint32 `mapstructure:"active_text"`
	InactiveText uint32 `mapstructure:"inactive_text"`
}

type RawKeyBinding struct {
	Key     string `mapstructure:"key"`
	Action  string `mapstructure:"action"`
	Command string `mapstructure:"command"`
}

type Action string

type Keybinding struct {
	Raw     string
	Mod     uint16
	Keycode uint16
	Command string
	Action  Action
}

const (
	ActionExec         Action = "exec"
	ActionCloseFocused Action = "close_focused"
	ActionQuit         Action = "quit"
	ActionFocusLeft    Action = "focus_left"
	ActionFocusRight   Action = "focus_right"
	ActionMoveLeft     Action = "move_left"
	ActionMoveRight    Action = "move_right"
)

var validKeybindingActions = []Action{
	ActionExec, ActionCloseFocused, ActionQuit,
	ActionFocusLeft, ActionFocusRight, ActionMoveLeft,
	ActionMoveRight,
}

func parseRawKeybinding(raw RawKeyBinding, defaultModifier uint16) (Keybinding, error) {
	if raw.Key == "" {
		return Keybinding{}, fmt.Errorf("keybinding is missing `key`")
	}
	if raw.Action == "" {
		return Keybinding{}, fmt.Errorf("keybinding is missing `action`")
	}

	action := Action(raw.Action)
	if !slices.Contains(validKeybindingActions, action) {
		return Keybinding{}, fmt.Errorf("keybinding %q has unknown action %q", raw.Key, action)
	}

	if action == ActionExec && raw.Command == "" {
		return Keybinding{}, fmt.Errorf("keybinding %q with action `exec` is missing command", raw.Key)
	}

	parts := strings.Split(raw.Key, "+")
	key := parts[len(parts)-1]
	modifiers := parts[:len(parts)-1]
	mod := defaultModifier

	if len(parts) >= 2 {
		for _, m := range modifiers {
			if !slices.Contains(constants.ValidModifiers, m) {
				return Keybinding{}, fmt.Errorf("unknown modifier %q in keybinding %q", m, raw.Key)
			}

			val, ok := constants.KeycodeMap[m]
			if !ok {
				return Keybinding{}, fmt.Errorf("unknown keycode string literal %q in keybinding %q", m, raw.Key)
			}

			mod |= val
		}
	}

	keycode, ok := constants.KeycodeMap[key]
	if !ok {
		return Keybinding{}, fmt.Errorf("unknown keycode string literal %q in keybinding %q", key, raw.Key)
	}

	return Keybinding{
		Raw:     raw.Key,
		Mod:     mod,
		Keycode: keycode,
		Command: raw.Command,
		Action:  Action(raw.Action),
	}, nil
}

func parse() (WmConfig, error) {
	var errs []error

	defaultModifier := viper.GetString("keybindings.modifier")
	if defaultModifier == "" {
		errs = append(errs, fmt.Errorf("missing `keybindings.modifier` (valid: %s)", strings.Join(constants.ValidModifiers, ",")))
	}

	defaultModifierVal, ok := constants.KeycodeMap[defaultModifier]
	if !ok {
		return WmConfig{}, fmt.Errorf("invalid modifier %q (valid: %s)", defaultModifier, strings.Join(constants.ValidModifiers, ","))
	}

	bg := viper.GetUint32("general.bg")

	var rawKeys []RawKeyBinding
	if err := viper.UnmarshalKey("keybindings.keys", &rawKeys); err != nil {
		return WmConfig{}, fmt.Errorf("could not parse keybindings - %w", err)
	}

	var bindings []Keybinding
	for _, raw := range rawKeys {
		kb, err := parseRawKeybinding(raw, defaultModifierVal)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		bindings = append(bindings, kb)
	}

	var tabBarConfig TabBarConfig
	if err := viper.UnmarshalKey("general.tab_bar", &tabBarConfig); err != nil {
		return WmConfig{}, fmt.Errorf("could not parse tab bar config - %w", err)
	}

	if len(errs) > 0 {
		return WmConfig{}, errors.Join(errs...)
	}

	return WmConfig{
		Modifier:     defaultModifierVal,
		Bg:           bg,
		Keybindings:  bindings,
		TabBarConfig: tabBarConfig,
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
