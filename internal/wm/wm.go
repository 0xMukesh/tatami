package wm

import (
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/0xmukesh/tatami/internal/config"
	"github.com/0xmukesh/tatami/internal/constants"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Wm struct {
	conn       *xgb.Conn
	setup      *xproto.SetupInfo
	screen     *xproto.ScreenInfo
	rootWindow xproto.Window

	// mapping between top-level client windows and frame windows
	clients map[xproto.Window]xproto.Window

	// additional configuration
	mod         uint16
	launcher    string
	borderWidth uint16
}

func New(wmConfig config.WmConfig) (*Wm, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to x server. error - %w", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	root := screen.Root

	wm := &Wm{
		conn:        conn,
		setup:       setup,
		screen:      screen,
		rootWindow:  root,
		mod:         wmConfig.ModifierMask,
		launcher:    wmConfig.Launcher,
		borderWidth: wmConfig.BorderWidth,
	}

	if err := wm.attachAsWm(); err != nil {
		return nil, fmt.Errorf("failed to attach as wm. error - %w", err)
	}

	return wm, nil
}

func (wm *Wm) Run() {
	for {
		ev, err := wm.conn.WaitForEvent()
		if err != nil {
			slog.Error("X server request failed", slog.String("error", err.Error()))
			continue
		} else {
			if ev == nil {
				break
			}
		}

		switch v := ev.(type) {
		case xproto.KeyPressEvent:
			wm.handleKeyPressEvent(v)
		case xproto.ConfigureRequestEvent:
			wm.handleConfigureRequest(v)
		case xproto.MapRequestEvent:
			wm.handleMapRequest(v)
		case xproto.UnmapNotifyEvent:
			wm.handleUnmapNotify(v)
		}
	}

}

func (wm *Wm) Close() {
	wm.conn.Close()
}

func (wm *Wm) attachAsWm() error {
	if err := xproto.ChangeWindowAttributesChecked(
		wm.conn,
		wm.rootWindow,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease |
				xproto.EventMaskButtonPress |
				xproto.EventMaskButtonRelease |
				xproto.EventMaskStructureNotify |
				xproto.EventMaskSubstructureRedirect,
		},
	).Check(); err != nil {
		if _, ok := err.(xproto.AccessError); ok {
			return fmt.Errorf("could not become wm. is another instance of wm already running?")
		}

		return fmt.Errorf("could not become wm. error - %w", err)
	}

	wm.registerShortcuts(wm.rootWindow)
	return nil
}

func (wm *Wm) registerShortcuts(window xproto.Window) {
	xproto.GrabKey(
		wm.conn,
		true,
		window,
		wm.mod,
		constants.KB_D,
		xproto.GrabModeAsync,
		xproto.GrabModeAsync,
	)
}

func (wm *Wm) handleKeyPressEvent(v xproto.KeyPressEvent) {
	keycode := int(v.Detail)
	mod := v.State

	switch {
	case keycode == constants.KB_ESC:
		wm.Close()
	case keycode == constants.KB_D && mod == wm.mod:
		exec.Command(wm.launcher).Start()
	}
}

func (wm *Wm) handleConfigureRequest(v xproto.ConfigureRequestEvent) {
	event := xproto.ConfigureNotifyEvent{
		Event:            v.Window,
		Window:           v.Window,
		AboveSibling:     0,
		X:                0,
		Y:                0,
		Width:            v.Width,
		Height:           v.Height,
		BorderWidth:      0,
		OverrideRedirect: false,
	}

	if err := xproto.SendEventChecked(
		wm.conn,
		false,
		v.Window,
		xproto.EventMaskStructureNotify,
		event.String(),
	).Check(); err != nil {
		slog.Error("failed to properly configure client window", slog.String("error", err.Error()))
	}

	if frame, ok := wm.clients[v.Window]; ok {
		if err := xproto.SendEventChecked(
			wm.conn,
			false,
			frame,
			uint32(v.ValueMask),
			event.String(),
		).Check(); err != nil {
			slog.Error("failed to properly configure frame window", slog.String("error", err.Error()))
		}
	}
}

