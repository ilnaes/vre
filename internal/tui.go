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

// expandTabs expands all tabs up to TABSTOP spaces and modifies boundaries to accomodate
func expandTabs(s []byte, bounds [][]int) (string, [][]int) {
	if len(s) == 0 {
		return "", make([][]int, 0)
	}

	last := 0
	var buf strings.Builder
	pad := 0   // extra spaces from tabs
	curr := -1 // current boundary point

	var nbounds [][]int
	if bounds == nil || len(bounds) == 0 {
		nbounds = nil
	} else {
		nbounds = make([][]int, len(bounds))

		// mark first number of an interval greater than 0
		if bounds[0][0] == 0 {
			curr = 1
			nbounds[0] = []int{0, 0}
		} else {
			curr = 0
		}
	}

	for j, c := range s {
		if c == '\t' {
			buf.Write(s[last:j])

			n := TABSTOP - buf.Len()%TABSTOP
			for k := 0; k < n; k++ {
				buf.WriteRune(' ')
			}

			pad += n - 1

			last = j + 1
		}

		// update bounds if any passed
		if curr != -1 {
			for ; curr < 2*len(bounds) && bounds[curr/2][curr%2] <= j; curr++ {
				if curr%2 == 0 {
					nbounds[curr/2] = []int{bounds[curr/2][0] + pad, 0}
				} else {
					nbounds[curr/2][1] = bounds[curr/2][1] + pad
				}
			}
		}
	}

	if curr != -1 {
		for ; curr < 2*len(bounds) && bounds[curr/2][curr%2] <= len(s); curr++ {
			if curr%2 == 0 {
				nbounds[curr/2] = []int{bounds[curr/2][0] + pad, 0}
			} else {
				nbounds[curr/2][1] = bounds[curr/2][1] + pad
			}
		}
	}

	buf.Write(s[last:])

	return buf.String(), nbounds
}

// getLine will expand the tabs and color the text between intervals in bnds
func getLine(s []byte, bnds [][]int, a, b int, color string) string {
	line, bounds := expandTabs(s, bnds)

	if a > len(line) {
		return "\r\n"
	}
	if b > len(line) {
		b = len(line)
	}

	buf := ""
	if bounds == nil || len(bounds) == 0 {
		// all ways the intervals might not exist
		buf = "\x1b[38;5;244m" + line[a:b] + "\x1b[0m"
	} else {
		last := a
		buf = "\x1b[1m\x1b[38;5;253m"

		for _, I := range bounds {
			if I[1] <= last {
				// I is before [a,b]
				continue
			}

			if I[0] < a {
				I[0] = a
			}
			if I[0] > b {
				I[0] = b
			}
			if I[1] > b {
				I[1] = b
			}

			buf += line[last:I[0]] + color
			buf += line[I[0]:I[1]] + "\x1b[38;5;253m"
			last = I[1]

			if last == b {
				break
			}
		}

		buf += line[last:b] + "\x1b[0m"
	}

	return buf + "\r\n"
}

const console string = "/dev/tty"

type Query struct {
	input string
	v     int
}

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

	prompt string
	query  Query

	doc      []*Chunk
	numLines int

	result    *Result
	numRes    int
	displayed bool
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
			case KEY_CTRLC, KEY_CTRLD:
				// Ctrl-C/D quit
				t.Close()
				t.mainEb.Put(EvtQuit, nil)
				done = true

			case KEY_ENTER:
				t.mainEb.Put(EvtSearchFinal, true)
				done = true

			case KEY_CTRLJ:
				if t.posY+t.height-1 < t.numLines {
					t.posY++
					t.Refresh()
				}

			case KEY_CTRLH:
				if t.posX > 0 {
					t.posX--
					t.Refresh()
				}

			case KEY_CTRLL:
				t.posX++
				t.Refresh()

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
				if t.offset < len(t.query.input) {
					t.offset++
					t.RefreshPrompt()
				}

			case KEY_RIGHT:
				if t.offset > 0 {
					t.offset--
					t.RefreshPrompt()
				}

			case KEY_DEL:
				if len(t.query.input) > 0 {
					if t.offset < len(t.query.input) {
						t.query.v++
						t.query.input = t.query.input[0:len(t.query.input)-t.offset-1] + t.query.input[len(t.query.input)-t.offset:]
						t.mainEb.Put(EvtSearchNew, t.query)
						t.RefreshPrompt()
					}
				}

			default:
				if b >= 20 && b <= 126 {
					t.query.v++
					// printable chars
					if t.offset == len(t.query.input) {
						t.query.input = string(b) + t.query.input
					} else {
						t.query.input = t.query.input[0:len(t.query.input)-t.offset] + string(b) + t.query.input[len(t.query.input)-t.offset:]
					}
					t.mainEb.Put(EvtSearchNew, t.query)
					t.RefreshPrompt()
				}
			}
		}
	}
}

