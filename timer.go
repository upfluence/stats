package stats

import (
	"sync"
	"time"
)

type TimerVector interface {
	WithLabels(...string) Timer
}

type timerVector struct {
	entityVector

	scope Scope
	name  string
	opts  timerOptions
}

func NewTimerVector(scope Scope, name string, labels []string, opts ...TimerOption) TimerVector {
	var tv = timerVector{
		entityVector: entityVector{
			marshaler: newDefaultMarshaler(),
			labels:    labels,
		},
		scope: scope,
		name:  name,
		opts:  defaultTimerOptions,
	}

	for _, opt := range opts {
		opt(&tv.opts)
	}

	tv.newFunc = tv.newTimer

	return &tv
}

func (tv *timerVector) newTimer(vs map[string]string) interface{} {
	return newTimer(tv.scope.Scope("", vs), tv.name, tv.opts)
}

func (tv *timerVector) WithLabels(ls ...string) Timer {
	return tv.entity(ls).(*timer)
}

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

type TimerOption func(*timerOptions)

type timerOptions struct {
	hOpts  []HistogramOption
	suffix string
}

func WithHistogramOptions(hOpts ...HistogramOption) TimerOption {
	return func(opts *timerOptions) {
		opts.hOpts = hOpts
	}
}

func WithTimerSuffix(s string) TimerOption {
	return func(opts *timerOptions) {
		opts.suffix = s
	}
}

var defaultTimerOptions = timerOptions{
	suffix: "_seconds",
}

func (sw *stopWatch) Stop() {
	sw.timer.Record(time.Since(sw.t0).Seconds())
	sw.timer.p.Put(sw)
}

func NewTimer(scope Scope, name string, tOpts ...TimerOption) Timer {
	var opts = defaultTimerOptions

	for _, opt := range tOpts {
		opt(&opts)
	}

	return newTimer(scope, name, opts)
}

func newTimer(scope Scope, name string, opts timerOptions) *timer {
	var t = timer{Histogram: scope.Histogram(name+opts.suffix, opts.hOpts...)}

	t.p = sync.Pool{New: func() interface{} { return &stopWatch{timer: &t} }}

	return &t
}
