package expvar

import (
	"bytes"
	"encoding/json"
	"expvar"
	"math"
	"net/http"

	"github.com/upfluence/stats"
)

type Collector struct{}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Handler() http.Handler { return expvar.Handler() }
func (c *Collector) Close() error          { return nil }

type int64Wrapper struct {
	stats.Int64VectorGetter

	vType string
}

func (cw int64Wrapper) String() string {
	return serializeJSON(
		struct {
			Type  string
			Value []*stats.Int64Value
		}{Type: cw.vType, Value: cw.Get()},
	)
}

func (c *Collector) RegisterCounter(n string, g stats.Int64VectorGetter) {
	expvar.Publish(n, int64Wrapper{Int64VectorGetter: g, vType: "counter"})
}

func (c *Collector) RegisterGauge(n string, g stats.Int64VectorGetter) {
	expvar.Publish(n, int64Wrapper{Int64VectorGetter: g, vType: "gauge"})
}

type histogramWrapper struct {
	stats.HistogramVectorGetter
}

func (hw histogramWrapper) String() string {
	vs := hw.Get()

	for _, v := range vs {
		bs := make([]stats.Bucket, len(v.Buckets))

		for i, b := range v.Buckets {
			if math.IsInf(b.UpperBound, 0) {
				b.UpperBound = 0
			}

			bs[i] = b
		}

		v.Buckets = bs
	}

	return serializeJSON(
		struct {
			Type  string
			Value []*stats.HistogramValue
		}{Type: "histogram", Value: vs},
	)
}

func (c *Collector) RegisterHistogram(n string, g stats.HistogramVectorGetter) {
	expvar.Publish(n, histogramWrapper{HistogramVectorGetter: g})
}

func serializeJSON(payload interface{}) string {
	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(payload)

	return buf.String()
}
