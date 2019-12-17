package vre

import (
	"sync"
)

type EventType int
type Events map[EventType]interface{}

type EventBox struct {
	cond   *sync.Cond
	events Events
}

func NewEventBox() *EventBox {
	m := &sync.Mutex{}

	return &EventBox{
		cond:   sync.NewCond(m),
		events: make(map[EventType]interface{}),
	}
}

// Clear removes all events.  Typically used in callback for Wait
func (eb *EventBox) Clear() {
	for e := range eb.events {
		delete(eb.events, e)
	}
}

// Wait runs when Put has been called and applies callback to the events
func (eb *EventBox) Wait(callback func(*Events)) {
	eb.cond.L.Lock()

	if len(eb.events) == 0 {
		eb.cond.Wait()
	}

	callback(&eb.events)
	eb.cond.L.Unlock()
}

func (eb *EventBox) Put(e EventType, val interface{}) {
	eb.cond.L.Lock()
	eb.events[e] = val
	eb.cond.Broadcast()
	eb.cond.L.Unlock()
}
