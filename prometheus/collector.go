package prometheus

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/upfluence/stats"
)

var DefaultCollector = NewCollector(prometheus.DefaultRegisterer)

type Collector struct {
	r prometheus.Registerer

	histogramGettersMu sync.Mutex
	histogramGetters   map[string]*multiHistogramVectorGetter

	int64GettersMu sync.Mutex
	int64Getters   map[string]*multiInt64VectorGetter
}

// NewDefaultCollector returns a collector based on the default prometheus
// backend
//
// Deprecated: Prefer using the DefaultCollector singleton
func NewDefaultCollector() *Collector {
	return NewCollector(prometheus.DefaultRegisterer)
}

func NewCollector(r prometheus.Registerer) *Collector {
	c := &Collector{
		r:                r,
		histogramGetters: make(map[string]*multiHistogramVectorGetter),
		int64Getters:     make(map[string]*multiInt64VectorGetter),
	}

	return c
}

func (c *Collector) Close() error { return nil }
func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
}

func (c *Collector) RegisterHistogram(n string, g stats.HistogramVectorGetter) {
	c.histogramGettersMu.Lock()
	mhvg, ok := c.histogramGetters[n]

	if !ok {
		mhvg = &multiHistogramVectorGetter{}
		c.histogramGetters[n] = mhvg
	}

	c.histogramGettersMu.Unlock()

	mhvg.appendGetter(n, g)

	if ok {
		return
	}

	c.r.MustRegister(
		&histogramWrapper{
			g:    mhvg,
			n:    n,
			desc: prometheus.NewDesc(n, "no help", g.Labels(), nil),
		},
	)
}

func (c *Collector) RegisterGauge(n string, g stats.Int64VectorGetter) {
	c.registerInt64Collector(n, g, gauge)
}

func (c *Collector) RegisterCounter(n string, g stats.Int64VectorGetter) {
	c.registerInt64Collector(n, g, counter)
}

func (c *Collector) registerInt64Collector(n string, g stats.Int64VectorGetter, m registrarMode) {
	c.int64GettersMu.Lock()
	mivg, ok := c.int64Getters[n]

	if !ok {
		mivg = &multiInt64VectorGetter{}
		c.int64Getters[n] = mivg
	}

	c.int64GettersMu.Unlock()

	mivg.appendGetter(n, m, g)

	if ok {
		return
	}

	c.r.MustRegister(
		&int64Wrapper{
			g:       mivg,
			n:       n,
			desc:    prometheus.NewDesc(n, "no help", g.Labels(), nil),
			stapler: registrarModeOps[m],
		},
	)
}
