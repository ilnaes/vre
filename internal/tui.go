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
	mainEb    *EventBox
	width     int
	height    int
	posY      int
	prompt    string
	input     string
}

func NewTerminal(eb *EventBox) *Terminal {
	return &Terminal{
		mainEb: eb,
	}
}

func (t *Terminal) Loop() {
	t.Init()

	b := make([]byte, 1)
	done := false

	for !done {
		// read a byte from terminal input
		syscall.Read(t.fd, b)
		if int(b[0]) == 97 {
			t.Close()
			done = true
			t.mainEb.Put(EvtQuit, nil)
		}
	}
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

	fmt.Fprintf(os.Stderr, "\x1b[?1049h")
	fmt.Fprintf(os.Stderr, "\x1b[2J")
	fmt.Fprintf(os.Stderr, "\x1b[H")
}

// Close closes alternate screen buffer and restores original terminal state
func (t *Terminal) Close() {
	fmt.Fprintf(os.Stderr, "\x1b[?1049l")
	terminal.Restore(t.fd, t.origState)
}
