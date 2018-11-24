package stats

type StaticCollector struct {
	counters map[string]Int64VectorGetter
	gauges   map[string]Int64VectorGetter
}

func NewStaticCollector() *StaticCollector {
	return &StaticCollector{
		counters: make(map[string]Int64VectorGetter),
		gauges:   make(map[string]Int64VectorGetter),
	}
}

func (c *StaticCollector) Close() error { return nil }

func (c *StaticCollector) RegisterCounter(n string, g Int64VectorGetter) {
	c.counters[n] = g
}

func (c *StaticCollector) RegisterGauge(n string, g Int64VectorGetter) {
	c.gauges[n] = g
}

type Int64Snapshot struct {
	Name   string
	Labels map[string]string
	Value  int64
}

func int64snapshots(n string, g Int64VectorGetter) []Int64Snapshot {
	var sns []Int64Snapshot

	for _, v := range g.Get() {
		sns = append(
			sns,
			Int64Snapshot{
				Name:   n,
				Labels: v.Tags,
				Value:  v.Value,
			},
		)
	}

	return sns
}

type Snapshot struct {
	Counters []Int64Snapshot
	Gauges   []Int64Snapshot
}

func (c *StaticCollector) Get() Snapshot {
	var counters, gauges []Int64Snapshot

	for n, g := range c.counters {
		counters = append(counters, int64snapshots(n, g)...)
	}

	for n, g := range c.gauges {
		gauges = append(gauges, int64snapshots(n, g)...)
	}

	return Snapshot{Counters: counters, Gauges: gauges}
}
