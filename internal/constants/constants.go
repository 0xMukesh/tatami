package constants

import (
	"github.com/jezek/xgb/xproto"
)

const (
	KB_MOD1     = xproto.ModMask1
	KB_MOD2     = xproto.ModMask2
	KB_MOD3     = xproto.ModMask3
	KB_MOD4     = xproto.ModMask4
	KB_MOD5     = xproto.ModMask5
	KB_MODSHIFT = xproto.ModMaskShift
	KB_MODCTRL  = xproto.ModMaskControl

	KB_Q = 24
	KB_W = 25
	KB_E = 26
	KB_R = 27
	KB_T = 28
	KB_Y = 29
	KB_U = 30
	KB_I = 31
	KB_O = 32
	KB_P = 33
	KB_A = 38
	KB_S = 39
	KB_D = 40
	KB_F = 41
	KB_G = 42
	KB_H = 43
	KB_J = 44
	KB_K = 45
	KB_L = 46
	KB_Z = 52
	KB_X = 53
	KB_C = 54
	KB_V = 55
	KB_B = 56
	KB_N = 57
	KB_M = 58

	KB_NUM1 = 10
	KB_NUM2 = 11
	KB_NUM3 = 12
	KB_NUM4 = 13
	KB_NUM5 = 14
	KB_NUM6 = 15
	KB_NUM7 = 16
	KB_NUM8 = 17
	KB_NUM9 = 18
	KB_NUM0 = 19

	KB_F1     = 67
	KB_F2     = 68
	KB_F3     = 69
	KB_F4     = 70
	KB_F5     = 71
	KB_F6     = 72
	KB_F7     = 73
	KB_F8     = 74
	KB_F9     = 75
	KB_F10    = 76
	KB_F11    = 95
	KB_F12    = 96
	KB_INS    = 118
	KB_PRTSCR = 107
	KB_DEL    = 119

	KB_ESC   = 9
	KB_ENTER = 36

	KB_UP    = 111
	KB_LEFT  = 113
	KB_RIGHT = 114
	KB_DOWN  = 116

	ATOM_WM_PROTOCOLS     = "WM_PROTOCOLS"
	ATOM_WM_DELETE_WINDOW = "WM_DELETE_WINDOW"
	ATOM_NET_WM_NAME      = "_NET_WM_NAME"
	ATOM_TYPE_UTF8_STRING = "UTF8_STRING"
)

var ValidModifiers = []string{"mod1", "mod2", "mod3", "shift", "ctrl", "esc", "escape", "enter"}
var KeycodeMap = map[string]uint16{
	"mod1":  KB_MOD1,
	"mod2":  KB_MOD2,
	"mod3":  KB_MOD3,
	"mod4":  KB_MOD4,
	"mod5":  KB_MOD5,
	"shift": KB_MODSHIFT,
	"ctrl":  KB_MODCTRL,

	"q": KB_Q,
	"w": KB_W,
	"e": KB_E,
	"r": KB_R,
	"t": KB_T,
	"y": KB_Y,
	"u": KB_U,
	"i": KB_I,
	"o": KB_O,
	"p": KB_P,
	"a": KB_A,
	"s": KB_S,
	"d": KB_D,
	"f": KB_F,
	"g": KB_G,
	"h": KB_H,
	"j": KB_J,
	"k": KB_K,
	"l": KB_L,
	"z": KB_Z,
	"x": KB_X,
	"c": KB_C,
	"v": KB_V,
	"b": KB_B,
	"n": KB_N,
	"m": KB_M,

	"1": KB_NUM1,
	"2": KB_NUM2,
	"3": KB_NUM3,
	"4": KB_NUM4,
	"5": KB_NUM5,
	"6": KB_NUM6,
	"7": KB_NUM7,
	"8": KB_NUM8,
	"9": KB_NUM9,
	"0": KB_NUM0,

	"f1":     KB_F1,
	"f2":     KB_F2,
	"f3":     KB_F3,
	"f4":     KB_F4,
	"f5":     KB_F5,
	"f6":     KB_F6,
	"f7":     KB_F7,
	"f8":     KB_F8,
	"f9":     KB_F9,
	"f10":    KB_F10,
	"f11":    KB_F11,
	"f12":    KB_F12,
	"prtscr": KB_PRTSCR,
	"del":    KB_DEL,

	"esc":    KB_ESC,
	"escape": KB_ESC,
	"enter":  KB_ENTER,
	"return": KB_ENTER,

	"left":  KB_LEFT,
	"right": KB_RIGHT,
	"up":    KB_UP,
	"down":  KB_DOWN,
}
