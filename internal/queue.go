package vre

import (
	"sync"
)

type Event int

type Queue struct {
	cond *sync.Cond
}

func (q *Queue) Wait() {
	q.cond.L.Lock()

	q.cond.Wait()

	q.cond.L.Unlock()
}

func (q *Queue) Put(e Event) {
	q.cond.L.Lock()

	q.cond.L.Unlock()
}
