package stats

import "testing"

func TestGauge(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mutate     func(Scope)
		introspect func(*testing.T, Snapshot)
	}{
		{
			name:   "simple gauge on root",
			mutate: func(s Scope) { s.Gauge("foo").Update(12) },
			introspect: snapshotEqual(
				Snapshot{
					Gauges: []Int64Snapshot{
						{
							Name:   "foo",
							Labels: map[string]string{},
							Value:  12,
						},
					},
				},
			),
		},
		{
			name: "simple gauge on scope",
			mutate: func(s Scope) {
				s.Scope("bar", map[string]string{"fiz": "buz"}).Gauge("foo").Update(1)
			},
			introspect: snapshotEqual(
				Snapshot{
					Gauges: []Int64Snapshot{
						{
							Name:   "bar_foo",
							Labels: map[string]string{"fiz": "buz"},
							Value:  1,
						},
					},
				},
			),
		},
		{
			name: "labeled gauge on root",
			mutate: func(s Scope) {
				s.GaugeVector("foo", []string{"bar"}).WithLabels("buz").Update(42)
			},
			introspect: snapshotEqual(
				Snapshot{
					Gauges: []Int64Snapshot{
						{
							Name:   "foo",
							Labels: map[string]string{"bar": "buz"},
							Value:  42,
						},
					},
				},
			),
		},
		{
			name: "labeled gauge on scope",
			mutate: func(s Scope) {
				s.Scope("bar", map[string]string{"fiz": "buz"}).GaugeVector(
					"foo",
					[]string{"bar"},
				).WithLabels("bua").Update(1)
			},
			introspect: snapshotEqual(
				Snapshot{
					Gauges: []Int64Snapshot{
						{
							Name:   "bar_foo",
							Labels: map[string]string{"fiz": "buz", "bar": "bua"},
							Value:  1,
						},
					},
				},
			),
		},
		{
			name: "multiple use of labeled gauge",
			mutate: func(s Scope) {
				c := s.GaugeVector("foo", []string{"bar"})

				c.WithLabels("bua").Update(1)
				c.WithLabels("bua").Update(2)
				c.WithLabels("buz").Update(13)
			},
			introspect: snapshotEqual(
				Snapshot{
					Gauges: []Int64Snapshot{
						{
							Name:   "foo",
							Labels: map[string]string{"bar": "bua"},
							Value:  2,
						},
						{
							Name:   "foo",
							Labels: map[string]string{"bar": "buz"},
							Value:  13,
						},
					},
				},
			),
		},
		{
			name: "multi use of the same gauge",
			mutate: func(s Scope) {
				c := s.Gauge("foo")

				c.Update(3)
				c.Update(4)
			},
			introspect: snapshotEqual(
				Snapshot{
					Gauges: []Int64Snapshot{
						{
							Name:   "foo",
							Labels: map[string]string{},
							Value:  4,
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

func BenchmarkGaugeInc(b *testing.B) {
	c := RootScope(NewStaticCollector()).Gauge("foo")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Update(37)
	}
}

func BenchmarkVectorGauge(b *testing.B) {
	c := RootScope(NewStaticCollector()).GaugeVector(
		"foo",
		[]string{"bar", "buz"},
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.WithLabels("foo", "bar").Update(37)
	}
}
