package stats

import (
	"bytes"
	"encoding/json"
	"sort"
)

type StaticCollector struct {
	counters   map[string]Int64VectorGetter
	gauges     map[string]Int64VectorGetter
	histograms map[string]HistogramVectorGetter
}

func NewStaticCollector() *StaticCollector {
	return &StaticCollector{
		counters:   make(map[string]Int64VectorGetter),
		gauges:     make(map[string]Int64VectorGetter),
		histograms: make(map[string]HistogramVectorGetter),
	}
}

func (c *StaticCollector) Close() error { return nil }

func (c *StaticCollector) RegisterCounter(n string, g Int64VectorGetter) {
	c.counters[n] = g
}

func (c *StaticCollector) RegisterGauge(n string, g Int64VectorGetter) {
	c.gauges[n] = g
}

func (c *StaticCollector) RegisterHistogram(n string, g HistogramVectorGetter) {
	c.histograms[n] = g
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

type HistogramSnapshot struct {
	Name  string
	Value HistogramValue
}

type Snapshot struct {
	Counters   []Int64Snapshot
	Gauges     []Int64Snapshot
	Histograms []HistogramSnapshot
}

func compareLabels(x, y map[string]string) int {
	var (
		xj, _ = json.Marshal(x)
		yj, _ = json.Marshal(y)
	)

	return bytes.Compare(xj, yj)
}

type Int64Snapshots []Int64Snapshot

func (ss Int64Snapshots) Len() int { return len(ss) }

func (ss Int64Snapshots) Less(i int, j int) bool {
	if ss[i].Name != ss[j].Name {
		return ss[i].Name < ss[j].Name
	}

	return compareLabels(ss[i].Labels, ss[j].Labels) < 0
}

func (ss Int64Snapshots) Swap(i int, j int) {
	ss[j], ss[i] = ss[i], ss[j]
}

type HistogramSnapshots []HistogramSnapshot

func (ss HistogramSnapshots) Len() int { return len(ss) }

func (ss HistogramSnapshots) Less(i int, j int) bool {
	if ss[i].Name != ss[j].Name {
		return ss[i].Name < ss[j].Name
	}

	return compareLabels(ss[i].Value.Tags, ss[j].Value.Tags) < 0
}

func (ss HistogramSnapshots) Swap(i int, j int) {
	ss[j], ss[i] = ss[i], ss[j]
}

func (c *StaticCollector) Get() Snapshot {
	var (
		counters, gauges []Int64Snapshot
		histograms       []HistogramSnapshot
	)

	for n, g := range c.counters {
		counters = append(counters, int64snapshots(n, g)...)
	}

	for n, g := range c.gauges {
		gauges = append(gauges, int64snapshots(n, g)...)
	}

	for n, g := range c.histograms {
		for _, v := range g.Get() {
			histograms = append(histograms, HistogramSnapshot{Name: n, Value: *v})
		}
	}

	sort.Sort(Int64Snapshots(counters))
	sort.Sort(Int64Snapshots(gauges))
	sort.Sort(HistogramSnapshots(histograms))

	return Snapshot{Counters: counters, Gauges: gauges, Histograms: histograms}
}
