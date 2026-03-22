package wm

import (
	"log/slog"
	"os/exec"

	"github.com/0xmukesh/tatami/internal/config"
)

type Keybinding struct {
	keycode uint16
	mod     uint16
	handler func()
}

func (wm *Wm) newKb(keycode, mod uint16, handler func()) Keybinding {
	return Keybinding{
		keycode: keycode,
		mod:     mod,
		handler: handler,
	}
}

func (wm *Wm) setupKeybindings() {
	for _, kb := range wm.config.Keybindings {
		var handler func()

		switch kb.Action {
		case config.ActionExec:
			handler = func() {
				if err := exec.Command(kb.Command).Start(); err != nil {
					slog.Error("failed to launch app launcher", slog.String("error", err.Error()))
					return
				}
			}
		case config.ActionCloseFocused:
			handler = func() {
				ws := wm.workspaces[wm.activeWorkspace]
				if len(ws.clients) > 0 {
					wm.closeWindow(ws.clients[ws.active])
				}
			}
		case config.ActionFocusLeft:
			handler = func() {
				wm.handleFocusWindow(true)
			}
		case config.ActionFocusRight:
			handler = func() {
				wm.handleFocusWindow(false)
			}
		case config.ActionMoveLeft:
			handler = func() {
				wm.handleMoveWindow(true)
			}
		case config.ActionMoveRight:
			handler = func() {
				wm.handleMoveWindow(false)
			}
		case config.ActionQuit:
			handler = func() {
				wm.Close()
			}
		}

		wm.keybindings = append(wm.keybindings, wm.newKb(kb.Keycode, kb.Mod, handler))
	}
}
