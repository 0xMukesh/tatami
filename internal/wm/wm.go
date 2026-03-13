package wm

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/0xmukesh/tatami/internal/constants"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Wm struct {
	conn       *xgb.Conn
	setup      *xproto.SetupInfo
	rootWindow xproto.Window

	display string
	mod     uint16
}

func New(mod uint16) (*Wm, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to x server. error - %w", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	root := screen.Root

	if err := xproto.ChangeWindowAttributesChecked(
		conn,
		root,
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
			return nil, fmt.Errorf("could not become wm. is another instance of wm already running?")
		}

		return nil, fmt.Errorf("could not become wm. error - %w", err)
	}

	xproto.GrabKey(conn, true, root, mod, constants.KEYCODE_D, xproto.GrabModeAsync, xproto.GrabModeAsync)

	wm := &Wm{
		conn:       conn,
		setup:      setup,
		rootWindow: root,
		display:    os.Getenv("DISPLAY"),
		mod:        mod,
	}

	return wm, nil
}

func (w *Wm) Run() {
	for {
		ev, err := w.conn.WaitForEvent()
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
			w.handleKeyPressEvent(v)
		}
	}

}

func (w *Wm) Close() {
	w.conn.Close()
}

func (w *Wm) handleKeyPressEvent(v xproto.KeyPressEvent) {
	keycode := int(v.Detail)
	mod := v.State

	switch {
	case keycode == constants.KEYCODE_ESC:
		w.Close()
	case keycode == constants.KEYCODE_D && mod == w.mod:
		w.spawnWithDisplay("dmenu_run")
	}
}

func (w *Wm) spawnWithDisplay(cmd string) {
	c := exec.Command(cmd)
	c.Env = append(os.Environ(), "DISPLAY="+w.display)
	if err := c.Start(); err != nil {
		slog.Error("failed to spawn", slog.String("error", err.Error()))
	}
}