func (wm *Wm) handleMapRequest(v xproto.MapRequestEvent) {
	child := v.Window

	winattrib, err := xproto.GetWindowAttributes(wm.conn, child).Reply()
	if err != nil {
		slog.Error("failed to get window attributes", slog.String("error", err.Error()))
		return
	}

	geo, err := xproto.GetGeometry(wm.conn, xproto.Drawable(child)).Reply()
	if err != nil {
		slog.Error("failed to get geometry of window", slog.String("error", err.Error()))
		return
	}

	frame, err := xproto.NewWindowId(wm.conn)
	if err != nil {
		slog.Error("failed to create a frame window", slog.String("error", err.Error()))
		return
	}

	colorReply, err := xproto.AllocColor(wm.conn, wm.screen.DefaultColormap, 0xffff, 0x0000, 0x0000).Reply()
	if err != nil {
		slog.Error("failed to alloc color", slog.String("error", err.Error()))
		return
	}

	xproto.CreateWindow(
		wm.conn,
		wm.screen.RootDepth,
		frame,
		wm.rootWindow,
		geo.X,
		geo.Y,
		geo.Width+2*wm.borderWidth,
		geo.Height+2*wm.borderWidth,
		wm.borderWidth,
		xproto.WindowClassInputOutput,
		wm.screen.RootVisual,
		xproto.CwBackPixel|xproto.CwBorderPixel|xproto.CwEventMask,
		[]uint32{
			constants.DEFAULT_BACKGROUND,
			colorReply.Pixel,
			xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify,
		},
	)

	if err := xproto.ReparentWindowChecked(
		wm.conn,
		v.Window,
		frame,
		int16(wm.borderWidth),
		int16(wm.borderWidth),
	).Check(); err != nil {
		slog.Error("failed to reparent windows", slog.String("error", err.Error()))
		return
	}

	if err := xproto.ChangeSaveSetChecked(wm.conn, xproto.SetModeInsert, v.Window).Check(); err != nil {
		slog.Error("failed to save window to save set", slog.String("error", err.Error()))
		return
	}

	// only process it if override redirect is set to be false
	if !winattrib.OverrideRedirect {
		if err := xproto.MapWindowChecked(wm.conn, frame).Check(); err != nil {
			slog.Error("failed to map frame window", slog.String("error", err.Error()))
			return
		}

		if err := xproto.MapWindowChecked(wm.conn, v.Window).Check(); err != nil {
			slog.Error("failed to map child window", slog.String("error", err.Error()))
			return
		}

		wm.clients[child] = frame
		wm.registerShortcuts(frame)
	}
}

func (wm *Wm) handleUnmapNotify(v xproto.UnmapNotifyEvent) {
	frame, ok := wm.clients[v.Window]
	if !ok {
		slog.Error("trying to unmap a non-client window. ignoring...")
		return
	}

	if err := xproto.ReparentWindowChecked(
		wm.conn,
		v.Window,
		wm.rootWindow,
		0, 0,
	).Check(); err != nil {
		slog.Error("failed to reparent client window to root", slog.String("error", err.Error()))
		return
	}

	if err := xproto.ChangeSaveSetChecked(wm.conn, xproto.SetModeDelete, v.Window).Check(); err != nil {
		slog.Error("failed to remove client window from save set", slog.String("error", err.Error()))
		return
	}

	if err := xproto.UnmapWindowChecked(wm.conn, frame).Check(); err != nil {
		slog.Error("failed to unmap frame window", slog.String("error", err.Error()))
		return
	}

	if err := xproto.DestroyWindow(wm.conn, frame).Check(); err != nil {
		slog.Error("failed to destroy frame window", slog.String("error", err.Error()))
		return
	}

	delete(wm.clients, v.Window)
}
