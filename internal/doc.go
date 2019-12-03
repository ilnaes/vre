package vre

import (
	"os"
	"sync"
)

// ChunkSize is the number of lines in a chunk
const ChunkSize int = 100

type Chunk struct {
	lines [ChunkSize]string
	num   int
}

// Doc acts as the model
type Doc struct {
	chunks    []*Chunk
	mu        sync.Mutex
	numChunks int
}

func (d *Doc) Read(io *os.File) {
	d.mu.Lock()
	d.mu.Unlock()
}
