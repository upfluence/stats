package stats

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errMock = errors.New("mock")

func TestInstrument(t *testing.T) {
	for _, tt := range []struct {
		name    string
		timerFn func(Collector) Instrument
		in      error
		assert  func(*testing.T, Snapshot)
	}{
		{
			name: "plain success",
			timerFn: func(c Collector) Instrument {
				return NewInstrument(
					RootScope(c),
					"foo",
					WithHistogramOptions(StaticBuckets(nil)),
				)
			},
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(t, 0, len(s.Gauges))
				assert.Equal(
					t,
					[]Int64Snapshot{
						Int64Snapshot{
							Name:   "foo_started_total",
							Labels: map[string]string{},
							Value:  1,
						},
						Int64Snapshot{
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
			name: "custom timer suffix",
			timerFn: func(c Collector) Instrument {
				return NewInstrument(
					RootScope(c),
					"foo",
					WithHistogramOptions(StaticBuckets(nil)),
					WithTimerSuffixOptions("_custom"),
				)
			},
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(t, "foo_duration_custom", s.Histograms[0].Name)
			},
		},
		{
			name: "custom error formatter",
			timerFn: func(c Collector) Instrument {
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
						Int64Snapshot{
							Name:   "foo_started_total",
							Labels: map[string]string{},
							Value:  1,
						},
						Int64Snapshot{
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
			timerFn: func(c Collector) Instrument {
				return NewInstrument(RootScope(c), "foo")
			},
			in: errMock,
			assert: func(t *testing.T, s Snapshot) {
				assert.Equal(t, 0, len(s.Gauges))
				assert.Equal(
					t,
					[]Int64Snapshot{
						Int64Snapshot{
							Name:   "foo_started_total",
							Labels: map[string]string{},
							Value:  1,
						},
						Int64Snapshot{
							Name:   "foo_total",
							Labels: map[string]string{"status": "mock"},
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
			i := tt.timerFn(c)

			err := i.Exec(func() error { return tt.in })

			assert.Equal(t, tt.in, err)
			tt.assert(t, c.Get())
		})
	}
}
