package vre

import (
	"regexp"
	"sync"
)

// Output.output is what gets printed at the end
type Output struct {
	replace bool
	output  [][]*[]byte
}

// Result goes to tui for display
type Result struct {
	bounds  []*Bounds
	output  [][]*[]byte
	v       int
	replace bool
}

type Bounds struct {
	index [][ChunkSize][][]int
}

type Re struct {
	prog     *regexp.Regexp
	mainEb   *EventBox
	localEb  *EventBox
	doneChan chan<- *Output
	mu       sync.Mutex

	sleep    bool
	finalDoc bool
	finalRe  bool

	doc       []*Doc
	currDoc   int // current doc processing
	currChunk int // current chunk processing

	replace bool
	res     []*Bounds
	matches [][]*[]byte
	v       int
}

func NewRe(eb *EventBox, ch chan<- *Output) *Re {
	return &Re{
		mainEb:   eb,
		localEb:  NewEventBox(),
		mu:       sync.Mutex{},
		doneChan: ch,
		res:      make([]*Bounds, 0),
	}
}

// Loop keep applying regexp to re.doc starting at re.curr
func (re *Re) Loop() {
	done := false
	for !done {
		// re.currDoc and re.currChunk has been initially set
		// use to loop over re.doc
		i := 0
		for {
			re.mu.Lock()

			if re.currDoc < len(re.doc)-1 && re.currChunk == len(re.doc[re.currDoc].chunks) {
				// hit end of current doc but there is another
				re.currDoc++
				re.currChunk = 0
			}

			// only way to exit the inner loop
			if re.doc == nil || re.prog == nil ||
				(re.currDoc == len(re.doc)-1 && re.currChunk == len(re.doc[re.currDoc].chunks)) {
				break
			}

			doc := re.doc[re.currDoc]
			ch := doc.chunks[re.currChunk]

			// allocate new list per doc
			for re.currDoc >= len(re.res) {
				re.res = append(re.res, &Bounds{index: make([][ChunkSize][][]int, 0)})
				re.matches = append(re.matches, make([]*[]byte, 0))
			}

			// allocate new bound per chunk
			for re.currChunk >= len(re.res[re.currDoc].index) {
				re.res[re.currDoc].index = append(re.res[re.currDoc].index, [ChunkSize][][]int{})
			}

			// record regexp output
			for i, s := range ch.lines {
				if s != nil {
					re.res[re.currDoc].index[re.currChunk][i] = re.prog.FindAllIndex(*s, 1)
					if len(re.res[re.currDoc].index[re.currChunk][i]) > 0 {
						re.matches[re.currDoc] = append(re.matches[re.currDoc], ch.lines[i])
						// re.docNums = append(re.docNums, re.currDoc)
					}
				}
			}

			re.currChunk++
			i++
			if i%50 == 0 || re.currChunk == len(doc.chunks) {
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

			if re.finalDoc && re.finalRe {
				// we are done if all things are final and we are at the end
				// we check in this order to avoid slice errors
				done = re.currDoc == len(re.doc)-1 && re.currChunk == len(re.doc[re.currDoc].chunks)
			}

			re.mu.Unlock()
		})
	}

	// send results
	re.doneChan <- &Output{
		output: re.matches,
		// docs:    re.docNums,
		replace: re.replace,
	}
}

func (re *Re) UpdateDoc(d []*Doc, final bool) {
	re.mu.Lock()
	re.doc = d
	re.finalDoc = re.finalDoc || final

	// wake up if asleep
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
		// re.docNums = nil
		re.prog = nil
		re.currDoc = 0
		re.currChunk = 0
		re.mu.Unlock()
		return
	}

	re.mu.Lock()
	if re.v < q.v {
		// only update if newer query
		re.v = q.v
		for i := range re.matches {
			re.matches[i] = make([]*[]byte, 0)
		}
		// re.docNums = make([]int, 0)
		re.prog = r
		re.currDoc = 0
		re.currChunk = 0

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
// It is called inside a critical section
func (re *Re) Snapshot() *Result {
	res := Result{
		bounds:  make([]*Bounds, 0),
		output:  make([][]*[]byte, len(re.matches)),
		v:       re.v,
		replace: re.replace,
	}

	for i, l := range re.matches {
		res.output[i] = make([]*[]byte, len(l))
		copy(res.output[i], l)
	}

	for i, r := range re.res {
		b := Bounds{}

		if i < re.currDoc {
			// a previous doc so copy everything
			b.index = make([][ChunkSize][][]int, len(re.doc[i].chunks))
			copy(b.index, r.index)
		} else if i == re.currDoc {
			b.index = make([][ChunkSize][][]int, re.currChunk)
			copy(b.index, r.index[:re.currChunk])
		} else {
			break
		}

		res.bounds = append(res.bounds, &b)
	}

	return &res
}
