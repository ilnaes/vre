package vre

import (
	"bufio"
	"os"
	"sync"
	"time"
)

// ChunkSize is the number of lines in a chunk (XXX: this should be changed to be more reasonable later)
const ChunkSize int = 2

type Chunk struct {
	lines [ChunkSize]string
	num   int
}

// Doc acts as the model
type Doc struct {
	mu     sync.Mutex
	mainEb *EventBox

	chunks    []*Chunk
	numChunks int
}

func NewDoc(eb *EventBox) *Doc {
	return &Doc{
		chunks: make([]*Chunk, 0),
		mainEb: eb,
		mu:     sync.Mutex{},
	}
}

// Snapshot returns the current items in the document
func (d *Doc) Snapshot() (*[]*Chunk, int) {
	d.mu.Lock()

	res := make([]*Chunk, d.numChunks)
	copy(res, d.chunks)

	d.mu.Unlock()

	return &res, d.numChunks
}

// Read reads the file in ChunkSize chunks and appends to Doc
func (d *Doc) Read(io *os.File) {
	scanner := bufio.NewScanner(io)
	chunk := &Chunk{}

	for scanner.Scan() {
		chunk.lines[chunk.num] = scanner.Text()
		chunk.num++

		if chunk.num == ChunkSize {
			d.mu.Lock()
			d.chunks = append(d.chunks, chunk)
			d.numChunks++
			d.mu.Unlock()

			chunk = &Chunk{}
			d.mainEb.Put(EvtReadNew, nil)
			// XXX: used to simulate delay
			time.Sleep(500 * time.Millisecond)
		}
	}

	if chunk.num != 0 {
		d.mu.Lock()
		d.chunks = append(d.chunks, chunk)
		d.numChunks++
		d.mu.Unlock()
	}

	d.mainEb.Put(EvtReadDone, nil)
}
