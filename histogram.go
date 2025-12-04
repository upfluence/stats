package stats

import (
	"fmt"
	"math"
	"sync"
)

var defaultCutoffs = []float64{
	.005,
	.01,
	.025,
	.05,
	.1,
	.25,
	.5,
	1.,
	2.5,
	5.,
	10.,
	math.Inf(0),
}

// HistogramValue represents a snapshot of a histogram's state including
// its tags, total count, sum, and bucket counts.
type HistogramValue struct {
	Tags map[string]string

	Count   int64
	Sum     float64
	Buckets []Bucket
}

// HistogramVectorGetter provides read access to histogram vectors for collectors.
type HistogramVectorGetter interface {
	// Labels returns the label names for this histogram vector.
	Labels() []string

	// Cutoffs returns the upper bounds for each bucket.
	Cutoffs() []float64

	// Get returns all histogram values with their label combinations.
	Get() []*HistogramValue
}

// HistogramVector is a multi-dimensional histogram that creates histogram instances
// with specific label values.
type HistogramVector interface {
	// WithLabels returns a Histogram with the specified label values.
	// The number of values must match the number of labels defined for this vector.
	WithLabels(...string) Histogram
}

type histogramVector struct {
	labels  []string
	cutoffs []float64

	mu sync.RWMutex
	hs map[uint64]*histogram

	marshaler labelMarshaler
}

func (hv *histogramVector) Labels() []string   { return hv.labels }
func (hv *histogramVector) Cutoffs() []float64 { return hv.cutoffs }

func (hv *histogramVector) buildTags(key uint64) map[string]string {
	var tags = make(map[string]string, len(hv.labels))

	for i, val := range hv.marshaler.unmarshal(key, len(hv.labels)) {
		tags[hv.labels[i]] = val
	}

	return tags
}

func (hv *histogramVector) Get() []*HistogramValue {
	var res = make([]*HistogramValue, 0, len(hv.hs))

	for k, h := range hv.hs {
		res = append(
			res,
			&HistogramValue{
				Tags:    hv.buildTags(k),
				Count:   h.Count(),
				Sum:     h.Sum(),
				Buckets: h.Buckets(),
			},
		)
	}

	return res
}

func (hv *histogramVector) WithLabels(ls ...string) Histogram {
	if len(ls) != len(hv.labels) {
		panic(
			fmt.Sprintf(
				"Not the correct number of labels: labels: %v, values: %v",
				hv.labels,
				ls,
			),
		)
	}

	k := hv.marshaler.marshal(ls)

	hv.mu.RLock()
	h, ok := hv.hs[k]
	hv.mu.RUnlock()

	if ok {
		return h
	}

	hv.mu.Lock()

	h = &histogram{
		cutoffs: hv.cutoffs,
		counts:  make([]atomicInt64, len(hv.cutoffs)),
	}

	hv.hs[k] = h

	hv.mu.Unlock()
	return h
}

// Bucket represents a single histogram bucket with its count and upper bound.
type Bucket struct {
	Count      int64
	UpperBound float64
}

// Histogram tracks the distribution of values across predefined buckets.
// Histograms are useful for measuring request durations, response sizes, etc.
type Histogram interface {
	// Record adds a single observation to the histogram.
	Record(float64)

	// Count returns the total number of observations.
	Count() int64

	// Sum returns the sum of all observed values.
	Sum() float64

	// Buckets returns the current state of all buckets.
	Buckets() []Bucket
}

type histogram struct {
	cutoffs []float64

	sum    atomicFloat64
	counts []atomicInt64
}

func (h *histogram) Record(v float64) {
	for i, c := range h.cutoffs {
		if v <= c {
			h.counts[i].Inc()
			h.sum.Add(v)
			break
		}
	}
}

func (h *histogram) Sum() float64 { return h.sum.Get() }

func (h *histogram) Count() int64 {
	var res int64

	for _, c := range h.counts {
		res += c.Get()
	}

	return res
}

func (h *histogram) Buckets() []Bucket {
	var bs = make([]Bucket, len(h.cutoffs))

	for i, cutoff := range h.cutoffs {
		bs[i].UpperBound = cutoff
		bs[i].Count = h.counts[i].Get()
	}

	return bs
}

// StaticBuckets creates a HistogramOption that configures custom bucket boundaries.
// An infinity bucket is automatically appended to catch all values above the highest cutoff.
func StaticBuckets(cutoffs []float64) HistogramOption {
	return func(hv *histogramVector) {
		hv.cutoffs = append(cutoffs, math.Inf(0))
	}
}

type partialHistogramVector struct {
	hv HistogramVector
	vs []string
}

func (phv partialHistogramVector) WithLabels(labels ...string) Histogram {
	return phv.hv.WithLabels(append(phv.vs, labels...)...)
}

type reorderHistogramVector struct {
	hv HistogramVector
	labelOrderer
}

func (rhv reorderHistogramVector) WithLabels(ls ...string) Histogram {
	return rhv.hv.WithLabels(rhv.order(ls)...)
}

var (
	// NoopHistogram is a histogram that discards all operations.
	NoopHistogram Histogram = noopHistogram{}

	// NoopHistogramVector is a histogram vector that returns noop histograms.
	NoopHistogramVector HistogramVector = noopHistogramVector{}
)

type noopHistogram struct{}

func (noopHistogram) Record(float64)    {}
func (noopHistogram) Count() int64      { return 0 }
func (noopHistogram) Sum() float64      { return 0 }
func (noopHistogram) Buckets() []Bucket { return nil }

type noopHistogramVector struct{}

func (noopHistogramVector) WithLabels(...string) Histogram {
	return noopHistogram{}
}
