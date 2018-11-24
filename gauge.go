package stats

type GaugeVector interface {
	Int64VectorGetter

	WithLabels(...string) Gauge
}

type Gauge interface {
	Update(int64)
	Get() int64
}

type gaugeVector struct {
	*atomicInt64Vector
}

func (cv *gaugeVector) WithLabels(ls ...string) Gauge {
	return cv.fetchValue(ls)
}
