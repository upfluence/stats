package stats

import (
	"math"
	"testing"
)

func buildBuckets(cutoffs []float64, vs ...float64) []Bucket {
	var bs []Bucket

	for _, c := range cutoffs {
		b := Bucket{UpperBound: c}

		bs = append(bs, b)
	}

	for _, v := range vs {
		for i, c := range cutoffs {
			if v <= c {
				bs[i].Count++
				break
			}
		}
	}

	return bs
}

func TestHistogram(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mutate     func(Scope)
		introspect func(*testing.T, Snapshot)
	}{
		{
			name:   "simple histogram on root",
			mutate: func(s Scope) { s.Histogram("foo").Record(.012) },
			introspect: snapshotEqual(
				Snapshot{
					Histograms: []HistogramSnapshot{
						{
							Name: "foo",
							Value: HistogramValue{
								Tags:    map[string]string{},
								Count:   1,
								Sum:     .012,
								Buckets: buildBuckets(defaultCutoffs, .012),
							},
						},
					},
				},
			),
		},
		{
			name: "multiple histogram with the same name",
			mutate: func(s Scope) {
				s.Histogram("foo").Record(0.012)
				s.Histogram("foo").Record(0.018)
			},
			introspect: snapshotEqual(
				Snapshot{
					Histograms: []HistogramSnapshot{
						{
							Name: "foo",
							Value: HistogramValue{
								Tags:    map[string]string{},
								Count:   2,
								Sum:     .030,
								Buckets: buildBuckets(defaultCutoffs, .012, .018),
							},
						},
					},
				},
			),
		},
		{
			name: "simple histogram on scope",
			mutate: func(s Scope) {
				s.Scope("bar", map[string]string{"fiz": "buz"}).Histogram("foo").Record(.001)
			},
			introspect: snapshotEqual(
				Snapshot{
					Histograms: []HistogramSnapshot{
						{
							Name: "bar_foo",
							Value: HistogramValue{
								Tags:    map[string]string{"fiz": "buz"},
								Count:   1,
								Sum:     .001,
								Buckets: buildBuckets(defaultCutoffs, .001),
							},
						},
					},
				},
			),
		},
		{
			name: "vector histogram  on root",
			mutate: func(s Scope) {
				s.HistogramVector("foo", []string{"bar"}).WithLabels("buz").Record(.042)
			},
			introspect: snapshotEqual(
				Snapshot{
					Histograms: []HistogramSnapshot{
						{
							Name: "foo",
							Value: HistogramValue{
								Tags:    map[string]string{"bar": "buz"},
								Count:   1,
								Sum:     .042,
								Buckets: buildBuckets(defaultCutoffs, .042),
							},
						},
					},
				},
			),
		},
		{
			name: "vector histogram on scope",
			mutate: func(s Scope) {
				s.Scope("bar", map[string]string{"biz": "baa"}).HistogramVector(
					"foo",
					[]string{"bar"},
				).WithLabels("buz").Record(.042)
			},
			introspect: snapshotEqual(
				Snapshot{
					Histograms: []HistogramSnapshot{
						{
							Name: "bar_foo",
							Value: HistogramValue{
								Tags:    map[string]string{"biz": "baa", "bar": "buz"},
								Count:   1,
								Sum:     .042,
								Buckets: buildBuckets(defaultCutoffs, .042),
							},
						},
					},
				},
			),
		},
		{
			name: "histogram with custom cutoffs",
			mutate: func(s Scope) {
				s.HistogramVector(
					"foo",
					[]string{"bar"},
					StaticBuckets([]float64{.005, .01}),
				).WithLabels("buz").Record(.042)
			},
			introspect: snapshotEqual(
				Snapshot{
					Histograms: []HistogramSnapshot{
						{
							Name: "foo",
							Value: HistogramValue{
								Tags:    map[string]string{"bar": "buz"},
								Count:   1,
								Sum:     .042,
								Buckets: buildBuckets([]float64{.005, .01, math.Inf(0)}, .042),
							},
						},
					},
				},
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := NewStaticCollector()

			tt.mutate(RootScope(c))
			tt.introspect(t, c.Get())
		})
	}
}

func BenchmarkHistogramInc(b *testing.B) {
	c := RootScope(NewStaticCollector()).Histogram("foo")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Record(.5)
	}
}

func BenchmarkVectorHistogram(b *testing.B) {
	c := RootScope(NewStaticCollector()).HistogramVector(
		"foo",
		[]string{"bar", "buz"},
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.WithLabels("foo", "bar").Record(.5)
	}
}
