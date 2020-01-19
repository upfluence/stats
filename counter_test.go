package stats

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func snapshotEqual(expected Snapshot) func(*testing.T, Snapshot) {
	return func(t *testing.T, actual Snapshot) {
		t.Helper()
		assert.Equal(t, expected, actual)
	}
}

func TestCounter(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mutate     func(Scope)
		introspect func(*testing.T, Snapshot)
	}{
		{
			name:       "no mutation",
			mutate:     func(Scope) {},
			introspect: snapshotEqual(Snapshot{}),
		},
		{
			name:   "simple counter on root",
			mutate: func(s Scope) { s.Counter("foo").Add(42) },
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
						{
							Name:   "foo",
							Labels: map[string]string{},
							Value:  42,
						},
					},
				},
			),
		},
		{
			name: "simple counter on scope",
			mutate: func(s Scope) {
				s.Scope("bar", map[string]string{"fiz": "buz"}).Counter("foo").Inc()
			},
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
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
			name: "simple counter on scope and root scope",
			mutate: func(s Scope) {
				s.Scope(
					"bar",
					map[string]string{"fiz": "buz"},
				).RootScope().Counter("foo").Inc()
			},
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
						{Name: "foo", Labels: map[string]string{}, Value: 1},
					},
				},
			),
		},
		{
			name: "labeled counter on root",
			mutate: func(s Scope) {
				s.CounterVector("foo", []string{"bar"}).WithLabels("buz").Add(42)
			},
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
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
			name: "labeled counter on scope",
			mutate: func(s Scope) {
				s.Scope("bar", map[string]string{"fiz": "buz"}).CounterVector(
					"foo",
					[]string{"bar"},
				).WithLabels("bua").Inc()
			},
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
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
			name: "multiple use of labeled counter",
			mutate: func(s Scope) {
				c := s.CounterVector("foo", []string{"bar"})

				c.WithLabels("bua").Inc()
				c.WithLabels("bua").Inc()
				c.WithLabels("buz").Inc()
			},
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
						{
							Name:   "foo",
							Labels: map[string]string{"bar": "buz"},
							Value:  1,
						},
						{
							Name:   "foo",
							Labels: map[string]string{"bar": "bua"},
							Value:  2,
						},
					},
				},
			),
		},
		{
			name: "multiple instance and matching labels",
			mutate: func(s Scope) {
				s.CounterVector(
					"foo",
					[]string{"foo", "bar"},
				).WithLabels("fiz", "buz").Inc()

				s.CounterVector(
					"foo",
					[]string{"bar", "foo"},
				).WithLabels("buz", "fiz").Inc()
			},
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
						{
							Name: "foo",
							Labels: map[string]string{
								"bar": "buz",
								"foo": "fiz",
							},
							Value: 2,
						},
					},
				},
			),
		},
		{
			name: "multiple instance of the same counter adds up",
			mutate: func(s Scope) {
				s.Counter("foo").Inc()
				s.Counter("foo").Inc()
			},
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
						{
							Name:   "foo",
							Labels: map[string]string{},
							Value:  2,
						},
					},
				},
			),
		},
		{
			name: "multi use of the same counter",
			mutate: func(s Scope) {
				c := s.Counter("foo")

				c.Inc()
				c.Inc()
			},
			introspect: snapshotEqual(
				Snapshot{
					Counters: []Int64Snapshot{
						{
							Name:   "foo",
							Labels: map[string]string{},
							Value:  2,
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

func TestConcurrentCounterVector(t *testing.T) {
	s := RootScope(NewStaticCollector())

	for i := 0; i < 100; i++ {
		i := i
		go func() {
			cv := s.CounterVector("foo", []string{"bar"})
			cv.WithLabels(strconv.Itoa(i)).Inc()
		}()
	}
}

func BenchmarkCounterDeclare(b *testing.B) {
	s := RootScope(NewStaticCollector())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Counter("foo").Inc()
	}
}

func BenchmarkCounterInc(b *testing.B) {
	c := RootScope(NewStaticCollector()).Counter("foo")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

func BenchmarkVectorCounter(b *testing.B) {
	c := RootScope(NewStaticCollector()).CounterVector(
		"foo",
		[]string{"bar", "buz"},
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.WithLabels("foo", "bar").Inc()
	}
}
