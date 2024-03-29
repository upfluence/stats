package prometheus

import (
	"math"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/upfluence/stats"
)

func TestCollect(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mutate     func(stats.Scope)
		introspect func(*testing.T, []*dto.MetricFamily)
	}{
		{
			name:   "no mutation",
			mutate: func(stats.Scope) {},
			introspect: func(t *testing.T, fs []*dto.MetricFamily) {
				assert.Equal(t, 0, len(fs))
			},
		},
		{
			name: "one histogram",
			mutate: func(s stats.Scope) {
				s.Histogram("foo", stats.StaticBuckets([]float64{1., 10.})).Record(5.)
			},
			introspect: func(t *testing.T, fs []*dto.MetricFamily) {
				mt := dto.MetricType_HISTOGRAM
				assert.Equal(
					t,
					[]*dto.MetricFamily{
						&dto.MetricFamily{
							Name: proto.String("foo"),
							Help: proto.String("no help"),
							Type: &mt,
							Metric: []*dto.Metric{
								&dto.Metric{
									Histogram: &dto.Histogram{
										SampleCount: proto.Uint64(1),
										SampleSum:   proto.Float64(5.),
										Bucket: []*dto.Bucket{
											&dto.Bucket{
												CumulativeCount: proto.Uint64(0),
												UpperBound:      proto.Float64(1.),
											},
											&dto.Bucket{
												CumulativeCount: proto.Uint64(1),
												UpperBound:      proto.Float64(10.),
											},
											&dto.Bucket{
												CumulativeCount: proto.Uint64(1),
												UpperBound:      proto.Float64(math.Inf(0)),
											},
										},
									},
								},
							},
						},
					},
					fs,
				)
			},
		},
		{
			name:   "one gauge",
			mutate: func(s stats.Scope) { s.Gauge("foo").Update(14) },
			introspect: func(t *testing.T, fs []*dto.MetricFamily) {
				mt := dto.MetricType_GAUGE
				assert.Equal(
					t,
					[]*dto.MetricFamily{
						&dto.MetricFamily{
							Name: proto.String("foo"),
							Help: proto.String("no help"),
							Type: &mt,
							Metric: []*dto.Metric{
								&dto.Metric{Gauge: &dto.Gauge{Value: proto.Float64(14)}},
							},
						},
					},
					fs,
				)
			},
		},
		{
			name:   "one counter",
			mutate: func(s stats.Scope) { s.Counter("foo").Inc() },
			introspect: func(t *testing.T, fs []*dto.MetricFamily) {
				mt := dto.MetricType_COUNTER
				assert.Equal(
					t,
					[]*dto.MetricFamily{
						&dto.MetricFamily{
							Name: proto.String("foo"),
							Help: proto.String("no help"),
							Type: &mt,
							Metric: []*dto.Metric{
								&dto.Metric{Counter: &dto.Counter{Value: proto.Float64(1)}},
							},
						},
					},
					fs,
				)
			},
		},
		{
			name: "one labeled counter",
			mutate: func(s stats.Scope) {
				s.CounterVector("foo", []string{"bar"}).WithLabels("buz").Inc()
			},
			introspect: func(t *testing.T, fs []*dto.MetricFamily) {
				mt := dto.MetricType_COUNTER
				assert.Equal(
					t,
					[]*dto.MetricFamily{
						&dto.MetricFamily{
							Name: proto.String("foo"),
							Help: proto.String("no help"),
							Type: &mt,
							Metric: []*dto.Metric{
								&dto.Metric{
									Label: []*dto.LabelPair{
										&dto.LabelPair{
											Name:  proto.String("bar"),
											Value: proto.String("buz"),
										},
									},
									Counter: &dto.Counter{Value: proto.Float64(1)},
								},
							},
						},
					},
					fs,
				)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := prometheus.NewRegistry()

			s := stats.RootScope(NewCollector(r))
			tt.mutate(s)

			fs, err := r.Gather()
			assert.Nil(t, err)

			tt.introspect(t, fs)
		})
	}
}

func TestMultiRegisterGauge(t *testing.T) {
	r := prometheus.NewRegistry()
	c := NewCollector(r)

	for i := 0; i < 5; i++ {
		s := stats.RootScope(c)
		s.GaugeVector("foo", []string{"bar"}).WithLabels("buz").Update(1.)
	}

	fs, err := r.Gather()
	assert.Nil(t, err)
	mt := dto.MetricType_GAUGE
	assert.Equal(
		t,
		[]*dto.MetricFamily{
			&dto.MetricFamily{
				Name: proto.String("foo"),
				Help: proto.String("no help"),
				Type: &mt,
				Metric: []*dto.Metric{
					&dto.Metric{
						Label: []*dto.LabelPair{
							&dto.LabelPair{
								Name:  proto.String("bar"),
								Value: proto.String("buz"),
							},
						},
						Gauge: &dto.Gauge{Value: proto.Float64(1)},
					},
				},
			},
		},
		fs,
	)
}

func TestMultiRegisterCounter(t *testing.T) {
	r := prometheus.NewRegistry()
	c := NewCollector(r)

	for i := 0; i < 5; i++ {
		s := stats.RootScope(c)
		s.CounterVector("foo", []string{"bar"}).WithLabels("buz").Inc()
	}

	fs, err := r.Gather()
	assert.Nil(t, err)
	mt := dto.MetricType_COUNTER
	assert.Equal(
		t,
		[]*dto.MetricFamily{
			&dto.MetricFamily{
				Name: proto.String("foo"),
				Help: proto.String("no help"),
				Type: &mt,
				Metric: []*dto.Metric{
					&dto.Metric{
						Label: []*dto.LabelPair{
							&dto.LabelPair{
								Name:  proto.String("bar"),
								Value: proto.String("buz"),
							},
						},
						Counter: &dto.Counter{Value: proto.Float64(5)},
					},
				},
			},
		},
		fs,
	)
}

func TestMultiRegisterHistogram(t *testing.T) {
	r := prometheus.NewRegistry()
	c := NewCollector(r)

	for i := 0; i < 5; i++ {
		s := stats.RootScope(c)
		s.Histogram("foo", stats.StaticBuckets([]float64{1., 10.})).Record(5.)
	}

	fs, err := r.Gather()
	assert.Nil(t, err)
	mt := dto.MetricType_HISTOGRAM
	assert.Equal(
		t,
		[]*dto.MetricFamily{
			&dto.MetricFamily{
				Name: proto.String("foo"),
				Help: proto.String("no help"),
				Type: &mt,
				Metric: []*dto.Metric{
					&dto.Metric{
						Histogram: &dto.Histogram{
							SampleCount: proto.Uint64(5),
							SampleSum:   proto.Float64(25.),
							Bucket: []*dto.Bucket{
								&dto.Bucket{
									CumulativeCount: proto.Uint64(0),
									UpperBound:      proto.Float64(1.),
								},
								&dto.Bucket{
									CumulativeCount: proto.Uint64(5),
									UpperBound:      proto.Float64(10.),
								},
								&dto.Bucket{
									CumulativeCount: proto.Uint64(5),
									UpperBound:      proto.Float64(math.Inf(0)),
								},
							},
						},
					},
				},
			},
		},
		fs,
	)
}
