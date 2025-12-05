package stats

import (
	"sync"
	"time"
)

// TimerVector is a multi-dimensional timer that creates timer instances
// with specific label values.
type TimerVector interface {
	// WithLabels returns a Timer with the specified label values.
	// The number of values must match the number of labels defined for this vector.
	WithLabels(...string) Timer
}

type timerVector struct {
	entityVector

	scope Scope
	name  string
	opts  timerOptions
}

// NewTimerVector creates a new timer vector with the given scope, name, labels, and options.
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

// StopWatch represents an active timing measurement.
// Call Stop to record the elapsed duration.
type StopWatch interface {
	// Stop records the elapsed time since Start was called.
	Stop()
}

// Timer is a convenience wrapper around Histogram for measuring durations.
// It automatically records durations in seconds by default.
type Timer interface {
	// Start begins a new timing measurement and returns a StopWatch.
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

// TimerOption configures a Timer with custom settings.
type TimerOption func(*timerOptions)

type timerOptions struct {
	hOpts  []HistogramOption
	suffix string
}

// WithHistogramOptions configures the underlying histogram with custom options.
func WithHistogramOptions(hOpts ...HistogramOption) TimerOption {
	return func(opts *timerOptions) {
		opts.hOpts = hOpts
	}
}

// WithTimerSuffix configures a custom suffix for the timer's metric name.
// Default suffix is "_seconds".
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

// NewTimer creates a new timer with the given scope, name, and options.
// The timer automatically appends "_seconds" suffix to the name (configurable via WithTimerSuffix).
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

type noopTimer struct{}

func (nt noopTimer) Start() StopWatch {
	return noopStopWatch{}
}

type noopStopWatch struct{}

func (nsw noopStopWatch) Stop() {}

// NoopTimer is a no-operation timer that does nothing.
// Useful for disabling timing measurements.
var NoopTimer Timer = noopTimer{}
