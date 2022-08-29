package prometheus

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/upfluence/stats"
)

type registrarMode int

const (
	counter registrarMode = iota + 1
	gauge
)

var registrarModeOps = map[registrarMode]func(*dto.Metric, float64){
	gauge:   func(m *dto.Metric, v float64) { m.Gauge = &dto.Gauge{Value: &v} },
	counter: func(m *dto.Metric, v float64) { m.Counter = &dto.Counter{Value: &v} },
}

type multiInt64VectorGetter struct {
	mode registrarMode
	gs   []stats.Int64VectorGetter
}

func (mivg *multiInt64VectorGetter) appendGetter(n string, m registrarMode, g stats.Int64VectorGetter) {
	if len(mivg.gs) > 0 {
		if hashSlice(g.Labels()) != hashSlice(mivg.gs[0].Labels()) {
			panic(
				fmt.Sprintf(
					"%s: int64 vector cannot be registered because the labels are different, has: %v, want %v",
					n,
					g.Labels(),
					mivg.gs[0].Labels(),
				),
			)
		}

		if mivg.mode != m {
			panic(
				fmt.Sprintf(
					"%s: invalid int64 vector mode, passed a counter but is a gauge or vice versa",
					n,
				),
			)
		}
	}

	mivg.mode = m
	mivg.gs = append(mivg.gs, g)
}

func (mivg *multiInt64VectorGetter) Labels() []string {
	if len(mivg.gs) == 0 {
		return nil
	}

	return mivg.gs[0].Labels()
}

func (mivg *multiInt64VectorGetter) Get() []*stats.Int64Value {
	switch len(mivg.gs) {
	case 0:
		return nil
	case 1:
		return mivg.gs[0].Get()
	}

	var (
		tags   = make(map[uint64]map[string]string)
		values = make(map[uint64]int64)
	)

	for _, g := range mivg.gs {
		for _, iv := range g.Get() {
			key := hashTags(iv.Tags)

			if _, ok := tags[key]; !ok {
				tags[key] = iv.Tags
				values[key] = iv.Value
			} else if mivg.mode == counter {
				values[key] += iv.Value
			}
		}
	}

	res := make([]*stats.Int64Value, 0, len(tags))

	for key, ts := range tags {
		res = append(res, &stats.Int64Value{Tags: ts, Value: values[key]})
	}

	return res
}

type int64Wrapper struct {
	g stats.Int64VectorGetter
	n string

	desc *prometheus.Desc

	stapler func(*dto.Metric, float64)
}

type int64MetricImpl struct {
	desc    *prometheus.Desc
	v       *stats.Int64Value
	stapler func(*dto.Metric, float64)
}

func (im *int64MetricImpl) Desc() *prometheus.Desc { return im.desc }
func (im *int64MetricImpl) Write(m *dto.Metric) error {
	var ps []*dto.LabelPair

	for k, v := range im.v.Tags {
		k, v := k, v
		ps = append(ps, &dto.LabelPair{Name: &k, Value: &v})
	}

	im.stapler(m, float64(im.v.Value))
	m.Label = ps

	return nil
}

func (cw *int64Wrapper) metrics() []prometheus.Metric {
	var ms []prometheus.Metric

	for _, v := range cw.g.Get() {
		ms = append(ms, &int64MetricImpl{desc: cw.desc, v: v, stapler: cw.stapler})
	}

	return ms
}

func (iw *int64Wrapper) Describe(ch chan<- *prometheus.Desc) {
	ch <- iw.desc
}

func (iw *int64Wrapper) Collect(ch chan<- prometheus.Metric) {
	for _, m := range iw.metrics() {
		ch <- m
	}
}
