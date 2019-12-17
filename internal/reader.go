package vre

import (
	"bufio"
	"os"
	"sync"
)

const ChunkSize int = 1000

type Doc struct {
	chunks   []*Chunk
	filename string
}

type Chunk struct {
	lines [ChunkSize]*[]byte
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

// ReadFiles reads the files given
func (r *Reader) ReadFiles(fs []string) {
	for _, fname := range fs {
		f, err := os.Open(fname)
		if err != nil {
			r.mainEb.Put(EvtReadError, fname)
		}

		r.ReadFile(f)
	}
}

// ReadStream reads the file in ChunkSize chunks and appends to Reader
func (r *Reader) ReadFile(io *os.File) {
	reader := bufio.NewReaderSize(io, 64*1024)
	chunk := &Chunk{}

	for {
		buf, err := reader.ReadBytes('\n')
		if len(buf) > 0 && err == nil {
			line := buf[:len(buf)-1]
			chunk.lines[chunk.num] = &line
			chunk.num++

			if chunk.num == ChunkSize {
				r.mu.Lock()
				r.doc = append(r.doc, chunk)
				r.mu.Unlock()

				chunk = &Chunk{}
				r.mainEb.Put(EvtReadNew, nil)
			}
		}
		if err != nil {
			break
		}
	}

	if chunk.num != 0 {
		r.mu.Lock()
		r.doc = append(r.doc, chunk)
		r.mu.Unlock()
	}

	io.Close()

	r.mainEb.Put(EvtReadDone, nil)
}

// Snapshot returns a copy of the current items in the document
func (r *Reader) Snapshot() []*Chunk {
	r.mu.Lock()
	res := make([]*Chunk, len(r.doc))
	copy(res, r.doc)
	r.mu.Unlock()

	return res
}
