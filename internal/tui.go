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
			buf.Write([]byte(strings.Repeat(" ", n)))

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
// it also pads out the line with spaces until it is b-a length
func getLine(s []byte, bnds [][]int, a, b int, color string) string {
	line, bounds := expandTabs(s, bnds)
	L := len(line)

	if L > b {
		L = b
	}

	if a > L {
		return strings.Repeat(" ", b-a)
	}

	buf := ""
	if bounds == nil || len(bounds) == 0 {
		// all ways the intervals might not exist
		buf = "\x1b[38;5;244m" + line[a:L] + "\x1b[0m"
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
			if I[0] > L {
				I[0] = L
			}
			if I[1] > L {
				I[1] = L
			}

			buf += line[last:I[0]] + color
			buf += line[I[0]:I[1]] + "\x1b[38;5;253m"
			last = I[1]

			if last == L {
				break
			}
		}

		buf += line[last:L] + "\x1b[0m"
	}
	if len(line) < b {
		buf += strings.Repeat(" ", b-len(line))
	}

	return buf
}

func getSplitLine(match []byte, matchIndex [][]int, sub []byte, subIndex [][]int, start, end int, color string) string {
	w := (end - start) / 2
	d := (end-start)%2 == 0

	line := getLine(match, matchIndex, start, start+w, color)
	line += "\u2502"

	if d {
		line += getLine(sub, subIndex, start, start+w-3, color)
	} else {
		line += getLine(sub, subIndex, start, start+w-2, color)
	}
	return line
}

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

	hide   bool
	hideY  int
	width  int
	height int
	posY   int
	posX   int
	offset int // cursor offset

	prompt string
	query  Query

	doc      []*Doc
	files    int
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

