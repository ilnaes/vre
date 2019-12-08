package vre

const (
	EvtReadNew EventType = iota
	EvtReadDone
	EvtQuit
	EvtSearchNew
	EvtFinish
	EvtSearchProgress
	EvtSearchCancel
)

const (
	KEY_CTRLC = 3
	KEY_CTRLJ = 10
	KEY_CTRLK = 11
	KEY_CTRLH = 8
	KEY_CTRLF = 6
	KEY_CTRLB = 2
	KEY_CTRLL = 12
	KEY_ESC   = 27
	KEY_DEL   = 127
	KEY_LEFT  = 279168
	KEY_RIGHT = 279167
)
