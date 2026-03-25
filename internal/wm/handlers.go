package wm

import (
	"log/slog"
	"slices"

	"github.com/jezek/xgb/xproto"
)

func (wm *Wm) handleKeyPressEvent(v xproto.KeyPressEvent) {
	for _, kb := range wm.keybindings {
		if v.Detail == xproto.Keycode(kb.keycode) && v.State == kb.mod {
			kb.handler()
			return
		}
	}
}

func (wm *Wm) handleFocusWindow(isLeft bool) {
	ws := wm.workspaces[wm.activeWorkspace]
	if (isLeft && ws.active <= 0) || (!isLeft && ws.active >= len(ws.clients)-1) {
		return
	}

	if err := xproto.UnmapWindowChecked(wm.conn, ws.clients[ws.active]).Check(); err != nil {
		slog.Error("failed to unmap window", slog.String("error", err.Error()))
		return
	}

	if isLeft {
		ws.active -= 1
	} else {
		ws.active += 1
	}

	if err := xproto.MapWindowChecked(wm.conn, ws.clients[ws.active]).Check(); err != nil {
		slog.Error("failed to map window", slog.String("error", err.Error()))
		return
	}

	wm.renderTabBarWindow()
}

func (wm *Wm) handleMoveWindow(isLeft bool) {
	ws := wm.workspaces[wm.activeWorkspace]
	if (isLeft && ws.active <= 0) || (!isLeft && ws.active >= len(ws.clients)-1) {
		return
	}

	if isLeft {
		wm.rearrangeWindows(ws.active, ws.active-1)
		ws.active -= 1
	} else {
		wm.rearrangeWindows(ws.active, ws.active+1)
		ws.active += 1
	}

	wm.renderTabBarWindow()
}

func (wm *Wm) handleFocusWorkspace(wsNum int) {
	currIdx := wm.activeWorkspace
	newIdx := wsNum - 1
	currWs := wm.workspaces[currIdx]
	newWs, ok := wm.workspaces[newIdx]

	if newIdx == currIdx {
		return
	}

	wm.activeWorkspace = newIdx

	if !ok {
		ws, err := wm.createWorkspace(newIdx)
		if err != nil {
			slog.Error("failed to create new workspace", slog.String("error", err.Error()))
			return
		}

		wm.workspaces[newIdx] = ws
		newWs = ws
	}

	// unmap current workspace's frame window, active window and tab bar
	if err := xproto.UnmapWindowChecked(wm.conn, currWs.frame).Check(); err != nil {
		slog.Error("failed to unmap frame window", slog.String("error", err.Error()))
		return
	}

	if err := xproto.UnmapWindowChecked(wm.conn, currWs.tabBar).Check(); err != nil {
		slog.Error("failed to unmap tab bar", slog.String("error", err.Error()))
		return
	}

	if len(currWs.clients) > 0 {
		if err := xproto.UnmapWindowChecked(wm.conn, currWs.clients[currWs.active]).Check(); err != nil {
			slog.Error("failed to unmap active window", slog.String("error", err.Error()))
			return
		}
	}

	if len(currWs.clients) == 0 {
		xproto.DestroyWindow(wm.conn, currWs.frame)
		delete(wm.workspaces, currIdx)
	}

	// map new workspace's frame window, active window and tab bar
	if err := xproto.MapWindowChecked(wm.conn, newWs.frame).Check(); err != nil {
		slog.Error("failed to map new ws frame window", slog.String("error", err.Error()))
		return
	}

	if len(newWs.clients) > 0 {
		if err := xproto.MapWindowChecked(wm.conn, newWs.clients[newWs.active]).Check(); err != nil {
			slog.Error("failed to map window", slog.String("error", err.Error()))
			return
		}
	}

	if len(newWs.clients) > 0 {
		if err := xproto.MapWindowChecked(wm.conn, newWs.tabBar).Check(); err != nil {
			slog.Error("failed to map tab bar", slog.String("error", err.Error()))
			return
		}
	}

	wm.renderBottomBarWindow()
}

func (wm *Wm) handleExposeEvent(v xproto.ExposeEvent) {
	if v.Count != 0 {
		return
	}

	ws := wm.workspaces[wm.activeWorkspace]
	switch v.Window {
	case ws.tabBar:
		wm.renderTabBarWindow()
	case wm.bottomBar:
		wm.renderBottomBarWindow()
	}
}

func (wm *Wm) handleConfigureRequest(v xproto.ConfigureRequestEvent) {
	ws := wm.getWorkspaceByWindow(v.Window)

	x := int16(0)
	y := int16(wm.config.TabBarConfig.Height)
	width := wm.screen.WidthInPixels
	height := wm.screen.HeightInPixels - uint16(wm.config.TabBarConfig.Height)

	if ws == nil {
		x = v.X
		y = v.Y
		width = v.Width
		height = v.Height
	}

	event := xproto.ConfigureNotifyEvent{
		Event:            v.Window,
		Window:           v.Window,
		AboveSibling:     0,
		X:                x,
		Y:                y,
		Width:            width,
		Height:           height,
		BorderWidth:      0,
		OverrideRedirect: false,
	}

	if err := xproto.SendEventChecked(
		wm.conn, false, v.Window,
		xproto.EventMaskStructureNotify, event.String(),
	).Check(); err != nil {
		slog.Error("failed to properly configure client window", slog.String("error", err.Error()))
		return
	}

	if ws != nil {
		frameEvent := xproto.ConfigureNotifyEvent{
			Event:            ws.frame,
			Window:           ws.frame,
			AboveSibling:     0,
			X:                0,
			Y:                0,
			Width:            wm.screen.WidthInPixels,
			Height:           wm.screen.HeightInPixels,
			BorderWidth:      0,
			OverrideRedirect: false,
		}

		if err := xproto.SendEventChecked(
			wm.conn, false, ws.frame,
			uint32(v.ValueMask), frameEvent.String(),
		).Check(); err != nil {
			slog.Error("failed to properly configure frame window", slog.String("error", err.Error()))
			return
		}
	}
}

