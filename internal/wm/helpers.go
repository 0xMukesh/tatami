package wm

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/0xmukesh/tatami/internal/constants"
	"github.com/jezek/xgb/xproto"
)

func (wm *Wm) createWorkspace(workspace int) (ws *Workspace, err error) {
	if ws, ok := wm.workspaces[workspace]; ok {
		return ws, nil
	}

	frame, err := xproto.NewWindowId(wm.conn)
	if err != nil {
		return ws, fmt.Errorf("failed to assign frame window id - %w", err)
	}

	if err := xproto.CreateWindowChecked(
		wm.conn,
		wm.screen.RootDepth,
		frame,
		wm.root,
		0, 0,
		wm.screen.WidthInPixels, wm.screen.HeightInPixels,
		0,
		xproto.WindowClassInputOutput,
		wm.screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			wm.config.Bg,
			xproto.EventMaskSubstructureNotify | xproto.EventMaskSubstructureRedirect,
		},
	).Check(); err != nil {
		return ws, fmt.Errorf("failed to create frame window - %w", err)
	}

	tabBar, err := xproto.NewWindowId(wm.conn)
	if err != nil {
		return ws, fmt.Errorf("failed to assign tab bar window id - %w", err)
	}

	if err := xproto.CreateWindowChecked(
		wm.conn,
		wm.screen.RootDepth,
		tabBar,
		frame,
		0, 0,
		wm.screen.WidthInPixels, wm.config.TabBarConfig.Height,
		0,
		xproto.WindowClassInputOutput,
		wm.screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			wm.config.TabBarConfig.InactiveBg,
			xproto.EventMaskExposure,
		},
	).Check(); err != nil {
		return ws, fmt.Errorf("failed to create tab bar window - %w", err)
	}

	bottomBar, err := xproto.NewWindowId(wm.conn)
	if err != nil {
		return ws, fmt.Errorf("failed to assign bottom bar window id - %w", err)
	}

	if err := xproto.CreateWindowChecked(
		wm.conn,
		wm.screen.RootDepth,
		bottomBar,
		frame,
		0, int16(wm.screen.HeightInPixels)-int16(wm.config.BottomBarConfig.Height),
		wm.screen.WidthInPixels, wm.config.BottomBarConfig.Height,
		0,
		xproto.WindowClassInputOutput,
		wm.screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			wm.config.BottomBarConfig.InactiveBg,
			xproto.EventMaskExposure,
		},
	).Check(); err != nil {
		return ws, fmt.Errorf("failed to create bottom bar window - %w", err)
	}

	if err := xproto.MapWindowChecked(wm.conn, frame).Check(); err != nil {
		return ws, fmt.Errorf("failed to map frame window - %w", err)
	}

	if err := xproto.MapWindowChecked(wm.conn, bottomBar).Check(); err != nil {
		return ws, fmt.Errorf("failed to map bottom bar window - %w", err)
	}

	wm.registerShortcuts(frame)

	return &Workspace{
		frame:     frame,
		tabBar:    tabBar,
		bottomBar: bottomBar,
	}, nil
}

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
		fillGc := wm.gcCache.Tab.Inactive.Fill
		textGc := wm.gcCache.Tab.Inactive.Text

		if isActive {
			fillGc = wm.gcCache.Tab.Active.Fill
			textGc = wm.gcCache.Tab.Active.Text
		}

		if err := xproto.PolyFillRectangleChecked(
			wm.conn, xproto.Drawable(ws.tabBar),
			fillGc,
			[]xproto.Rectangle{{
				X: int16(startingX), Y: 0,
				Width: uint16(width), Height: wm.config.TabBarConfig.Height,
			}},
		).Check(); err != nil {
			slog.Error("failed to properly render fill rectangle in tab bar", slog.String("error", err.Error()))
			return
		}

		title := wm.getWindowTitle(window)
		if err := xproto.ImageText8Checked(
			wm.conn, byte(len(title)), xproto.Drawable(ws.tabBar), textGc,
			int16(startingX)+8, int16(wm.config.TabBarConfig.Height/2)+4,
			title,
		).Check(); err != nil {
			slog.Error("failed to properly render window title in tab err", slog.String("error", err.Error()))
			return
		}
	}
}

func (wm *Wm) renderBottomBarWindow() {
	width := 20

	for i, ws := range wm.workspaces {
		startingX := i * width // TODO: update this

		fillGc := wm.gcCache.Bottom.Inactive.Fill
		textGc := wm.gcCache.Bottom.Inactive.Text

		if i == wm.activeWorkspace {
			fillGc = wm.gcCache.Bottom.Active.Fill
			textGc = wm.gcCache.Bottom.Active.Text
		}

		if err := xproto.PolyFillRectangleChecked(
			wm.conn, xproto.Drawable(ws.bottomBar),
			fillGc,
			[]xproto.Rectangle{{
				X: int16(startingX), Y: 0,
				Width: uint16(width), Height: wm.config.BottomBarConfig.Height,
			}},
		).Check(); err != nil {
			slog.Error("failed to properly render fill rectangle in bottom bar", slog.String("error", err.Error()))
			return
		}

		wsStr := strconv.Itoa(i + 1)

		if err := xproto.ImageText8Checked(
			wm.conn, byte(len(wsStr)), xproto.Drawable(ws.bottomBar), textGc,
			int16(startingX)+8, int16(wm.config.BottomBarConfig.Height/2)+4,
			wsStr,
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
