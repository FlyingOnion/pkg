package log

import (
	"sync"
	"time"
)

var eventPool = sync.Pool{
	New: func() any {
		return &Event{}
	},
}

func newEvent(l Level, callerSkip int, t time.Time) *Event {
	if e, ok := eventPool.Get().(*Event); ok {
		e.Level = l
		e.CallerSkip = callerSkip
		e.Time = t
		return e
	}
	return &Event{Level: l, CallerSkip: callerSkip, Time: t}
}

func recycleEvent(e *Event) {
	eventPool.Put(e)
}

type Event struct {
	CallerSkip int
	Level      Level
	Time       time.Time
}
