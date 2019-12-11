package vre

import (
	"regexp"
	"sync"
)

type Result struct {
	v       int
	index   [][ChunkSize][][]int
	matches []*[]byte
}

type Re struct {
	prog     *regexp.Regexp
	mainEb   *EventBox
	localEb  *EventBox
	doneChan chan<- []*[]byte
	mu       sync.Mutex

	sleep    bool
	finalDoc bool
	finalRe  bool

	doc  []*Chunk
	curr int // current chunk processing

	res Result
}

func NewRe(eb *EventBox, ch chan<- []*[]byte) *Re {
	return &Re{
		mainEb:   eb,
		localEb:  NewEventBox(),
		mu:       sync.Mutex{},
		doneChan: ch,
		res: Result{
			index:   make([][ChunkSize][][]int, 0),
			matches: make([]*[]byte, 0),
		},
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
			for re.curr >= len(re.res.index) {
				re.res.index = append(re.res.index, [ChunkSize][][]int{})
			}

			// record regexp output
			for i, s := range ch.lines {
				if s != nil {
					re.res.index[re.curr][i] = re.prog.FindAllIndex(*s, 1)
					if len(re.res.index[re.curr][i]) > 0 {
						re.res.matches = append(re.res.matches, ch.lines[i])
					}
				}
			}

			re.curr++
			if re.curr%10 == 0 || re.curr == len(re.doc) {
				re.mainEb.Put(EvtSearchProgress, re.Snapshot())
			}
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
	re.doneChan <- re.res.matches
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
func (re *Re) UpdateRe(q Query) {
	r, err := regexp.Compile(q.input)

	if len(q.input) == 0 || err != nil {
		// not proper regexp
		re.mu.Lock()
		re.res.matches = nil
		re.prog = nil
		re.curr = 0
		re.mu.Unlock()
		return
	}

	re.mu.Lock()
	if re.res.v < q.v {
		// only update if newer query
		re.res.v++
		re.res.matches = make([]*[]byte, 0)
		re.prog = r
		re.curr = 0

		if re.sleep {
			re.sleep = false
			re.localEb.Put(EvtFinish, false)
		}
	}
	re.mu.Unlock()
}

func (re *Re) Finish() {
	re.mu.Lock()
	re.finalRe = true
	re.localEb.Put(EvtFinish, false)
	re.mu.Unlock()
}

// Snapshot returns a copy of the current outputs of the regexp program
// It is called already inside a critical section
func (re *Re) Snapshot() *Result {
	index := make([][ChunkSize][][]int, re.curr)
	copy(index, re.res.index[:re.curr])
	res := Result{
		v:     re.res.v,
		index: index,
	}

	return &res
}
