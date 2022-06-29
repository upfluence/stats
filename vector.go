package stats

import (
	"fmt"
	"sync"
)

type Int64Value struct {
	Tags  map[string]string
	Value int64
}

type Int64VectorGetter interface {
	Labels() []string
	Get() []*Int64Value
}

type atomicInt64Vector struct {
	entityVector
}

func newAtomicInt64Vector(ls []string, lm labelMarshaler) *atomicInt64Vector {
	return &atomicInt64Vector{
		entityVector: entityVector{
			labels:    ls,
			marshaler: lm,
			newFunc:   func(map[string]string) interface{} { return &atomicInt64{} },
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
	var res []*Int64Value

	v.entities.Range(func(k, vv interface{}) bool {
		res = append(
			res,
			&Int64Value{
				Tags:  v.buildTags(k.(uint64)),
				Value: vv.(*atomicInt64).Get(),
			},
		)

		return true
	})

	return res
}

func (v *atomicInt64Vector) fetchValue(ls []string) *atomicInt64 {
	return v.entity(ls).(*atomicInt64)
}

type entityVector struct {
	newFunc func(map[string]string) interface{}

	labels   []string
	entities sync.Map

	marshaler labelMarshaler
}

func (ev *entityVector) entity(ls []string) interface{} {
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
	v, ok := ev.entities.Load(k)

	if ok {
		return v
	}

	vs := make(map[string]string, len(ev.labels))

	for i, k := range ev.labels {
		vs[k] = ls[i]
	}

	v, _ = ev.entities.LoadOrStore(k, ev.newFunc(vs))
	return v
}
