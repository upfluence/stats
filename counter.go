package stats

// CounterVector is a multi-dimensional counter that creates counter instances
// with specific label values.
type CounterVector interface {
	// WithLabels returns a Counter with the specified label values.
	// The number of values must match the number of labels defined for this vector.
	WithLabels(...string) Counter
}

// Counter represents a monotonically increasing metric.
// Counters are used to track totals, such as request counts or error counts.
// Once incremented, a counter never decreases (except on process restart).
type Counter interface {
	// Inc increments the counter by 1.
	Inc()

	// Add increments the counter by the given value.
	// The value should be non-negative.
	Add(int64)

	// Get returns the current value of the counter.
	Get() int64
}

type counterVector struct {
	*atomicInt64Vector
}

func (cv counterVector) WithLabels(ls ...string) Counter {
	return cv.fetchValue(ls)
}

type partialCounterVector struct {
	cv CounterVector
	vs []string
}

func (pcv partialCounterVector) WithLabels(ls ...string) Counter {
	return pcv.cv.WithLabels(append(pcv.vs, ls...)...)
}

type reorderCounterVector struct {
	cv CounterVector
	labelOrderer
}

func (rcv reorderCounterVector) WithLabels(ls ...string) Counter {
	return rcv.cv.WithLabels(rcv.order(ls)...)
}

var (
	// NoopCounter is a counter that discards all operations.
	NoopCounter Counter = noopCounter{}

	// NoopCounterVector is a counter vector that returns noop counters.
	NoopCounterVector CounterVector = noopCounterVector{}
)

type noopCounter struct{}

func (noopCounter) Inc()       {}
func (noopCounter) Add(int64)  {}
func (noopCounter) Get() int64 { return 0 }

type noopCounterVector struct{}

func (noopCounterVector) WithLabels(...string) Counter { return noopCounter{} }
