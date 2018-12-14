package stats

import (
	"sync"
	"time"
)

type StopWatch interface {
	Stop()
}

type Timer interface {
	Start() StopWatch
}

type timer struct {
	Histogram

	p sync.Pool
}

func (t *timer) Start() StopWatch {
	var sw = t.p.Get().(*stopWatch)

	sw.t0 = time.Now()

	return sw
}

type stopWatch struct {
	t0    time.Time
	timer *timer
}

func (sw *stopWatch) Stop() {
	sw.timer.Record(time.Since(sw.t0).Seconds())
	sw.timer.p.Put(sw)
}

func NewTimer(scope Scope, name string, opts ...HistogramOption) Timer {
	var t = &timer{Histogram: scope.Histogram(name+"_seconds", opts...)}

	t.p = sync.Pool{New: func() interface{} { return &stopWatch{timer: t} }}

	return t
}
