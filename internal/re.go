package vre

import (
	"regexp"
	"sync"
)

type Re struct {
	prog     *regexp.Regexp
	mainEb   *EventBox
	localEb  *EventBox
	doneChan chan<- []*string
	mu       sync.Mutex

	sleep    bool
	finalDoc bool
	finalRe  bool

	doc     []*Chunk
	bounds  [][ChunkSize][][]int
	matches []*string
	curr    int // current chunk processing
}

func NewRe(eb *EventBox, ch chan<- []*string) *Re {
	return &Re{
		mainEb:   eb,
		localEb:  NewEventBox(),
		mu:       sync.Mutex{},
		bounds:   make([][ChunkSize][][]int, 0),
		doneChan: ch,
	}
}

// Loop keep applying regexp to re.doc starting at re.curr
func (re *Re) Loop() {
	done := false
	for !done {
		// re.curr has been initially set
		// use to loop over re.doc
		for {
			re.mu.Lock()

			// only way to exit the inner loop
			if re.doc == nil || re.prog == nil || re.curr >= len(re.doc) {
				break
			}

			ch := re.doc[re.curr]

			// allocate new bound per chunk
			for re.curr >= len(re.bounds) {
				re.bounds = append(re.bounds, [ChunkSize][][]int{})
			}

			// record regexp output
			for i, s := range ch.lines {
				re.bounds[re.curr][i] = re.prog.FindAllStringIndex(s, 1)
				if len(re.bounds[re.curr][i]) > 0 {
					re.matches = append(re.matches, &ch.lines[i])
				}
			}

			re.mainEb.Put(EvtSearchProgress, re.bounds)
			re.curr++
			re.mu.Unlock()
		}
		re.sleep = true
		re.mu.Unlock()

		// finished current doc and wait for signal to continue/finish
		re.localEb.Wait(func(events *Events) {
			re.mu.Lock()
			re.localEb.Clear()

			// we are done if all things are final and we are at the end
			done = re.finalDoc && re.finalRe && len(re.doc) == re.curr
			re.mu.Unlock()
		})
	}

	// send results
	re.doneChan <- re.matches
}

func (re *Re) UpdateDoc(d []*Chunk, final bool) {
	re.mu.Lock()
	re.doc = d
	re.finalDoc = re.finalDoc || final

	if re.sleep {
		re.sleep = false
		re.localEb.Put(EvtReadNew, false)
	}
	re.mu.Unlock()
}

// UpdateRe updates the regexp if possible
func (re *Re) UpdateRe(s string) {
	r, err := regexp.Compile(s)

	if len(s) == 0 || err != nil {
		// not proper regexp
		re.mu.Lock()
		re.prog = nil
		re.curr = 0
		re.mu.Unlock()
		return
	}

	re.mu.Lock()
	re.prog = r
	re.curr = 0
	re.matches = make([]*string, 0)

	if re.sleep {
		re.sleep = false
		re.localEb.Put(EvtFinish, false)
	}
	re.mu.Unlock()
}

func (re *Re) Finish() {
	re.mu.Lock()
	re.finalRe = true
	re.localEb.Put(EvtFinish, false)
	re.mu.Unlock()
}
