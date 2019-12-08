package vre

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

func csi(s string) {
	fmt.Fprintf(os.Stderr, "\x1b["+s)
}

func getLine(doc []*Chunk, bounds [][ChunkSize][][]int, ch, i int, color string) string {
	if bounds == nil || len(bounds) < ch+1 || len(bounds[ch][i]) == 0 {
		// all ways the intervals might not exist
		return doc[ch].lines[i] + "\r\n"
	}

	last := 0
	line := doc[ch].lines[i]
	buf := ""

	for _, I := range bounds[ch][i] {
		buf += line[last:I[0]] + color + line[I[0]:I[1]] + "\x1b[0m"
		last = I[1]
	}

	return buf + line[last:] + "\r\n"
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
	posX   int
	offset int // cursor offset

	prompt   string
	input    string
	misc     []string
	doc      []*Chunk
	bounds   [][ChunkSize][][]int
	numLines int
}

func NewTerminal(eb *EventBox) *Terminal {
	return &Terminal{
		mainEb: eb,
		mu:     sync.Mutex{},
	}
}

func (t *Terminal) Loop() {
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

			case KEY_CTRLJ:
				if t.posY+t.height-1 < t.numLines {
					t.posY++
					t.Refresh()
				}

			case KEY_CTRLK:
				if t.posY > 0 {
					t.posY--
					t.Refresh()
				}

			case KEY_CTRLF:
				if t.posY+t.height-1 < t.numLines {
					t.posY += t.height
					if t.posY+t.height-1 > t.numLines {
						t.posY = t.numLines - t.height + 1
					}
					t.Refresh()
				}

			case KEY_CTRLB:
				if t.posY > 0 {
					t.posY -= t.height
					if t.posY < 0 {
						t.posY = 0
					}
					t.Refresh()
				}

			case KEY_LEFT:
				if t.offset < len(t.input) {
					t.offset++
					t.Refresh()
				}

			case KEY_RIGHT:
				if t.offset > 0 {
					t.offset--
					t.Refresh()
				}

			case KEY_DEL:
				if len(t.input) > 0 {
					if t.offset < len(t.input) {
						t.input = t.input[0:len(t.input)-t.offset-1] + t.input[len(t.input)-t.offset:]
						t.mainEb.Put(EvtSearchNew, t.input)
						t.Refresh()
					}
				}

			default:
				if b >= 20 && b <= 126 {
					// printable chars
					if t.offset == len(t.input) {
						t.input = string(b) + t.input
					} else {
						t.input = t.input[0:len(t.input)-t.offset] + string(b) + t.input[len(t.input)-t.offset:]
					}
					t.mainEb.Put(EvtSearchNew, t.input)
					t.Refresh()
				}
			}
		}
	}
}

func (t *Terminal) getch(ch chan<- int) {
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

	var buf strings.Builder

	nrows := 0
	ch := t.posY / ChunkSize
	i := t.posY - ch*ChunkSize

	// prints lines in view
	for ; ch < len(t.doc); ch++ {
		chunk := t.doc[ch]

		for ; i < chunk.num; i++ {
			buf.WriteString(getLine(t.doc, t.bounds, ch, i, "\x1b[31;1m"))
			nrows++

			if nrows > t.height-2 {
				break
			}
		}
		i = 0

		if nrows > t.height-2 {
			break
		}
	}

	for j := 0; j < t.height-nrows-1; j++ {
		buf.WriteString("\r\n")
	}

	// prompt
	buf.WriteString("\x1b[31;1m> \x1b[0m\x1b[37;1m")
	if len(t.prompt) > 0 {
		buf.WriteString(t.prompt + " ")
	}

	buf.WriteString(t.input + "\x1b[0m")
	if t.offset > 0 {
		buf.WriteString("\x1b[" + strconv.Itoa(t.offset) + "D")
	}

	t.mu.Unlock()

	fmt.Fprintf(os.Stderr, buf.String())
}

func (t *Terminal) ClearBounds() {
	t.mu.Lock()
	t.bounds = nil
	t.mu.Unlock()
	t.Refresh()
}

func (t *Terminal) UpdateBounds(x [][ChunkSize][][]int) {
	t.mu.Lock()
	t.bounds = x
	t.mu.Unlock()
	t.Refresh()
}

func (t *Terminal) UpdatePrompt(s string) {
	t.mu.Lock()
	t.prompt = s
	t.mu.Unlock()
	t.Refresh()
}

// UpdateChunks saves input snapshot
func (t *Terminal) UpdateChunks(d []*Chunk) {
	t.mu.Lock()

	t.doc = d
	t.numLines = 0
	for _, c := range t.doc {
		t.numLines += c.num
	}

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
