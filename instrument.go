package stats

import "fmt"

// InstrumentVector is a multi-dimensional instrument that creates instrument instances
// with specific label values.
type InstrumentVector interface {
	// WithLabels returns an Instrument with the specified label values.
	// The number of values must match the number of labels defined for this vector.
	WithLabels(...string) Instrument
}

type instrumentVector struct {
	entityVector

	scope Scope
	name  string
	opts  instrumentOptions
}

// NewInstrumentVector creates a new instrument vector with the given scope, name, labels, and options.
func NewInstrumentVector(scope Scope, name string, labels []string, opts ...InstrumentOption) InstrumentVector {
	if _, ok := scope.(noopScope); ok {
		return NoopInstrumentVector
	}

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

// Instrument provides automatic instrumentation for function execution.
// It tracks three metrics automatically:
//   - <name>_started_total: counter of operation starts
//   - <name>_total{status="..."}: counter of completions by status
//   - <name>_duration_seconds: histogram of execution durations
//
// The started counter is useful for computing in-flight operations:
// in_flight = started_total - sum(total)
type Instrument interface {
	// Exec executes the given function and records metrics.
	// Returns the error from the function unchanged.
	Exec(func() error) error
}

// InstrumentOption configures an Instrument with custom settings.
type InstrumentOption func(*instrumentOptions)

var defaultInstrumentOptions = instrumentOptions{
	formatter:     defaultFormatter,
	trackStarted:  true,
	trackDuration: true,
	counterLabel:  "status",
}

// DisableStartedCounter disables tracking of the started counter.
// Use this when you only care about completions and duration, not in-flight count.
func DisableStartedCounter() InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.trackStarted = false
	}
}

// DisableDurationTracking disables tracking of operation duration.
// Use this when you only care about counters, not timing.
func DisableDurationTracking() InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.trackDuration = false
	}
}

// WithFormatter configures a custom error formatter to categorize errors.
// The formatter converts errors into status label values.
// Default formatter returns "success" for nil and "failed" for any error.
func WithFormatter(f ErrorFormatter) InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.formatter = f
	}
}

// WithCounterLabel configures a custom label name for the status label.
// Default is "status".
func WithCounterLabel(s string) InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.counterLabel = s
	}
}

// WithTimerOptions configures the underlying timer with custom options.
func WithTimerOptions(tOpts ...TimerOption) InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.tOpts = tOpts
	}
}

type instrumentOptions struct {
	formatter     ErrorFormatter
	tOpts         []TimerOption
	trackStarted  bool
	trackDuration bool
	counterLabel  string
}

// NewInstrument creates a new instrument with the given scope, name, and options.
//
// The instrument automatically creates three metrics:
//   - <name>_started_total: counter (optional, see DisableStartedCounter)
//   - <name>_total{status="..."}: counter with status label
//   - <name>_duration_seconds: histogram (optional, see DisableDurationTracking)
func NewInstrument(scope Scope, name string, iOpts ...InstrumentOption) Instrument {
	if _, ok := scope.(noopScope); ok {
		return NoopInstrument
	}

	var opts = defaultInstrumentOptions

	for _, opt := range iOpts {
		opt(&opts)
	}

	return newInstrument(scope, name, opts)
}

func newInstrument(scope Scope, name string, opts instrumentOptions) *instrument {
	var (
		startedCounter Counter = noopCounter{}
		timer          Timer   = noopTimer{}
	)

	if opts.trackStarted {
		startedCounter = scope.Counter(fmt.Sprintf("%s_started_total", name))
	}

	if opts.trackDuration {
		timer = NewTimer(
			scope,
			fmt.Sprintf("%s_duration", name),
			opts.tOpts...,
		)
	}

	return &instrument{
		instrumentOptions: opts,
		timer:             timer,
		started:           startedCounter,
		finished: scope.CounterVector(
			fmt.Sprintf("%s_total", name),
			[]string{opts.counterLabel},
		),
	}
}

// ErrorFormatter converts an error into a status label value.
// Used by Instrument to categorize operation results.
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

// defaultFormatter , look into https://github.com/upfluence/errors/blob/master/stats/statuser.go#L36
// for more advanced reporting
func defaultFormatter(err error) string {
	if err == nil {
		return "success"
	}

	return "failed"
}

// ExecInstrument2 is a generic wrapper around Instrument.Exec that handles functions
// returning both a value and an error. This is a convenience function for instrumenting
// operations that return results.
//
// Note: The error from the instrumented function is tracked but not returned by this
// function. Use Instrument.Exec directly if you need to handle the error.
func ExecInstrument2[T any](i Instrument, fn func() (T, error)) (T, error) {
	var res T

	i.Exec(func() error {
		var err error

		res, err = fn()

		return err
	})

	return res, nil
}

type noopInstrument struct{}

func (n noopInstrument) Exec(fn func() error) error {
	return fn()
}

type noopInstrumentVector struct{}

func (n noopInstrumentVector) WithLabels(...string) Instrument {
	return noopInstrument{}
}

var (
	// NoopInstrument is an instrument that discards all operations.
	// Useful for testing or when instrumentation is conditionally disabled.
	NoopInstrument Instrument = noopInstrument{}

	// NoopInstrumentVector is an instrument vector that returns noop instruments.
	// Useful for testing or when instrumentation is conditionally disabled.
	NoopInstrumentVector InstrumentVector = noopInstrumentVector{}
)