Loop:
	for {
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
				break Loop

			case KEY_ENTER:
				t.mainEb.Put(EvtSearchFinal, true)
				break Loop

			case KEY_CTRLJ:
				if !t.hide {
					if t.posY+t.height-1 < t.numLines {
						t.posY++
						t.Refresh()
					}
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
				if !t.hide {
					if t.posY > 0 {
						t.posY--
						t.Refresh()
					}
				}

			case KEY_CTRLF:
				if !t.hide {
					if t.posY+t.height-1 < t.numLines {
						t.posY += t.height
						if t.posY+t.height-1 > t.numLines {
							t.posY = t.numLines - t.height + 1
						}
						t.Refresh()
					}
				}

			case KEY_CTRLB:
				if !t.hide {
					if t.posY > 0 {
						t.posY -= t.height
						if t.posY < 0 {
							t.posY = 0
						}
						t.Refresh()
					}
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

			case KEY_CTRLT:
				t.hide = !t.hide
				t.Refresh()

			case KEY_DEL:
				if t.offset > 0 && len(t.query.input) > 0 {
					t.query.v++
					t.query.input = t.query.input[0:len(t.query.input)-t.offset] + t.query.input[len(t.query.input)-t.offset+1:]
					t.offset--
					t.mainEb.Put(EvtSearchNew, t.query)
					t.RefreshPrompt()
				}

			case KEY_BACKSPACE:
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

func (t *Terminal) getch(ch chan<- int) {
	b := make([]byte, 1)

	done := false
	for !done {
		_, err := syscall.Read(t.fd, b)
		if err != nil {
			// TODO: figure out why the fd becomes bad
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
			} else if b[0] == 51 {
				syscall.Read(t.fd, b)
				if b[0] == 126 {
					ch <- KEY_DEL
				}
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

	prevLines := 0
	d := 0

	posY := t.posY
	if t.hide {
		posY = t.hideY
	}

	// find first relevant doc
	for {
		if d >= len(t.doc) {
			break
		}
		if !t.hide && prevLines+t.doc[d].numLines+t.files >= posY {
			break
		}
		if t.hide && (t.result == nil || d >= len(t.result.matchLines) ||
			prevLines+len(t.result.matchLines[d])+t.files >= posY) {
			break
		}
		if t.hide {
			prevLines += len(t.result.matchLines[d]) + t.files
		} else {
			prevLines += t.doc[d].numLines + t.files
		}
		d++
	}

	if t.files == 1 {
		if prevLines != posY {
			// skip filename line
			prevLines++
		} else {
			// print filename
			buf.WriteString("\x1b[K")
			buf.WriteString(fileColor)
			buf.WriteString("******  " + t.doc[d].filename + "  ******\x1b[0m\r\n")
			nrows++
		}
	}

	// XXX: probably better way to do this
	if !t.hide {
		ch := (posY - prevLines) / ChunkSize
		i := posY - prevLines - ch*ChunkSize

		// prints lines in view
		// d, ch, i have been previously set up
	Loop:
		for ; d < len(t.doc); d++ {
			doc := t.doc[d]
			for ; ch < len(doc.chunks); ch++ {
				chunk := doc.chunks[ch]

				for ; i < chunk.num; i++ {
					buf.WriteString("\x1b[K")

					line := ""
					if t.result != nil && len(t.result.matchIndex) > d && len(t.result.matchIndex[d].index) > ch {
						if t.result.output == nil {
							line = getLine(*chunk.lines[i], t.result.matchIndex[d].index[ch][i], t.posX, t.posX+t.width, matchColor)
						} else {
							line = getSplitLine(*chunk.lines[i], t.result.matchIndex[d].index[ch][i], *t.result.output[d][ch*ChunkSize+i],
								t.result.subIndex[d].index[ch][i], t.posX, t.posX+t.width, matchColor)
						}
					} else {
						// there is no bounds for this
						if t.result == nil || t.result.output == nil {
							line = getLine(*chunk.lines[i], nil, t.posX, t.posX+t.width, matchColor)
						} else {
							line = getLine(*chunk.lines[i], nil, t.posX, t.posX+t.width, matchColor)
						}
					}
					buf.WriteString(line)
					buf.WriteString("\r\n")
					nrows++

					if nrows > t.height-3 {
						break Loop
					}
				}
				i = 0

				if nrows > t.height-3 {
					break Loop
				}
			}
			ch = 0

			if d != len(t.doc)-1 {
				buf.WriteString("\x1b[K")
				buf.WriteString(fileColor)
				buf.WriteString("******  " + t.doc[d+1].filename + "  ******\x1b[0m\r\n")
				nrows++
				if nrows > t.height-3 {
					break Loop
				}
			}
		}
	} else {
		i := posY - prevLines

		// prints matches in view
		// d, i have been previously set up
	Loop2:
		for ; d < len(t.doc); d++ {
			if t.result != nil && d < len(t.result.matchLines) {
				matches := t.result.matchLines[d]
				for ; i < len(matches); i++ {
					ch := matches[i] / ChunkSize
					j := matches[i] % ChunkSize
					chunk := t.doc[d].chunks[ch]

					buf.WriteString("\x1b[K")

					line := getLine(*chunk.lines[j], t.result.matchIndex[d].index[ch][j], t.posX, t.posX+t.width, matchColor)
					buf.WriteString(line)
					buf.WriteString("\r\n")
					nrows++

					if nrows > t.height-3 {
						break Loop2
					}
				}
				i = 0
			}

			if nrows > t.height-3 {
				break Loop2
			}

			if d != len(t.doc)-1 {
				buf.WriteString("\x1b[K")
				buf.WriteString(fileColor)
				buf.WriteString("******  " + t.doc[d+1].filename + "  ******\x1b[0m\r\n")
				nrows++
				if nrows > t.height-3 {
					break Loop2
				}
			}
		}
	}

	for j := nrows; j < t.height-1; j++ {
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
	buf := "\x1b[G\x1b[F"

	if t.doc == nil {
		buf += "\r\n"
	} else {
		matchCount := t.numLines
		if t.result != nil && t.result.matchLines != nil {
			matchCount = 0
			for _, x := range t.result.matchLines {
				matchCount += len(x)
			}
		}
		buf += fmt.Sprintf("\x1b[37;1m%d\x1b[31;1m/\x1b[37;1m%d\x1b[0m\r\n", matchCount, t.numLines)
	}

	buf += "\x1b[31;1m> \x1b[0m\x1b[37;1m"

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
	if !t.displayed {
		n := len(t.result.matchIndex) - 1
		numLines := 0

		for d := 0; d < len(t.result.matchIndex); d++ {
			if len(t.result.matchIndex[d].index) == len(t.doc[d].chunks) {
				numLines += t.doc[d].numLines + t.files
			} else {
				numLines += t.files + len(t.result.matchIndex[d].index)*ChunkSize
			}
		}

		if (len(t.result.matchIndex) == len(t.doc) && len(t.result.matchIndex[n].index) == len(t.doc[n].chunks)) ||
			numLines > t.posY+t.height {
			t.displayed = true
			refresh = true
		}
	}

	t.mu.Unlock()
	if refresh || t.hide {
		t.Refresh()
	}
	t.RefreshPrompt()
}

func (t *Terminal) UpdatePrompt(s string) {
	t.mu.Lock()
	t.prompt = s
	t.mu.Unlock()
	t.Refresh()
}

// UpdateChunks saves input snapshot
func (t *Terminal) UpdateChunks(docs []*Doc, final bool) {
	t.mu.Lock()

	t.doc = docs

	oldLines := t.numLines
	t.numLines = 0
	for _, d := range t.doc {
		t.numLines += d.numLines
	}

	// update the view if new relevant chunks came in
	refresh := t.doc == nil || (oldLines < t.posX+t.height && t.numLines >= t.posX+t.height)

	t.mu.Unlock()

	if refresh {
		t.Refresh()
	} else {
		t.RefreshPrompt()
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
func (t *Terminal) Init(files int) {
	t.files = files

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
