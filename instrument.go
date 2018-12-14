package stats

type Instrument interface {
	Exec(func() error) error
}

type InstrumentOption func(*instrumentOptions)

var defaultOptions = instrumentOptions{formatter: defaultFormatter}

func WithFormatter(f ErrorFormatter) InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.formatter = f
	}
}

func WithHistogramOptions(hOpts ...HistogramOption) InstrumentOption {
	return func(opts *instrumentOptions) {
		opts.hOpts = hOpts
	}
}

type instrumentOptions struct {
	hOpts     []HistogramOption
	formatter ErrorFormatter
}

func NewInstrument(scope Scope, name string, iOpts ...InstrumentOption) Instrument {
	opts := defaultOptions

	for _, opt := range iOpts {
		opt(&opts)
	}

	return &instrument{
		instrumentOptions: opts,
		timer:             NewTimer(scope, name+"_duration", opts.hOpts...),
		started:           scope.Counter(name + "_started_total"),
		finished:          scope.CounterVector(name+"_total", []string{"status"}),
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
