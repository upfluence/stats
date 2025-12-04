package stats

// GaugeVector is a multi-dimensional gauge that creates gauge instances
// with specific label values.
type GaugeVector interface {
	// WithLabels returns a Gauge with the specified label values.
	// The number of values must match the number of labels defined for this vector.
	WithLabels(...string) Gauge
}

// Gauge represents a metric that can increase or decrease.
// Gauges are used to track current values, such as active connections,
// memory usage, or queue depth.
type Gauge interface {
	// Update sets the gauge to the given value.
	Update(int64)

	// Get returns the current value of the gauge.
	Get() int64
}

type gaugeVector struct {
	*atomicInt64Vector
}

func (gv gaugeVector) WithLabels(ls ...string) Gauge {
	return gv.fetchValue(ls)
}

type partialGaugeVector struct {
	gv GaugeVector
	vs []string
}

func (pgv partialGaugeVector) WithLabels(ls ...string) Gauge {
	return pgv.gv.WithLabels(append(pgv.vs, ls...)...)
}

type reorderGaugeVector struct {
	gv GaugeVector
	labelOrderer
}

func (rgv reorderGaugeVector) WithLabels(ls ...string) Gauge {
	return rgv.gv.WithLabels(rgv.order(ls)...)
}

var (
	// NoopGauge is a gauge that discards all operations.
	NoopGauge Gauge = noopGauge{}

	// NoopGaugeVector is a gauge vector that returns noop gauges.
	NoopGaugeVector GaugeVector = noopGaugeVector{}
)

type noopGauge struct{}

func (noopGauge) Update(int64) {}
func (noopGauge) Get() int64   { return 0 }

type noopGaugeVector struct{}

func (noopGaugeVector) WithLabels(...string) Gauge { return noopGauge{} }
