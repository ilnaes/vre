package vre

import (
	"bufio"
	"os"
	"sync"
)

// ChunkSize is the number of lines in a chunk (XXX: this should be changed to be more reasonable later)
const ChunkSize int = 2

type Chunk struct {
	lines [ChunkSize]string
	num   int
}

// Doc acts as the model
type Doc struct {
	chunks    []*Chunk
	mu        sync.Mutex
	numChunks int
	mainEb    *EventBox
}

func NewDoc(eb *EventBox) *Doc {
	return &Doc{
		chunks: make([]*Chunk, 0),
		mainEb: eb,
		mu:     sync.Mutex{},
	}
}

// Read reads the file in ChunkSize chunks and appends to Doc
func (d *Doc) Read(io *os.File) {
	scanner := bufio.NewScanner(io)
	chunk := Chunk{}

	for scanner.Scan() {
		chunk.lines[chunk.num] = scanner.Text()
		chunk.num++

		if chunk.num == ChunkSize {
			d.mu.Lock()
			d.chunks = append(d.chunks, &chunk)
			d.numChunks++
			d.mu.Unlock()

			chunk = Chunk{}
			d.mainEb.Put(EvtReadNew, nil)
		}
	}

	if chunk.num != 0 {
		d.mu.Lock()
		d.chunks = append(d.chunks, &chunk)
		d.numChunks++
		d.mu.Unlock()
	}

	d.mainEb.Put(EvtReadDone, nil)
}
