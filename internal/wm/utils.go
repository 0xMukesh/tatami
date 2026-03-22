package wm

import (
	"fmt"
	"slices"

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

	if err := xproto.MapWindowChecked(wm.conn, frame).Check(); err != nil {
		return ws, fmt.Errorf("failed to map frame window - %w", err)
	}

	wm.registerShortcuts(frame)

	return &Workspace{
		frame:  frame,
		tabBar: tabBar,
	}, nil
}

func (wm *Wm) setupGcCache() (gc GcCache, err error) {
	activeFill, err := wm.createGraphicalContext(xproto.GcForeground, []uint32{wm.config.TabBarConfig.ActiveBg})
	if err != nil {
		return gc, fmt.Errorf("failed to create active fill gc - %w", err)
	}

	inactiveFill, err := wm.createGraphicalContext(xproto.GcForeground, []uint32{wm.config.TabBarConfig.InactiveBg})
	if err != nil {
		return gc, fmt.Errorf("failed to create inactive fill gc - %w", err)
	}

	activeText, err := wm.createGraphicalContext(xproto.GcForeground|xproto.GcBackground, []uint32{wm.config.TabBarConfig.ActiveText, wm.config.TabBarConfig.ActiveBg})
	if err != nil {
		return gc, fmt.Errorf("failed to create active text gc - %w", err)
	}

	inactiveText, err := wm.createGraphicalContext(xproto.GcForeground|xproto.GcBackground, []uint32{wm.config.TabBarConfig.InactiveText, wm.config.TabBarConfig.InactiveBg})
	if err != nil {
		return gc, fmt.Errorf("failed to create inactive text gc - %w", err)
	}

	return GcCache{
		activeFill:   activeFill,
		activeText:   activeText,
		inactiveFill: inactiveFill,
		inactiveText: inactiveText,
	}, nil
}

func (wm *Wm) registerShortcuts(window xproto.Window) {
	for _, kb := range wm.keybindings {
		xproto.GrabKey(wm.conn, true, window, kb.mod, xproto.Keycode(kb.keycode), xproto.GrabModeAsync, xproto.GrabModeAsync)
	}
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

func (wm *Wm) createGraphicalContext(valueMask uint32, valueList []uint32) (xproto.Gcontext, error) {
	gcId, err := xproto.NewGcontextId(wm.conn)
	if err != nil {
		return xproto.BadGContext, fmt.Errorf("failed to assign graphical context id - %w", err)
	}

	if err := xproto.CreateGCChecked(
		wm.conn, gcId, xproto.Drawable(wm.root),
		valueMask, valueList,
	).Check(); err != nil {
		return xproto.BadGContext, fmt.Errorf("failed to create graphical context - %w", err)
	}

	return gcId, nil
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

func (wm *Wm) rearrangeWindows(wid int, to int) {
	ws := wm.workspaces[wm.activeWorkspace]
	if ws == nil {
		return
	}

	window := ws.clients[wid]
	ws.clients = slices.Delete(ws.clients, wid, wid+1)
	ws.clients = slices.Insert(ws.clients, to, window)
}
