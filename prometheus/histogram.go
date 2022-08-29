package prometheus

import (
	"fmt"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/proto"

	"github.com/upfluence/stats"
)

type multiHistogramVectorGetter struct {
	gs []stats.HistogramVectorGetter
}

func (mhvg *multiHistogramVectorGetter) appendGetter(n string, g stats.HistogramVectorGetter) {
	if len(mhvg.gs) > 0 {
		if hashSlice(g.Labels()) != hashSlice(mhvg.gs[0].Labels()) {
			panic(
				fmt.Sprintf(
					"%s: histogram cannot be registered because the labels are different, has: %v, want %v",
					n,
					g.Labels(),
					mhvg.gs[0].Labels(),
				),
			)
		}

		if hashFloat64Slice(g.Cutoffs()) != hashFloat64Slice(mhvg.gs[0].Cutoffs()) {
			panic(
				fmt.Sprintf(
					"%s: histogram cannot be registered because the cutoffs are different, has: %v, want %v",
					n,
					g.Cutoffs(),
					mhvg.gs[0].Cutoffs(),
				),
			)
		}
	}

	mhvg.gs = append(mhvg.gs, g)
}

func (mhvg *multiHistogramVectorGetter) Labels() []string {
	if len(mhvg.gs) == 0 {
		return nil
	}

	return mhvg.gs[0].Labels()
}

func (mhvg *multiHistogramVectorGetter) Cutoffs() []float64 {
	if len(mhvg.gs) == 0 {
		return nil
	}

	return mhvg.gs[0].Cutoffs()
}

func (mhvg *multiHistogramVectorGetter) Get() []*stats.HistogramValue {
	switch len(mhvg.gs) {
	case 0:
		return nil
	case 1:
		return mhvg.gs[0].Get()
	}

	var (
		tags    = make(map[uint64]map[string]string)
		counts  = make(map[uint64]int64)
		sums    = make(map[uint64]float64)
		buckets = make(map[uint64]map[float64]int64)
	)

	for _, g := range mhvg.gs {
		for _, hv := range g.Get() {
			key := hashTags(hv.Tags)

			if _, ok := tags[key]; !ok {
				tags[key] = hv.Tags
				buckets[key] = make(map[float64]int64, len(hv.Buckets))
			}

			counts[key] += hv.Count
			sums[key] += hv.Sum

			for _, b := range hv.Buckets {
				buckets[key][b.UpperBound] += b.Count
			}
		}
	}

	res := make([]*stats.HistogramValue, 0, len(tags))

	for key, ts := range tags {
		hv := stats.HistogramValue{
			Tags:    ts,
			Count:   counts[key],
			Sum:     sums[key],
			Buckets: make([]stats.Bucket, 0, len(buckets[key])),
		}

		for ub, count := range buckets[key] {
			hv.Buckets = append(
				hv.Buckets,
				stats.Bucket{UpperBound: ub, Count: count},
			)
		}

		sort.Slice(
			hv.Buckets,
			func(i, j int) bool {
				return hv.Buckets[i].UpperBound < hv.Buckets[j].UpperBound
			},
		)

		res = append(res, &hv)
	}

	return res
}

type histogramWrapper struct {
	g stats.HistogramVectorGetter
	n string

	desc *prometheus.Desc
}

func (hw *histogramWrapper) Describe(ch chan<- *prometheus.Desc) {
	ch <- hw.desc
}

func (hw *histogramWrapper) Collect(ch chan<- prometheus.Metric) {
	for _, v := range hw.g.Get() {
		ch <- &histogramMetric{desc: hw.desc, v: v}
	}
}

type histogramMetric struct {
	desc *prometheus.Desc
	v    *stats.HistogramValue
}

func (hm *histogramMetric) Desc() *prometheus.Desc {
	return hm.desc
}

func (hm *histogramMetric) Write(m *dto.Metric) error {
	var ps []*dto.LabelPair

	for k, v := range hm.v.Tags {
		k, v := k, v
		ps = append(ps, &dto.LabelPair{Name: &k, Value: &v})
	}

	m.Histogram = &dto.Histogram{
		SampleCount: proto.Uint64(uint64(hm.v.Count)),
		SampleSum:   proto.Float64(hm.v.Sum),
	}

	var sum int64

	for _, b := range hm.v.Buckets {
		sum += b.Count

		m.Histogram.Bucket = append(
			m.Histogram.Bucket,
			&dto.Bucket{
				CumulativeCount: proto.Uint64(uint64(sum)),
				UpperBound:      proto.Float64(b.UpperBound),
			},
		)
	}

	m.Label = ps

	return nil
}
