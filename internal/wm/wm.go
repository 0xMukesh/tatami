package wm

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"reflect"
	"slices"

	"github.com/0xmukesh/tatami/internal/config"
	"github.com/0xmukesh/tatami/internal/constants"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Workspace struct {
	frame   xproto.Window
	tabBar  xproto.Window
	clients []xproto.Window
	active  int
}

type GcCache struct {
	InactiveTabBar xproto.Gcontext
	ActiveTabBar   xproto.Gcontext
}

type Wm struct {
	conn            *xgb.Conn
	setup           *xproto.SetupInfo
	screen          *xproto.ScreenInfo
	root            xproto.Window
	activeWorkspace int
	workspaces      map[int]*Workspace
	atoms           map[string]xproto.Atom
	gcCache         GcCache
	mod             uint16
	launcher        string
	display         string
}

func New(wmConfig config.WmConfig) (*Wm, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to x server - %w", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	root := screen.Root
	display := os.Getenv("DISPLAY")

	wm := &Wm{
		conn:            conn,
		setup:           setup,
		screen:          screen,
		root:            root,
		activeWorkspace: 0,
		workspaces:      make(map[int]*Workspace),
		atoms:           make(map[string]xproto.Atom),
		mod:             wmConfig.ModifierMask,
		launcher:        wmConfig.Launcher,
		display:         display,
	}

	if err := xproto.ChangeWindowAttributesChecked(
		wm.conn,
		wm.root,
		xproto.CwEventMask,
		[]uint32{xproto.EventMaskKeyPress |
			xproto.EventMaskKeyRelease |
			xproto.EventMaskButtonPress |
			xproto.EventMaskButtonRelease |
			xproto.EventMaskStructureNotify |
			xproto.EventMaskSubstructureRedirect,
		},
	).Check(); err != nil {
		if _, ok := err.(xproto.AccessError); ok {
			return nil, fmt.Errorf("could not become wm. is another instance of wm already running?")
		}

		return nil, fmt.Errorf("could not become wm - %w", err)
	}

	defaultWorkspace, err := wm.createWorkspace(0)
	if err != nil {
		return nil, fmt.Errorf("failed to create default workspace - %w", err)
	}
	wm.workspaces[0] = defaultWorkspace

	gcCache, err := wm.createTabBarGraphicalContexts()
	if err != nil {
		return nil, fmt.Errorf("failed to create gc cache - %w", err)
	}
	wm.gcCache = gcCache

	wm.getAndCacheAtoms([]string{
		constants.ATOM_WM_PROTOCOLS, constants.ATOM_WM_DELETE_WINDOW, constants.ATOM_NET_WM_NAME, constants.ATOM_TYPE_UTF8_STRING,
	})

	wm.registerShortcuts(wm.root)
	return wm, nil
}

func (wm *Wm) Run() {
	for {
		ev, err := wm.conn.WaitForEvent()
		if err != nil {
			slog.Error("request failed", slog.String("error", err.Error()))
			continue
		} else {
			if ev == nil {
				break
			}
		}

		switch v := ev.(type) {
		case xproto.KeyPressEvent:
			wm.handleKeyPressEvent(v)
		case xproto.ExposeEvent:
			wm.handleExposeEvent(v)
		case xproto.ConfigureRequestEvent:
			wm.handleConfigureRequest(v)
		case xproto.MapRequestEvent:
			wm.handleMapRequest(v)
		case xproto.UnmapNotifyEvent:
			wm.handleUnmapNotify(v)
		case xproto.DestroyNotifyEvent:
			wm.handleDestroyNotify(v)
		}
	}
}

func (wm *Wm) Close() {
	gcCacheValue := reflect.ValueOf(wm.gcCache)
	for _, v := range gcCacheValue.Fields() {
		if gc, ok := v.Interface().(xproto.Gcontext); ok {
			xproto.FreeGC(wm.conn, gc)
		}
	}

	wm.conn.Close()
}

// core handlers
func (wm *Wm) handleKeyPressEvent(v xproto.KeyPressEvent) {
	keycode := int(v.Detail)
	mod := v.State

	switch {
	case keycode == constants.KB_D && mod == wm.mod:
		if err := exec.Command(wm.launcher).Start(); err != nil {
			slog.Error("failed to launch app launcher", slog.String("error", err.Error()))
			return
		}
	case keycode == constants.KB_Q && mod == wm.mod:
		ws := wm.workspaces[wm.activeWorkspace]
		if len(ws.clients) > 0 {
			wm.closeWindow(ws.clients[ws.active])
		}
	case keycode == constants.KB_LEFT_ARROW && mod == wm.mod:
		wm.handleWindowNavigation(true)
	case keycode == constants.KB_RIGHT_ARROW && mod == wm.mod:
		wm.handleWindowNavigation(false)
	}
}

