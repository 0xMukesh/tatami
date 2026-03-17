package wm

import (
	"fmt"
	"log/slog"
	"reflect"

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
	keybindings     []Keybinding
	gcCache         GcCache
	config          config.WmConfig
}

func New(wmConfig config.WmConfig) (*Wm, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to x server - %w", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	root := screen.Root

	wm := &Wm{
		conn:            conn,
		setup:           setup,
		screen:          screen,
		root:            root,
		activeWorkspace: 0,
		workspaces:      make(map[int]*Workspace),
		atoms:           make(map[string]xproto.Atom),
		config:          wmConfig,
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

	wm.setupKeybindings()
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
