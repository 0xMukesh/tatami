package wm

import (
	"log/slog"
	"os/exec"

	"github.com/0xmukesh/tatami/internal/constants"
)

type Keybinding struct {
	keycode uint16
	mod     uint16
	handler func()
}

func (wm *Wm) newKb(keycode uint16, handler func()) Keybinding {
	return Keybinding{
		keycode: keycode,
		mod:     wm.config.Modifier,
		handler: handler,
	}
}

func (wm *Wm) newKbWithExtraMod(keycode, extraMod uint16, handler func()) Keybinding {
	return Keybinding{
		keycode: keycode,
		mod:     wm.config.Modifier | extraMod,
		handler: handler,
	}
}

func (wm *Wm) setupKeybindings() {
	wm.keybindings = []Keybinding{
		wm.newKb(constants.KB_D, func() {
			if err := exec.Command(wm.config.Launcher).Start(); err != nil {
				slog.Error("failed to launch app launcher", slog.String("error", err.Error()))
				return
			}
		}),
		wm.newKb(constants.KB_ENTER, func() {
			if err := exec.Command(wm.config.Terminal).Start(); err != nil {
				slog.Error("failed to launch app launcher", slog.String("error", err.Error()))
				return
			}
		}),
		wm.newKb(constants.KB_Q, func() {
			ws := wm.workspaces[wm.activeWorkspace]
			if len(ws.clients) > 0 {
				wm.closeWindow(ws.clients[ws.active])
			}
		}),
		wm.newKb(constants.KB_LEFT_ARROW, func() {
			wm.handleWindowNavigation(true)
		}),
		wm.newKb(constants.KB_RIGHT_ARROW, func() {
			wm.handleWindowNavigation(false)
		}),
		wm.newKbWithExtraMod(constants.KB_LEFT_ARROW, constants.KB_MODSHIFT, func() {
			ws := wm.workspaces[wm.activeWorkspace]
			if ws.active <= 0 {
				return
			}

			wm.rearrangeWindows(ws.active, ws.active-1)
			ws.active -= 1
			wm.renderTabBarWindow()
		}),
		wm.newKbWithExtraMod(constants.KB_RIGHT_ARROW, constants.KB_MODSHIFT, func() {
			ws := wm.workspaces[wm.activeWorkspace]
			if ws.active >= len(ws.clients)-1 {
				return
			}

			wm.rearrangeWindows(ws.active, ws.active+1)
			ws.active += 1
			wm.renderTabBarWindow()
		}),
	}
}
