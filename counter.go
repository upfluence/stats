package stats

type CounterVector interface {
	Int64VectorGetter

	WithLabels(...string) Counter
}

type Counter interface {
	Inc()
	Add(int64)

	Get() int64
}

type counterVector struct {
	*atomicInt64Vector
}

func (cv *counterVector) WithLabels(ls ...string) Counter {
	return cv.fetchValue(ls)
}
