package wm

import (
	"fmt"
	"slices"

	"github.com/0xmukesh/tatami/internal/config"
	"github.com/0xmukesh/tatami/internal/constants"
	"github.com/jezek/xgb/xproto"
)

func (wm *Wm) setupBottomBar() (xproto.Window, error) {
	bottomBar, err := xproto.NewWindowId(wm.conn)
	if err != nil {
		return xproto.BadWindow, fmt.Errorf("failed to assign bottom bar window id - %w", err)
	}

	if err := xproto.CreateWindowChecked(
		wm.conn,
		wm.screen.RootDepth,
		bottomBar,
		wm.root,
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
		return xproto.BadWindow, fmt.Errorf("failed to create bottom bar window - %w", err)
	}

	if err := xproto.MapWindowChecked(wm.conn, bottomBar).Check(); err != nil {
		return xproto.BadWindow, fmt.Errorf("failed to map bottom bar window - %w", err)
	}

	return bottomBar, nil
}

func (wm *Wm) createBarGcState(cfg config.BarConfig) (GcState, error) {
	activeFill, err := wm.createGraphicalContext(xproto.GcForeground, []uint32{cfg.ActiveBg})
	if err != nil {
		return GcState{}, fmt.Errorf("active fill: %w", err)
	}

	inactiveFill, err := wm.createGraphicalContext(xproto.GcForeground, []uint32{cfg.InactiveBg})
	if err != nil {
		return GcState{}, fmt.Errorf("inactive fill: %w", err)
	}

	activeText, err := wm.createGraphicalContext(xproto.GcForeground|xproto.GcBackground, []uint32{cfg.ActiveText, cfg.ActiveBg})
	if err != nil {
		return GcState{}, fmt.Errorf("active text: %w", err)
	}

	inactiveText, err := wm.createGraphicalContext(xproto.GcForeground|xproto.GcBackground, []uint32{cfg.InactiveText, cfg.InactiveBg})
	if err != nil {
		return GcState{}, fmt.Errorf("inactive text: %w", err)
	}

	return GcState{
		Active:   GcPair{Fill: activeFill, Text: activeText},
		Inactive: GcPair{Fill: inactiveFill, Text: inactiveText},
	}, nil
}

func (wm *Wm) setupGcCache() (gc GcCache, err error) {
	gc.Tab, err = wm.createBarGcState(wm.config.TabBarConfig)
	if err != nil {
		return gc, fmt.Errorf("failed to setup tab bar gc cache - %w", err)
	}

	gc.Bottom, err = wm.createBarGcState(wm.config.BottomBarConfig)
	if err != nil {
		return gc, fmt.Errorf("failed to setup bottom bar gc cache - %w", err)
	}

	return gc, nil
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
