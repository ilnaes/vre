package vre

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

func csi(s string) {
	fmt.Fprintf(os.Stderr, "\x1b["+s)
}

const console string = "/dev/tty"

// Terminal acts as the view
type Terminal struct {
	fd        int
	origState *terminal.State
	mainEb    *EventBox
	mu        sync.Mutex

	width   int
	height  int
	posY    int
	xOffset int

	prompt string
	input  string
	misc   []string
	chunks *[]*Chunk
}

func NewTerminal(eb *EventBox) *Terminal {
	return &Terminal{
		mainEb: eb,
		mu:     sync.Mutex{},
	}
}

func (t *Terminal) Loop() {
	t.Init()
	inChan := make(chan int)
	winchChan := make(chan os.Signal)

	// set up signal for window resize
	signal.Notify(winchChan, syscall.SIGWINCH)

	go t.getch(inChan)

	done := false
	for !done {
		// read a byte from terminal input
		select {
		case <-winchChan:
			t.GetSize()
			t.Refresh()

		case b := <-inChan:
			switch b {
			case KEY_CTRLC:
				// Ctrl-C quit
				t.Close()
				t.mainEb.Put(EvtQuit, nil)
				done = true

			case KEY_LEFT:
				if t.xOffset < len(t.input) {
					t.xOffset++
					t.Refresh()
				}

			case KEY_RIGHT:
				if t.xOffset > 0 {
					t.xOffset--
					t.Refresh()
				}

			case KEY_DEL:
				if len(t.input) > 0 {
					if t.xOffset < len(t.input) {
						t.input = t.input[0:len(t.input)-t.xOffset-1] + t.input[len(t.input)-t.xOffset:]
						t.Refresh()
					}
				}

			default:
				if b >= 20 && b <= 126 {
					// printable chars
					if t.xOffset == len(t.input) {
						t.input = string(b) + t.input
					} else {
						t.input = t.input[0:len(t.input)-t.xOffset] + string(b) + t.input[len(t.input)-t.xOffset:]
					}
					t.Refresh()
				}
			}
		}
	}
}

func (t *Terminal) getch(ch chan int) {
	b := make([]byte, 1)

	done := false
	for !done {
		syscall.Read(t.fd, b)

		if b[0] == KEY_ESC {
			// escaped
			syscall.Read(t.fd, b)

			if b[0] != 91 {
				// not escape sequence
				ch <- int(b[0])
				continue
			}

			syscall.Read(t.fd, b)

			if b[0] == 68 {
				ch <- KEY_LEFT
			} else if b[0] == 67 {
				ch <- KEY_RIGHT
			}
		} else {
			ch <- int(b[0])

			if int(b[0]) == KEY_CTRLC {
				done = true
			}
		}
	}
}

// Refresh clears screen and prints contents
func (t *Terminal) Refresh() {
	t.mu.Lock()

	csi("2J")
	csi("H")

	buf := ""

	nrows := 0
	for _, ch := range *t.chunks {
		for i := 0; i < ch.num; i++ {
			buf += ch.lines[i] + "\r\n"
			nrows++

			if nrows > t.height-2 {
				break
			}
		}
		if nrows > t.height-2 {
			break
		}
	}

	for j := 0; j < t.height-nrows-1; j++ {
		buf += "\r\n"
	}

	// prompt
	buf += "\x1b[31;1m> \x1b[0m\x1b[37;1m" + t.input + "\x1b[0m"
	if t.xOffset > 0 {
		buf += "\x1b[" + strconv.Itoa(t.xOffset) + "D"
	}

	t.mu.Unlock()

	fmt.Fprintf(os.Stderr, buf)
}

// UpdateChunks saves input snapshot
func (t *Terminal) UpdateChunks(chunks *[]*Chunk) {
	t.mu.Lock()
	t.chunks = chunks
	t.mu.Unlock()

	t.Refresh()
}

// GetSize gets the size of the terminal
func (t *Terminal) GetSize() {
	w, h, err := terminal.GetSize(t.fd)
	if err != nil {
		log.Fatal("Could not get terminal size")
	}
	t.width = w
	t.height = h
}

// Init saves current state of terminal, sets up raw mode and alternate screen buffer
func (t *Terminal) Init() {
	tty, err := os.OpenFile(console, syscall.O_RDONLY, 0)
	if err != nil {
		log.Fatal("Could not open file")
	}
	t.fd = int(tty.Fd())

	origState, err := terminal.GetState(t.fd)
	if err != nil {
		log.Fatal("Could not get terminal state")
	}

	t.origState = origState
	t.GetSize()
	terminal.MakeRaw(t.fd)
	t.GetSize()

	csi("?1049h")
}

// Close closes alternate screen buffer and restores original terminal state
func (t *Terminal) Close() {
	csi("?1049l")
	terminal.Restore(t.fd, t.origState)
}
