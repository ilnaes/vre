package vre

import (
	"bufio"
	"os"
	"sync"
)

type Doc struct {
	chunks   []*Chunk
	filename string
	numLines int
}

type Chunk struct {
	lines [ChunkSize]*[]byte
	num   int
}

// Reader acts as the model
type Reader struct {
	mu     sync.Mutex
	mainEb *EventBox
	doc    []*Doc
}

func NewReader(eb *EventBox) *Reader {
	return &Reader{
		mainEb: eb,
		mu:     sync.Mutex{},
		doc:    make([]*Doc, 0),
	}
}

// ReadFiles reads the files given
func (r *Reader) ReadFiles(fs []string) {
	for i, fname := range fs {
		f, err := os.Open(fname)
		if err != nil {
			r.mainEb.Put(EvtReadError, fname)
		}

		r.ReadFile(f, fname, i == len(fs)-1)
	}
}

// ReadStream reads the file in ChunkSize chunks and appends to Reader
func (r *Reader) ReadFile(io *os.File, name string, final bool) {
	doc := Doc{
		chunks:   make([]*Chunk, 0),
		filename: name,
	}
	r.doc = append(r.doc, &doc)

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
				doc.chunks = append(doc.chunks, chunk)
				doc.numLines += chunk.num
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
		doc.chunks = append(doc.chunks, chunk)
		doc.numLines += chunk.num
		r.mu.Unlock()
	}

	io.Close()

	if final {
		// report finished reading
		r.mainEb.Put(EvtReadDone, nil)
	} else {
		r.mainEb.Put(EvtReadNew, nil)
	}
}

// Snapshot returns a copy of the current items in the document
func (r *Reader) Snapshot() []*Doc {
	r.mu.Lock()
	res := make([]*Doc, len(r.doc))
	copy(res, r.doc)
	r.mu.Unlock()

	return res
}
