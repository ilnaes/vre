package vre

import (
	// "fmt"
	"regexp"
	"sync"
)

type Re struct {
	prog    *regexp.Regexp
	mainEb  *EventBox
	localEb *EventBox
	mu      sync.Mutex
	sleep   bool

	doc  []*Chunk
	res  [][ChunkSize][][]int
	num  int // size of res
	curr int // current chunk processing
}

func NewRe(eb *EventBox) *Re {
	return &Re{
		mainEb:  eb,
		localEb: NewEventBox(),
		mu:      sync.Mutex{},
		res:     make([][ChunkSize][][]int, 0),
	}
}

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

			// allocate new result chunk
			for ; re.num <= re.curr; re.num++ {
				re.res = append(re.res, [ChunkSize][][]int{})
			}

			for i, s := range ch.lines {
				re.res[re.curr][i] = re.prog.FindAllStringIndex(s, 1)
			}

			re.mainEb.Put(EvtSearchProgress, re.res)
			re.curr++
			re.mu.Unlock()
		}
		re.sleep = true
		re.mu.Unlock()

		// finished current doc and wait for signal to continue/finish
		re.localEb.Wait(func(events *Events) {
			for e, v := range *events {
				if e == EvtFinish {
					if b := v.(bool); b {
						// finish up
						done = true
					}
				}
			}
		})
	}
}

// Finish either localEbes the loop or restarts it
func (re *Re) Finish(b bool) {
	re.localEb.Put(EvtFinish, b)
}

func (re *Re) UpdateDoc(d []*Chunk) {
	re.mu.Lock()
	re.doc = d

	if re.sleep {
		re.sleep = false
		re.localEb.Put(EvtFinish, false)
	}
	re.mu.Unlock()
}

// UpdateRe updates the regexp if possible
func (re *Re) UpdateRe(s string) {
	if len(s) == 0 {
		re.mu.Lock()
		re.prog = nil
		re.curr = 0
		re.mu.Unlock()
		return
	}
	r, err := regexp.Compile(s)
	if err != nil {
		return
	}

	re.mu.Lock()
	re.prog = r
	re.curr = 0

	if re.sleep {
		re.sleep = false
		re.localEb.Put(EvtFinish, false)
	}
	re.mu.Unlock()
}
