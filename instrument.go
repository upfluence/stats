package stats

import "fmt"

type InstrumentVector interface {
	WithLabels(...string) Instrument
}

type instrumentVector struct {
	entityVector

	scope Scope
	name  string
	opts  instrumentOptions
}

func NewInstrumentVector(scope Scope, name string, labels []string, opts ...InstrumentOption) InstrumentVector {
	var iv = instrumentVector{
		entityVector: entityVector{
			marshaler: newDefaultMarshaler(),
			labels:    labels,
		},
		scope: scope,
		name:  name,
		opts:  defaultInstrumentOptions,
	}

	for _, opt := range opts {
		opt(&iv.opts)
	}

	iv.newFunc = iv.newInstrument

	return &iv
}

func (iv *instrumentVector) newInstrument(vs map[string]string) interface{} {
	return newInstrument(iv.scope.Scope("", vs), iv.name, iv.opts)
}

func (iv *instrumentVector) WithLabels(ls ...string) Instrument {
	return iv.entity(ls).(*instrument)
}

type Instrument interface {
	Exec(func() error) error
}

type InstrumentOption func(*instrumentOptions)

var defaultInstrumentOptions = instrumentOptions{
	formatter:    defaultFormatter,
	trackStarted: true,
	counterLabel: "status",
}

func DisableStartedCounter() InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.trackStarted = false
	}
}

func WithFormatter(f ErrorFormatter) InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.formatter = f
	}
}

func WithCounterLabel(s string) InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.counterLabel = s
	}
}

func WithTimerOptions(tOpts ...TimerOption) InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.tOpts = tOpts
	}
}

type instrumentOptions struct {
	formatter    ErrorFormatter
	tOpts        []TimerOption
	trackStarted bool
	counterLabel string
}

func NewInstrument(scope Scope, name string, iOpts ...InstrumentOption) Instrument {
	var opts = defaultInstrumentOptions

	for _, opt := range iOpts {
		opt(&opts)
	}

	return newInstrument(scope, name, opts)
}

func newInstrument(scope Scope, name string, opts instrumentOptions) *instrument {
	var startedCounter Counter = noopCounter{}

	if opts.trackStarted {
		startedCounter = scope.Counter(fmt.Sprintf("%s_started_total", name))
	}

	return &instrument{
		instrumentOptions: opts,
		timer: NewTimer(
			scope,
			fmt.Sprintf("%s_duration", name),
			opts.tOpts...,
		),
		started: startedCounter,
		finished: scope.CounterVector(
			fmt.Sprintf("%s_total", name),
			[]string{opts.counterLabel},
		),
	}
}

type ErrorFormatter func(error) string

type instrument struct {
	instrumentOptions

	finished CounterVector
	started  Counter
	timer    Timer
}

func (i *instrument) Exec(fn func() error) error {
	i.started.Inc()
	sw := i.timer.Start()

	err := fn()

	sw.Stop()
	i.finished.WithLabels(i.formatter(err)).Inc()

	return err
}

func defaultFormatter(err error) string {
	if err == nil {
		return "success"
	}

	return err.Error()
}