func (wm *Wm) handleMapRequest(v xproto.MapRequestEvent) {
	win := v.Window
	ws := wm.workspaces[wm.activeWorkspace]

	winattrib, err := xproto.GetWindowAttributes(wm.conn, win).Reply()
	if err != nil {
		slog.Error("failed to get window attributes", slog.String("error", err.Error()))
		return
	}

	if winattrib.OverrideRedirect {
		xproto.MapWindow(wm.conn, win)
		return
	}

	if err := xproto.ReparentWindowChecked(
		wm.conn, win, ws.frame, 0, int16(wm.config.TabBarConfig.Height),
	).Check(); err != nil {
		slog.Error("failed to reparent client window", slog.String("error", err.Error()))
		return
	}

	screenWidth := wm.screen.WidthInPixels
	screenHeight := wm.screen.HeightInPixels
	tabBarHeight := wm.config.TabBarConfig.Height
	bottomBarHeight := wm.config.BottomBarConfig.Height

	if err := xproto.ConfigureWindowChecked(
		wm.conn,
		win,
		xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
		[]uint32{
			0, uint32(tabBarHeight),
			uint32(screenWidth), uint32(screenHeight) - uint32(tabBarHeight) - uint32(bottomBarHeight),
		},
	).Check(); err != nil {
		slog.Error("failed to configure child window", slog.String("error", err.Error()))
		return
	}

	if err := xproto.ChangeSaveSetChecked(wm.conn, xproto.SetModeInsert, win).Check(); err != nil {
		slog.Error("failed to save window to save set", slog.String("error", err.Error()))
		return
	}

	if !winattrib.OverrideRedirect {
		if len(ws.clients) == 0 {
			if err := xproto.MapWindowChecked(wm.conn, ws.tabBar).Check(); err != nil {
				slog.Error("failed to map tab bar window", slog.String("error", err.Error()))
				return
			}
		}

		for _, c := range ws.clients {
			xproto.UnmapWindow(wm.conn, c)
		}

		if err := xproto.MapWindowChecked(wm.conn, win).Check(); err != nil {
			slog.Error("failed to map child window", slog.String("error", err.Error()))
			return
		}

		wm.registerShortcuts(win)

		if !slices.Contains(ws.clients, win) {
			i := max(0, min(ws.active+1, len(ws.clients)))
			ws.clients = append(ws.clients, 0)
			copy(ws.clients[i+1:], ws.clients[i:])
			ws.clients[i] = win
			ws.active = i

			wm.renderTabBarWindow()
		}
	}
}

func (wm *Wm) handleUnmapNotify(v xproto.UnmapNotifyEvent) {
	ws := wm.getWorkspaceByWindow(v.Window)
	if ws != nil {
		return
	}

	if _, err := xproto.GetWindowAttributes(wm.conn, v.Window).Reply(); err == nil {
		if err := xproto.ReparentWindowChecked(
			wm.conn, v.Window, wm.root, 0, 0,
		).Check(); err != nil {
			slog.Error("failed to reparent client window to root", slog.String("error", err.Error()))
			return
		}
	}
}

func (wm *Wm) handleDestroyNotify(v xproto.DestroyNotifyEvent) {
	ws := wm.getWorkspaceByWindow(v.Window)
	if ws == nil {
		return
	}

	removedIndex := -1
	for i, win := range ws.clients {
		if win == v.Window {
			removedIndex = i
			ws.clients = append(ws.clients[:i], ws.clients[i+1:]...)
			break
		}
	}

	if removedIndex == -1 {
		return
	}

	if len(ws.clients) == 0 {
		ws.active = 0

		if err := xproto.SetInputFocusChecked(
			wm.conn, xproto.InputFocusPointerRoot, wm.root, xproto.TimeCurrentTime,
		).Check(); err != nil {
			slog.Error("failed to set root window as input", slog.String("error", err.Error()))
			return
		}

		if err := xproto.UnmapWindowChecked(wm.conn, ws.tabBar).Check(); err != nil {
			slog.Error("failed to unmap tab bar window", slog.String("error", err.Error()))
			return
		}

		return
	}

	if ws.active >= len(ws.clients) {
		ws.active = len(ws.clients) - 1
	}

	if ws == wm.workspaces[wm.activeWorkspace] {
		if err := xproto.MapWindowChecked(wm.conn, ws.clients[ws.active]).Check(); err != nil {
			slog.Error("failed to re-map client window", slog.String("error", err.Error()))
			return
		}

		if err := xproto.SetInputFocusChecked(
			wm.conn, xproto.InputFocusPointerRoot, ws.clients[ws.active], xproto.TimeCurrentTime,
		).Check(); err != nil {
			slog.Error("failed to focus client window", slog.String("error", err.Error()))
			return
		}
	}

	wm.renderTabBarWindow()
}
