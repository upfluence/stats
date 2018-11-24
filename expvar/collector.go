package expvar

import (
	"bytes"
	"encoding/json"
	"expvar"
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
	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(
		struct {
			Type  string
			Value []*stats.Int64Value
		}{Type: cw.vType, Value: cw.Get()},
	)

	return buf.String()
}

func (c *Collector) RegisterCounter(n string, g stats.Int64VectorGetter) {
	expvar.Publish(n, int64Wrapper{Int64VectorGetter: g, vType: "counter"})
}

func (c *Collector) RegisterGauge(n string, g stats.Int64VectorGetter) {
	expvar.Publish(n, int64Wrapper{Int64VectorGetter: g, vType: "gauge"})
}
