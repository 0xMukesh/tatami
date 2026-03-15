package constants

import "github.com/jezek/xgb/xproto"

const (
	KB_MOD1     = xproto.ModMask1
	KB_MOD2     = xproto.ModMask2
	KB_MOD3     = xproto.ModMask3
	KB_MOD4     = xproto.ModMask4
	KB_MOD5     = xproto.ModMask5
	KB_MODSHIFT = xproto.ModMaskShift
	KB_MODCTRL  = xproto.ModMaskControl

	KB_D     = 40
	KB_Q     = 24
	KB_ESC   = 9
	KB_ENTER = 36

	DEFAULT_BACKGROUND = 0x00000

	ATOM_WM_PROTOCOLS     = "WM_PROTOCOLS"
	ATOM_WM_DELETE_WINDOW = "WM_DELETE_WINDOW"
)
