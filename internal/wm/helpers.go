package wm

import (
	"log/slog"

	"github.com/0xmukesh/tatami/internal/constants"
	"github.com/jezek/xgb/xproto"
)

func (wm *Wm) renderTabBarWindow() {
	ws := wm.workspaces[wm.activeWorkspace]
	numClients := len(ws.clients)
	if numClients == 0 {
		return
	}

	totalWidth := uint32(wm.screen.WidthInPixels)
	tabWidth := totalWidth / uint32(numClients)

	for i, window := range ws.clients {
		startingX := uint32(i) * tabWidth
		width := tabWidth
		if i == numClients-1 {
			width = totalWidth - startingX
		}

		isActive := i == ws.active

		gc := wm.gcCache.InactiveTabBar
		if isActive {
			gc = wm.gcCache.ActiveTabBar
		}

		if err := xproto.PolyFillRectangleChecked(
			wm.conn, xproto.Drawable(ws.tabBar),
			gc,
			[]xproto.Rectangle{{
				X: int16(startingX), Y: 0, Width: uint16(width), Height: constants.TAB_BAR_HEIGHT,
			}},
		).Check(); err != nil {
			slog.Error("failed to properly render fill rectangle in tab bar", slog.String("error", err.Error()))
			return
		}

		title := wm.getWindowTitle(window)
		if err := xproto.ImageText8Checked(
			wm.conn, byte(len(title)), xproto.Drawable(ws.tabBar), gc, int16(startingX)+8, int16(constants.TAB_BAR_HEIGHT/2)+4,
			title,
		).Check(); err != nil {
			slog.Error("failed to properly render window title in tab err", slog.String("error", err.Error()))
			return
		}
	}
}

func (wm *Wm) closeWindow(window xproto.Window) {
	if wm.doesSupportDeleteProtocol(window) {
		event := xproto.ClientMessageEvent{
			Format: 32,
			Window: window,
			Type:   wm.atoms[constants.ATOM_WM_PROTOCOLS],
			Data:   xproto.ClientMessageDataUnionData32New([]uint32{uint32(wm.atoms[constants.ATOM_WM_DELETE_WINDOW]), uint32(xproto.TimeCurrentTime), 0, 0, 0}),
		}

		xproto.SendEventChecked(
			wm.conn, false, window, xproto.EventMaskNoEvent, event.String(),
		)
	} else {
		xproto.DestroyWindow(wm.conn, window)
	}
}
