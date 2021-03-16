package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimerVector(t *testing.T) {
	c := NewStaticCollector()
	tv := NewTimerVector(RootScope(c), "example", []string{"foo", "bar"})

	tv.WithLabels("foo", "bar").Start().Stop()
	tv.WithLabels("biz", "biz").Start().Stop()
	tv.WithLabels("foo", "bar").Start().Stop()

	assert.Len(t, c.Get().Histograms, 2)

	assert.Equal(t, "example_seconds", c.Get().Histograms[1].Name)
	assert.Equal(t, int64(1), c.Get().Histograms[1].Value.Count)
	assert.Equal(
		t,
		map[string]string{"bar": "biz", "foo": "biz"},
		c.Get().Histograms[1].Value.Tags,
	)

	assert.Equal(t, "example_seconds", c.Get().Histograms[0].Name)
	assert.Equal(t, int64(2), c.Get().Histograms[0].Value.Count)
	assert.Equal(
		t,
		map[string]string{"bar": "bar", "foo": "foo"},
		c.Get().Histograms[0].Value.Tags,
	)
}
