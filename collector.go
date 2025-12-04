package stats

import "io"

// Collector is the interface that metrics backends must implement to receive
// and export metrics. Collectors are notified when new metrics are registered
// and can access the current values through the provided getter interfaces.
//
// Common implementations include prometheus.Collector and expvar.Collector.
type Collector interface {
	io.Closer

	// RegisterCounter registers a counter metric with the given name.
	RegisterCounter(string, Int64VectorGetter)

	// RegisterGauge registers a gauge metric with the given name.
	RegisterGauge(string, Int64VectorGetter)

	// RegisterHistogram registers a histogram metric with the given name.
	RegisterHistogram(string, HistogramVectorGetter)
}
