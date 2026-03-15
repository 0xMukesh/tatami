package wm

import (
	"fmt"
	"log/slog"
	"os"
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
	mod      uint16
	launcher string

	display string
}

func New(wmConfig config.WmConfig) (*Wm, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to x server. error - %w", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	root := screen.Root
	display := os.Getenv("DISPLAY")

	wm := &Wm{
		conn:       conn,
		setup:      setup,
		screen:     screen,
		rootWindow: root,
		clients:    make(map[xproto.Window]xproto.Window),
		mod:        wmConfig.ModifierMask,
		launcher:   wmConfig.Launcher,
		display:    display,
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
	wm.conn.Close()
}

// event handlers
func (wm *Wm) handleKeyPressEvent(v xproto.KeyPressEvent) {
	keycode := int(v.Detail)
	mod := v.State

	switch {
	case keycode == constants.KB_D && mod == wm.mod:
		if err := exec.Command(wm.launcher).Start(); err != nil {
			slog.Error("failed to launch app launcher", slog.String("error", err.Error()))
			return
		}
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

	if winattrib.OverrideRedirect {
		xproto.MapWindow(wm.conn, child)
		return
	}

	frame, err := xproto.NewWindowId(wm.conn)
	if err != nil {
		slog.Error("failed to create a frame window", slog.String("error", err.Error()))
		return
	}

	if err := xproto.ConfigureWindowChecked(
		wm.conn,
		child,
		xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
		[]uint32{
			0,
			0,
			uint32(wm.screen.WidthInPixels),
			uint32(wm.screen.HeightInPixels),
		},
	).Check(); err != nil {
		slog.Error("failed to configure child window", slog.String("error", err.Error()))
		return
	}

	if err := xproto.CreateWindowChecked(
		wm.conn,
		wm.screen.RootDepth,
		frame,
		wm.rootWindow,
		0,
		0,
		wm.screen.WidthInPixels,
		wm.screen.HeightInPixels,
		0,
		xproto.WindowClassInputOutput,
		wm.screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			constants.DEFAULT_BACKGROUND,
			xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify,
		},
	).Check(); err != nil {
		slog.Error("failed to create a window", slog.String("error", err.Error()))
		return
	}

	if err := xproto.ReparentWindowChecked(
		wm.conn,
		v.Window,
		frame,
		0,
		0,
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

		wm.registerShortcuts(frame)
		wm.clients[child] = frame
	}
}

func (wm *Wm) handleUnmapNotify(v xproto.UnmapNotifyEvent) {
	wm.removeWindow(v.Window)
}

func (wm *Wm) handleDestroyNotify(v xproto.DestroyNotifyEvent) {
	wm.removeWindow(v.Window)

	if err := xproto.SetInputFocusChecked(wm.conn, xproto.InputFocusPointerRoot, wm.rootWindow, xproto.TimeCurrentTime).Check(); err != nil {
		slog.Error("failed to focus root window", slog.String("error", err.Error()))
	}
}

// utils
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

func (wm *Wm) removeWindow(window xproto.Window) {
	frame, ok := wm.clients[window]
	if !ok {
		slog.Debug("trying to remove a non-client window. ignoring...")
		return
	}

	if _, err := xproto.GetWindowAttributes(wm.conn, window).Reply(); err == nil {
		if err := xproto.ReparentWindowChecked(wm.conn, window, wm.rootWindow, 0, 0).Check(); err != nil {
			slog.Error("failed to reparent client window to root", slog.String("error", err.Error()))
			return
		}

		if err := xproto.ChangeSaveSetChecked(wm.conn, xproto.SetModeDelete, window).Check(); err != nil {
			slog.Error("failed to remove client window from save set", slog.String("error", err.Error()))
			return
		}
	}

	if err := xproto.UnmapWindowChecked(wm.conn, frame).Check(); err != nil {
		slog.Error("failed to unmap frame window", slog.String("error", err.Error()))
		return
	}

	if err := xproto.DestroyWindowChecked(wm.conn, frame).Check(); err != nil {
		slog.Error("failed to destroy frame window", slog.String("error", err.Error()))
		return
	}

	delete(wm.clients, window)
}
