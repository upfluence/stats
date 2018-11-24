package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/upfluence/stats"
)

type Collector struct {
	r prometheus.Registerer
}

func NewDefaultCollector() *Collector {
	return NewCollector(prometheus.DefaultRegisterer)
}

func NewCollector(r prometheus.Registerer) *Collector {
	c := &Collector{r: r}

	return c
}

func (c *Collector) Close() error          { return nil }
func (c *Collector) Handler() http.Handler { return prometheus.Handler() }

func (c *Collector) RegisterGauge(n string, g stats.Int64VectorGetter) {
	c.registerInt64Collector(
		n,
		g,
		func(m *dto.Metric, v float64) {
			m.Gauge = &dto.Gauge{Value: &v}
		},
	)
}

func (c *Collector) RegisterCounter(n string, g stats.Int64VectorGetter) {
	c.registerInt64Collector(
		n,
		g,
		func(m *dto.Metric, v float64) {
			m.Counter = &dto.Counter{Value: &v}
		},
	)
}

func (c *Collector) registerInt64Collector(n string, g stats.Int64VectorGetter, fn func(*dto.Metric, float64)) {
	c.r.MustRegister(
		&int64Wrapper{
			g:       g,
			n:       n,
			desc:    prometheus.NewDesc(n, "no help", g.Labels(), nil),
			stapler: fn,
		},
	)
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
