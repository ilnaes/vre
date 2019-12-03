package vre

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"syscall"
)

const console string = "/dev/tty"

// Terminal acts as the view
type Terminal struct {
	origState *terminal.State
	fd        int
	q         *Queue
	width     int
	height    int
}

func NewTerminal(q *Queue) *Terminal {
	return &Terminal{
		q: q,
	}
}

func (t *Terminal) Loop() {
}

// GetSize gets the size of the terminal
func (t *Terminal) GetSize() {
	w, h, err := terminal.GetSize(t.fd)
	if err != nil {
		t.width = w
		t.height = h
	}
}

// Init saves current state of terminal, sets up raw mode and alternate screen buffer
func (t *Terminal) Init() {
	tty, err := os.OpenFile(console, syscall.O_RDONLY, 0)
	if err != nil {
		log.Fatal("BAD")
	}
	t.fd = int(tty.Fd())

	origState, err := terminal.GetState(t.fd)
	if err != nil {
		log.Fatal("BAD")
	}

	t.origState = origState
	terminal.MakeRaw(t.fd)

	fmt.Fprintf(os.Stderr, "\x1b[?1049h")
	fmt.Fprintf(os.Stderr, "\x1b[2J")
	fmt.Fprintf(os.Stderr, "\x1b[H")
}

// Close closes alternate screen buffer and restores original terminal state
func (t *Terminal) Close() {
	fmt.Fprintf(os.Stderr, "\x1b[?1049l")
	terminal.Restore(t.fd, t.origState)
}

func (t *Terminal) GetEvent() {
	b := make([]byte, 1)

	done := false
	for !done {
		syscall.Read(t.fd, b)
		fmt.Fprintf(os.Stderr, "%d\r\n", int(b[0]))
		if int(b[0]) == 97 {
			done = true
		}
	}
}
