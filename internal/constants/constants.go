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

	KB_D           = 40
	KB_Q           = 24
	KB_ESC         = 9
	KB_ENTER       = 36
	KB_LEFT_ARROW  = 113
	KB_RIGHT_ARROW = 114

	TAB_BAR_HEIGHT        = 24
	BOTTOM_BAR_HEIGHT     = 20
	DEFAULT_BG            = 0x000000
	TAB_BAR_BG            = 0xffffff
	TAB_BAR_ACTIVE_BG     = 0x00a3cc
	TAB_BAR_INACTIVE_BG   = 0x262626
	TAB_BAR_ACTIVE_TEXT   = 0xffffff
	TAB_BAR_INACTIVE_TEXT = 0xa3a3a3

	ATOM_WM_PROTOCOLS     = "WM_PROTOCOLS"
	ATOM_WM_DELETE_WINDOW = "WM_DELETE_WINDOW"
	ATOM_NET_WM_NAME      = "_NET_WM_NAME"
	ATOM_TYPE_UTF8_STRING = "UTF8_STRING"
)
