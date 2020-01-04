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
	bounds     []*Bounds
	matchLines [][]int
	v          int
	replace    bool
}

type Bounds struct {
	index [][ChunkSize][][]int
}

type Machine struct {
	prog     *regexp.Regexp
	mainEb   *EventBox
	localEb  *EventBox
	doneChan chan<- *Output
	mu       sync.Mutex

	sleep        bool
	finalDoc     bool
	finalMachine bool

	doc       []*Doc
	currDoc   int // current doc processing
	currChunk int // current chunk processing

	replace    bool
	res        []*Bounds
	matches    [][]*[]byte
	matchLines [][]int
	v          int
}

func NewMachine(eb *EventBox, ch chan<- *Output) *Machine {
	return &Machine{
		mainEb:     eb,
		localEb:    NewEventBox(),
		mu:         sync.Mutex{},
		doneChan:   ch,
		res:        make([]*Bounds, 0),
		matches:    make([][]*[]byte, 0),
		matchLines: make([][]int, 0),
	}
}

// Loop keep applying regexp to m.doc starting at m.curr
func (m *Machine) Loop() {
	done := false
	for !done {
		// m.currDoc and m.currChunk has been initially set
		// use to loop over m.doc
		i := 0
		for {
			m.mu.Lock()

			if m.currDoc < len(m.doc)-1 && m.currChunk == len(m.doc[m.currDoc].chunks) {
				// hit end of current doc but them is another
				m.currDoc++
				m.currChunk = 0
			}

			// only way to exit the inner loop
			if m.doc == nil || m.prog == nil ||
				(m.currDoc == len(m.doc)-1 && m.currChunk == len(m.doc[m.currDoc].chunks)) {
				break
			}

			doc := m.doc[m.currDoc]
			ch := doc.chunks[m.currChunk]

			// allocate new list per doc
			for m.currDoc >= len(m.res) {
				m.res = append(m.res, &Bounds{index: make([][ChunkSize][][]int, 0)})
				m.matches = append(m.matches, make([]*[]byte, 0))
				m.matchLines = append(m.matchLines, make([]int, 0))
			}

			// allocate new bound per chunk
			for m.currChunk >= len(m.res[m.currDoc].index) {
				m.res[m.currDoc].index = append(m.res[m.currDoc].index, [ChunkSize][][]int{})
			}

			// record regexp output
			for i, s := range ch.lines {
				if s != nil {
					m.res[m.currDoc].index[m.currChunk][i] = m.prog.FindAllIndex(*s, 1)
					if len(m.res[m.currDoc].index[m.currChunk][i]) > 0 {
						m.matches[m.currDoc] = append(m.matches[m.currDoc], ch.lines[i])
						m.matchLines[m.currDoc] = append(m.matchLines[m.currDoc], m.currChunk*ChunkSize+i)
					}
				}
			}

			m.currChunk++
			i++
			if i%50 == 0 || m.currChunk == len(doc.chunks) {
				m.mainEb.Put(EvtSearchProgress, m.Snapshot())
			}
			m.mu.Unlock()
		}
		m.sleep = true
		m.mu.Unlock()

		// finished current doc and wait for signal to continue/finish
		m.localEb.Wait(func(events *Events) {
			m.mu.Lock()
			m.localEb.Clear()

			if m.finalDoc && m.finalMachine {
				// we am done if all things am final and we am at the end
				// we check in this order to avoid slice errors
				done = m.currDoc == len(m.doc)-1 && m.currChunk == len(m.doc[m.currDoc].chunks)
			}

			m.mu.Unlock()
		})
	}

	// send results
	m.doneChan <- &Output{
		output:  m.matches,
		replace: m.replace,
	}
}

func (m *Machine) UpdateDoc(d []*Doc, final bool) {
	m.mu.Lock()
	m.doc = d
	m.finalDoc = m.finalDoc || final

	// wake up if asleep
	if m.sleep {
		m.sleep = false
		m.localEb.Put(EvtReadNew, false)
	}
	m.mu.Unlock()
}

// UpdateMachine updates the regexp if possible
func (m *Machine) UpdateMachine(q Query) {
	r, err := regexp.Compile(q.input)

	if len(q.input) == 0 || err != nil {
		// not proper regexp
		m.mu.Lock()
		m.prog = nil
		m.currDoc = 0
		m.currChunk = 0
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	if m.v < q.v {
		// only update if newer query
		m.v = q.v
		for i := range m.matches {
			m.matches[i] = make([]*[]byte, 0)
			m.matchLines[i] = make([]int, 0)
		}
		m.prog = r
		m.currDoc = 0
		m.currChunk = 0

		if m.sleep {
			m.sleep = false
			m.localEb.Put(EvtFinish, false)
		}
	}
	m.mu.Unlock()
}

func (m *Machine) Finish() {
	m.mu.Lock()
	m.finalMachine = true
	m.localEb.Put(EvtFinish, false)
	m.mu.Unlock()
}

// Snapshot returns a copy of the current outputs of the regexp program
// It is called inside a critical section
func (m *Machine) Snapshot() *Result {
	res := Result{
		bounds:     make([]*Bounds, 0),
		matchLines: make([][]int, len(m.matchLines)),
		v:          m.v,
		replace:    m.replace,
	}

	for i, l := range m.matchLines {
		res.matchLines[i] = make([]int, len(l))
		copy(res.matchLines[i], l)
	}

	for i, r := range m.res {
		b := Bounds{}

		if i < m.currDoc {
			// a previous doc so copy everything
			b.index = make([][ChunkSize][][]int, len(m.doc[i].chunks))
			copy(b.index, r.index)
		} else if i == m.currDoc {
			b.index = make([][ChunkSize][][]int, m.currChunk)
			copy(b.index, r.index[:m.currChunk])
		} else {
			break
		}

		res.bounds = append(res.bounds, &b)
	}

	return &res
}
