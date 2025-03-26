package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiIncarnationScope(t *testing.T) {
	c := NewStaticCollector()

	sc := LocalIncarnationScope(RootScope(c), "incarnation")

	g1 := sc.Scope("foo", map[string]string{"bar": "buz"}).Gauge("bar")

	g1.Update(1)

	sc.GaugeVector("foo_bar", []string{"bar"}).WithLabels("buz").Update(2)

	g1.Update(3)

	sc.Scope("foo", map[string]string{"bar": "buz"}).Gauge("bar").Update(4)

	sc.Counter("foo_bar_1").Inc()

	assert.Equal(
		t,
		[]Int64Snapshot{
			{Name: "foo_bar", Labels: map[string]string{"bar": "buz", "incarnation": "0"}, Value: 3},
			{Name: "foo_bar", Labels: map[string]string{"bar": "buz", "incarnation": "1"}, Value: 2},
			{Name: "foo_bar", Labels: map[string]string{"bar": "buz", "incarnation": "2"}, Value: 4},
		},
		c.Get().Gauges,
	)

	assert.Equal(
		t,
		[]Int64Snapshot{
			{Name: "foo_bar_1", Labels: map[string]string{"incarnation": "0"}, Value: 1},
		},
		c.Get().Counters,
	)
}
