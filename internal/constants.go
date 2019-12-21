package vre

// number of spaces in a tab
var TABSTOP int = 8

// number of lines in a chunk
const ChunkSize int = 250

const console string = "/dev/tty"

const fileColor = "\x1b[35;1m"
const matchColor = "\x1b[32;1m"

const (
	EvtReadNew EventType = iota
	EvtReadDone
	EvtReadError
	EvtQuit
	EvtSearchNew
	EvtSearchFinal
	EvtSearchProgress
	EvtSearchCancel
	EvtHeartbeat
	EvtFinish
)

const (
	KEY_CTRLB     = 2
	KEY_CTRLC     = 3
	KEY_CTRLD     = 4
	KEY_CTRLF     = 6
	KEY_CTRLH     = 8
	KEY_CTRLJ     = 10
	KEY_CTRLK     = 11
	KEY_CTRLL     = 12
	KEY_ENTER     = 13
	KEY_CTRLT     = 20
	KEY_ESC       = 27
	KEY_BACKSPACE = 127
	KEY_LEFT      = 279168
	KEY_RIGHT     = 279167
	KEY_DEL       = 2791126
)