var i int = 1

func (t *Terminal) getch(ch chan<- int) {
	b := make([]byte, 1)

	done := false
	for !done {
		_, err := syscall.Read(t.fd, b)
		if err != nil {
			// TODO: figure out why the fd becomes bad
			// t.prompt = fmt.Sprintf("%d *", i)
			i++
			t.openConsole()
			_, err = syscall.Read(t.fd, b)
			if err != nil {
				panic(err)
			}
		}

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

// Refresh prints contents
func (t *Terminal) Refresh() {
	t.mu.Lock()

	var buf strings.Builder
	buf.WriteString("\x1b[?25l\x1b[H")

	nrows := 0
	ch := t.posY / ChunkSize
	i := t.posY - ch*ChunkSize

	// prints lines in view
	for ; ch < len(t.doc); ch++ {
		chunk := t.doc[ch]

		for ; i < chunk.num; i++ {
			buf.WriteString("\x1b[K")

			line := ""
			if t.result == nil || len(t.result.index) < ch+1 {
				line = getLine(*t.doc[ch].lines[i], nil, t.posX, t.posX+t.width, "\x1b[31;1m")
			} else {
				line = getLine(*t.doc[ch].lines[i], t.result.index[ch][i], t.posX, t.posX+t.width, "\x1b[31;1m")
			}
			buf.WriteString(line)
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
		buf.WriteString("\x1b[K\r\n")
	}
	buf.WriteString("\x1b[?25h")
	fmt.Fprint(os.Stderr, buf.String())

	t.mu.Unlock()

	t.RefreshPrompt()
}

// RefreshPrompt refreshes just the prompt line
func (t *Terminal) RefreshPrompt() {
	t.mu.Lock()
	buf := "\x1b[G\x1b[31;1m> \x1b[0m\x1b[37;1m"

	if len(t.prompt) > 0 {
		buf += t.prompt + " "
	}

	buf += t.query.input + "\x1b[K\x1b[0m"
	if t.offset > 0 {
		// set cursor
		buf += "\x1b[" + strconv.Itoa(t.offset) + "D"
	}

	fmt.Fprint(os.Stderr, buf)
	t.mu.Unlock()
}

func (t *Terminal) ClearBounds() {
	t.mu.Lock()
	t.result = nil
	t.mu.Unlock()
	t.Refresh()
}

func (t *Terminal) UpdateBounds(x *Result) {
	t.mu.Lock()

	if t.result == nil || x.v > t.result.v {
		t.displayed = false
	}
	t.result = x

	refresh := false

	// refresh the display if got a new version of result and got enough chunks
	if (len(t.result.index) == len(t.doc) || len(t.result.index) > (t.posY+t.height)/ChunkSize) && !t.displayed {
		t.displayed = true
		refresh = true
	}

	t.mu.Unlock()
	if refresh {
		t.Refresh()
	}
}

func (t *Terminal) UpdatePrompt(s string) {
	t.mu.Lock()
	t.prompt = s
	t.mu.Unlock()
	t.Refresh()
}

// UpdateChunks saves input snapshot
func (t *Terminal) UpdateChunks(d []*Chunk, final bool) {
	t.mu.Lock()

	refresh := t.doc == nil || (len(t.doc) < (t.posX+t.height)/ChunkSize && len(d) >= (t.posX+t.height)/ChunkSize)

	t.doc = d

	t.numLines = 0
	for _, c := range t.doc {
		t.numLines += c.num
	}

	t.mu.Unlock()

	if refresh {
		t.Refresh()
	}
}

// GetSize updates the size of the terminal
func (t *Terminal) GetSize() {
	w, h, err := terminal.GetSize(t.fd)
	if err != nil {
		t.Close()
		log.Fatal("Could not get terminal size")
	}
	t.width = w
	t.height = h
}

// Init saves current state of terminal, sets up raw mode and alternate screen buffer
func (t *Terminal) Init() {
	t.openConsole()

	origState, err := terminal.GetState(t.fd)
	if err != nil {
		log.Fatal("Could not get terminal state")
	}

	t.origState = origState
	terminal.MakeRaw(t.fd)
	t.GetSize()

	fmt.Fprint(os.Stderr, "\x1b[?1049h")
}

// Close closes alternate screen buffer and restores original terminal state
func (t *Terminal) Close() {
	fmt.Fprint(os.Stderr, "\x1b[?1049l")
	err := terminal.Restore(t.fd, t.origState)
	if err != nil {
		t.openConsole()
		terminal.Restore(t.fd, t.origState)
	}
}

func (t *Terminal) openConsole() {
	tty, err := os.OpenFile(console, syscall.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	t.fd = int(tty.Fd())
}