func (wm *Wm) handleWindowNavigation(isLeft bool) {
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

func (wm *Wm) handleExposeEvent(v xproto.ExposeEvent) {
	if v.Count != 0 {
		return
	}

	ws := wm.workspaces[wm.activeWorkspace]
	if v.Window == ws.tabBar {
		wm.renderTabBarWindow()
	}
}

func (wm *Wm) handleConfigureRequest(v xproto.ConfigureRequestEvent) {
	ws := wm.getWorkspaceByWindow(v.Window)

	x := int16(0)
	y := int16(constants.TAB_BAR_HEIGHT)
	width := wm.screen.WidthInPixels
	height := wm.screen.HeightInPixels - uint16(constants.TAB_BAR_HEIGHT)

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
		wm.conn, win, ws.frame, 0, int16(constants.TAB_BAR_HEIGHT),
	).Check(); err != nil {
		slog.Error("failed to reparent client window", slog.String("error", err.Error()))
		return
	}

	if err := xproto.ConfigureWindowChecked(
		wm.conn,
		win,
		xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
		[]uint32{
			0, constants.TAB_BAR_HEIGHT,
			uint32(wm.screen.WidthInPixels),
			uint32(wm.screen.HeightInPixels) - uint32(constants.TAB_BAR_HEIGHT),
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

		if !slices.Contains(ws.clients, win) {
			i := max(0, min(ws.active+1, len(ws.clients)))
			ws.clients = append(ws.clients, 0)
			copy(ws.clients[i+1:], ws.clients[i:])
			ws.clients[i] = win
			ws.active = i

			wm.registerShortcuts(win)
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

// helpers
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

// utils
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
			constants.DEFAULT_BG,
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
		wm.screen.WidthInPixels, constants.TAB_BAR_HEIGHT,
		0,
		xproto.WindowClassInputOutput,
		wm.screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			constants.TAB_BAR_INACTIVE_BG,
			xproto.EventMaskExposure,
		},
	).Check(); err != nil {
		return ws, fmt.Errorf("failed to create tab bar window - %w", err)
	}

	if err := xproto.MapWindowChecked(wm.conn, frame).Check(); err != nil {
		return ws, fmt.Errorf("failed to map frame window - %w", err)
	}

	wm.registerShortcuts(frame)

	return &Workspace{
		frame:  frame,
		tabBar: tabBar,
	}, nil
}

func (wm *Wm) createTabBarGraphicalContexts() (gc GcCache, err error) {
	activeTabBar, err := xproto.NewGcontextId(wm.conn)
	if err != nil {
		return gc, fmt.Errorf("failed to assign active tab gc id - %w", err)
	}

	inactiveTabBar, err := xproto.NewGcontextId(wm.conn)
	if err != nil {
		return gc, fmt.Errorf("failed to assign inactive tab gc id - %w", err)
	}

	if err := xproto.CreateGCChecked(
		wm.conn, activeTabBar, xproto.Drawable(wm.root),
		xproto.GcForeground|xproto.GcBackground,
		[]uint32{constants.TAB_BAR_FG, constants.TAB_BAR_ACTIVE_BG},
	).Check(); err != nil {
		return gc, fmt.Errorf("failed to create active tab bar gc - %w", err)
	}

	if err := xproto.CreateGCChecked(
		wm.conn, inactiveTabBar, xproto.Drawable(wm.root),
		xproto.GcForeground|xproto.GcBackground,
		[]uint32{constants.TAB_BAR_FG, constants.TAB_BAR_INACTIVE_BG},
	).Check(); err != nil {
		return gc, fmt.Errorf("failed to create inactive tab bar gc - %w", err)
	}

	return GcCache{
		InactiveTabBar: inactiveTabBar,
		ActiveTabBar:   activeTabBar,
	}, nil
}

func (wm *Wm) registerShortcuts(window xproto.Window) {
	xproto.GrabKey(wm.conn, true, window, wm.mod, constants.KB_D, xproto.GrabModeAsync, xproto.GrabModeAsync)

	xproto.GrabKey(wm.conn, true, window, wm.mod, constants.KB_Q, xproto.GrabModeAsync, xproto.GrabModeAsync)
}

func (wm *Wm) getAndCacheAtoms(properties []string) error {
	for _, p := range properties {
		reply, err := xproto.InternAtom(wm.conn, true, uint16(len(p)), p).Reply()
		if err != nil {
			return fmt.Errorf("failed to get atom for %v - %w", p, err)
		}

		wm.atoms[p] = reply.Atom
	}

	return nil
}

func (wm *Wm) doesSupportDeleteProtocol(window xproto.Window) bool {
	atomWmProtocol := wm.atoms[constants.ATOM_WM_PROTOCOLS]
	if atomWmProtocol == xproto.AtomNone {
		return false
	}

	atomWmDeleteWindow := wm.atoms[constants.ATOM_WM_DELETE_WINDOW]
	if atomWmDeleteWindow == xproto.AtomNone {
		return false
	}

	prop, err := xproto.GetProperty(wm.conn, false, window, atomWmProtocol, xproto.AtomAtom, 0, 32).Reply()
	if err != nil || prop.ValueLen == 0 {
		return false
	}

	for i := 0; i < int(prop.ValueLen); i++ {
		atom := xproto.Atom(
			uint32(prop.Value[i*4]) |
				uint32(prop.Value[i*4+1])<<8 |
				uint32(prop.Value[i*4+2])<<16 |
				uint32(prop.Value[i*4+3])<<24,
		)

		if atom == atomWmDeleteWindow {
			return true
		}
	}

	return false
}

func (wm *Wm) getWindowTitle(window xproto.Window) string {
	atomNetWmName := wm.atoms[constants.ATOM_NET_WM_NAME]
	atomTypeUtf8String := wm.atoms[constants.ATOM_TYPE_UTF8_STRING]

	prop, err := xproto.GetProperty(wm.conn, false, window, atomNetWmName, atomTypeUtf8String, 0, 256).Reply()
	if err == nil && prop.ValueLen > 0 {
		return string(prop.Value)
	}

	prop, err = xproto.GetProperty(wm.conn, false, window, xproto.AtomWmName, xproto.AtomString, 0, 256).Reply()
	if err == nil && prop.ValueLen > 0 {
		return string(prop.Value)
	}

	return "Untitled"
}

func (wm *Wm) getWorkspaceByWindow(window xproto.Window) *Workspace {
	for _, ws := range wm.workspaces {
		if slices.Contains(ws.clients, window) {
			return ws
		}
	}

	return nil
}
