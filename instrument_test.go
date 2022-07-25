package stats

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errMock = errors.New("mock")

func TestInstrument(t *testing.T) {
	for _, tt := range []struct {
		name         string
		instrumentFn func(Collector) Instrument
		in           error
		assert       func(*testing.T, Snapshot)
	}{
		{
			name: "plain success",
			instrumentFn: func(c Collector) Instrument {
				return NewInstrument(
					RootScope(c),
					"foo",
					WithTimerOptions(
						WithHistogramOptions(StaticBuckets(nil)),
					),
				)
			},
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(t, 0, len(s.Gauges))
				assert.Equal(
					t,
					[]Int64Snapshot{
						{
							Name:   "foo_started_total",
							Labels: map[string]string{},
							Value:  1,
						},
						{
							Name:   "foo_total",
							Labels: map[string]string{"status": "success"},
							Value:  1,
						},
					},
					s.Counters,
				)
				assert.Equal(t, 1, len(s.Histograms))
				assert.Equal(t, "foo_duration_seconds", s.Histograms[0].Name)
				assert.Equal(t, int64(1), s.Histograms[0].Value.Count)
				assert.Equal(t, 1, len(s.Histograms[0].Value.Buckets))
			},
		},
		{
			name: "custom counter label",
			instrumentFn: func(c Collector) Instrument {
				return NewInstrument(
					RootScope(c),
					"foo",
					WithCounterLabel("result"),
				)
			},
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(
					t,
					map[string]string{"result": "success"},
					s.Counters[1].Labels,
				)
			},
		},
		{
			name: "custom timer suffix",
			instrumentFn: func(c Collector) Instrument {
				return NewInstrument(
					RootScope(c),
					"foo",
					WithTimerOptions(
						WithHistogramOptions(StaticBuckets(nil)),
						WithTimerSuffix("_custom"),
					),
				)
			},
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(t, "foo_duration_custom", s.Histograms[0].Name)
			},
		},
		{
			name: "disable started counter",
			instrumentFn: func(c Collector) Instrument {
				return NewInstrument(
					RootScope(c),
					"foo",
					DisableStartedCounter(),
				)
			},
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(
					t,
					[]Int64Snapshot{
						{
							Name:   "foo_total",
							Labels: map[string]string{"status": "success"},
							Value:  1,
						},
					},
					s.Counters,
				)
			},
		},
		{
			name: "custom error formatter",
			instrumentFn: func(c Collector) Instrument {
				return NewInstrument(
					RootScope(c),
					"foo",
					WithFormatter(func(error) string { return "custom" }),
				)
			},
			in: errMock,
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(t, 0, len(s.Gauges))
				assert.Equal(
					t,
					[]Int64Snapshot{
						{
							Name:   "foo_started_total",
							Labels: map[string]string{},
							Value:  1,
						},
						{
							Name:   "foo_total",
							Labels: map[string]string{"status": "custom"},
							Value:  1,
						},
					},
					s.Counters,
				)
				assert.Equal(t, 1, len(s.Histograms))
				assert.Equal(t, "foo_duration_seconds", s.Histograms[0].Name)
			},
		},
		{
			name: "plain error",
			instrumentFn: func(c Collector) Instrument {
				return NewInstrument(RootScope(c), "foo")
			},
			in: errMock,
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(t, 0, len(s.Gauges))
				assert.Equal(
					t,
					[]Int64Snapshot{
						{
							Name:   "foo_started_total",
							Labels: map[string]string{},
							Value:  1,
						},
						{
							Name:   "foo_total",
							Labels: map[string]string{"status": "failed"},
							Value:  1,
						},
					},
					s.Counters,
				)
				assert.Equal(t, 1, len(s.Histograms))
				assert.Equal(t, "foo_duration_seconds", s.Histograms[0].Name)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := NewStaticCollector()
			i := tt.instrumentFn(c)

			err := i.Exec(func() error { return tt.in })

			assert.Equal(t, tt.in, err)
			tt.assert(t, c.Get())
		})
	}
}

func TestInstrumentVector(t *testing.T) {
	c := NewStaticCollector()
	iv := NewInstrumentVector(
		RootScope(c),
		"example",
		[]string{"foo", "bar"},
	)

	_ = iv.WithLabels("foo", "bar").Exec(func() error { return nil })
	_ = iv.WithLabels("biz", "buz").Exec(func() error { return nil })
	_ = iv.WithLabels("foo", "bar").Exec(func() error { return nil })
	_ = iv.WithLabels("foo", "bar").Exec(func() error { return errMock })

	assert.ElementsMatch(
		t,
		[]Int64Snapshot{
			{
				Name:   "example_started_total",
				Labels: map[string]string{"bar": "buz", "foo": "biz"},
				Value:  1,
			},
			{
				Name:   "example_started_total",
				Labels: map[string]string{"bar": "bar", "foo": "foo"},
				Value:  3,
			},
			{
				Name:   "example_total",
				Labels: map[string]string{"bar": "bar", "foo": "foo", "status": "failed"},
				Value:  1,
			},
			{
				Name:   "example_total",
				Labels: map[string]string{"bar": "bar", "foo": "foo", "status": "success"},
				Value:  2,
			},
			{
				Name:   "example_total",
				Labels: map[string]string{"bar": "buz", "foo": "biz", "status": "success"},
				Value:  1,
			},
		},
		c.Get().Counters,
	)
}
