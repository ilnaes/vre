package vre

import (
	"bufio"
	"os"
	"sync"
	"time"
)

const ChunkSize int = 10

type Chunk struct {
	lines [ChunkSize]string
	num   int
}

// Reader acts as the model
type Reader struct {
	mu     sync.Mutex
	mainEb *EventBox
	doc    []*Chunk
}

func NewReader(eb *EventBox) *Reader {
	return &Reader{
		mainEb: eb,
		mu:     sync.Mutex{},
		doc:    make([]*Chunk, 0),
	}
}

// Read reads the file in ChunkSize chunks and appends to Reader
func (r *Reader) Read(io *os.File) {
	scanner := bufio.NewScanner(io)
	chunk := &Chunk{}

	for scanner.Scan() {
		chunk.lines[chunk.num] = scanner.Text()
		chunk.num++

		if chunk.num == ChunkSize {
			r.mu.Lock()
			r.doc = append(r.doc, chunk)
			r.mu.Unlock()

			chunk = &Chunk{}
			r.mainEb.Put(EvtReadNew, nil)
			// XXX: used to simulate delay
			time.Sleep(250 * time.Millisecond)
		}
	}

	if chunk.num != 0 {
		r.mu.Lock()
		r.doc = append(r.doc, chunk)
		r.mu.Unlock()
	}

	r.mainEb.Put(EvtReadDone, nil)
}

// Snapshot returns the current items in the document
func (r *Reader) Snapshot() *[]*Chunk {
	r.mu.Lock()
	res := make([]*Chunk, len(r.doc))
	copy(res, r.doc)
	r.mu.Unlock()

	return &res
}
