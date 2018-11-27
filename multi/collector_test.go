package multi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upfluence/stats"
)

type mockCollector struct {
	closeCalled             bool
	registerCounterCalled   bool
	registerGaugeCalled     bool
	registerHistogramCalled bool
}

func (m *mockCollector) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockCollector) RegisterCounter(string, stats.Int64VectorGetter) {
	m.registerCounterCalled = true
}

func (m *mockCollector) RegisterGauge(string, stats.Int64VectorGetter) {
	m.registerGaugeCalled = true
}

func (m *mockCollector) RegisterHistogram(string, stats.HistogramVectorGetter) {
	m.registerHistogramCalled = true
}

var c0, c1 mockCollector

func TestClose(t *testing.T) {
	WrapCollectors(&c0, &c1).Close()

	assert.True(t, c0.closeCalled)
	assert.True(t, c1.closeCalled)
}

func TestRegisterHistogram(t *testing.T) {
	WrapCollectors(&c0, &c1).RegisterHistogram("foo", nil)

	assert.True(t, c0.registerHistogramCalled)
	assert.True(t, c1.registerHistogramCalled)
}

func TestRegisterCounter(t *testing.T) {
	WrapCollectors(&c0, &c1).RegisterCounter("foo", nil)

	assert.True(t, c0.registerCounterCalled)
	assert.True(t, c1.registerCounterCalled)
}

func TestRegisterGauge(t *testing.T) {
	WrapCollectors(&c0, &c1).RegisterGauge("foo", nil)

	assert.True(t, c0.registerGaugeCalled)
	assert.True(t, c1.registerGaugeCalled)
}

func TestWrapCollectors(t *testing.T) {
	for _, tt := range []struct {
		in  []stats.Collector
		out stats.Collector
	}{
		{},
		{in: []stats.Collector{&c0}, out: &c0},
		{
			in:  []stats.Collector{&c0, &c1},
			out: multiCollector([]stats.Collector{&c0, &c1}),
		},
	} {
		assert.Equal(t, tt.out, WrapCollectors(tt.in...))
	}
}
