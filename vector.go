package stats

import (
	"fmt"
	"sync"
)

// Int64Value represents a single int64 metric value with its associated tags.
type Int64Value struct {
	Tags  map[string]string
	Value int64
}

// Int64VectorGetter provides read access to int64 vectors for collectors.
// Used by both counters and gauges.
type Int64VectorGetter interface {
	// Labels returns the label names for this vector.
	Labels() []string

	// Get returns all values with their label combinations.
	Get() []*Int64Value
}

type atomicInt64Vector struct {
	entityVector[*atomicInt64]
}

func newAtomicInt64Vector(ls []string, lm *hashingMarshaler) *atomicInt64Vector {
	return &atomicInt64Vector{
		entityVector: entityVector[*atomicInt64]{
			labels:    ls,
			entities:  make(map[uint64]*atomicInt64),
			marshaler: lm,
			newFunc:   func(map[string]string) *atomicInt64 { return &atomicInt64{} },
		},
	}
}

func (v *atomicInt64Vector) Labels() []string { return v.labels }

func (v *atomicInt64Vector) buildTags(key uint64) map[string]string {
	var tags = make(map[string]string, len(v.labels))

	for i, val := range v.marshaler.unmarshal(key, len(v.labels)) {
		tags[v.labels[i]] = val
	}

	return tags
}

func (v *atomicInt64Vector) Get() []*Int64Value {
	type entry struct {
		key   uint64
		value *atomicInt64
	}

	v.mu.RLock()

	var entries = make([]entry, 0, len(v.entities))

	for k, vv := range v.entities {
		entries = append(entries, entry{key: k, value: vv})
	}

	v.mu.RUnlock()

	var res = make([]*Int64Value, 0, len(entries))

	for _, entry := range entries {
		res = append(
			res,
			&Int64Value{
				Tags:  v.buildTags(entry.key),
				Value: entry.value.Get(),
			},
		)
	}

	return res
}

func (v *atomicInt64Vector) fetchValue(ls []string) *atomicInt64 {
	return v.entity(ls)
}

type entityVector[T any] struct {
	newFunc func(map[string]string) T

	labels []string

	mu       sync.RWMutex
	entities map[uint64]T

	marshaler *hashingMarshaler
}

func (ev *entityVector[T]) entity(ls []string) T {
	if len(ls) != len(ev.labels) {
		panic(
			fmt.Sprintf(
				"Not the correct number of labels: labels: %v, values: %v",
				ev.labels,
				ls,
			),
		)
	}

	k := ev.marshaler.marshal(ls)

	ev.mu.RLock()
	v, ok := ev.entities[k]
	ev.mu.RUnlock()

	if ok {
		return v
	}

	vs := make(map[string]string, len(ev.labels))

	for i, k := range ev.labels {
		vs[k] = ls[i]
	}

	v = ev.newFunc(vs)

	ev.mu.Lock()

	if existing, ok := ev.entities[k]; ok {
		v = existing
	} else {
		ev.entities[k] = v
	}

	ev.mu.Unlock()

	return v
}
