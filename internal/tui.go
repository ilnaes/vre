package vre

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
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

	width  int
	height int
	posY   int

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
	inChan := make(chan byte)

	go t.getch(inChan)

	done := false
	for !done {
		// read a byte from terminal input
		select {
		case b := <-inChan:
			switch b {
			case 97:
				t.Close()
				t.mainEb.Put(EvtQuit, nil)
			default:
				t.misc = append(t.misc, strconv.Itoa(int(b)))
				t.Refresh()
			}
		}
	}
}

func (t *Terminal) getch(ch chan byte) {
	b := make([]byte, 1)

	done := false
	for !done {
		syscall.Read(t.fd, b)
		ch <- b[0]

		if int(b[0]) == 17 {
			done = true
		}
	}
}

func (t *Terminal) Refresh() {
	csi("2J")
	csi("H")

	buf := ""

	for _, ch := range *t.chunks {
		for i := 0; i < ch.num; i++ {
			buf += ch.lines[i] + "\r\n"
		}
	}

	for _, s := range t.misc {
		buf += s + "\r\n"
	}

	fmt.Fprintf(os.Stderr, buf)
}

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

	csi("?1049h")
	csi("2J")
	csi("H")

	// fmt.Fprintf(os.Stderr, "\x1b[?1049h")
	// fmt.Fprintf(os.Stderr, "\x1b[2J")
	// fmt.Fprintf(os.Stderr, "\x1b[H")
}

// Close closes alternate screen buffer and restores original terminal state
func (t *Terminal) Close() {
	csi("?1049l")
	terminal.Restore(t.fd, t.origState)
}
