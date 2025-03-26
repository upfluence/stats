package stats

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/upfluence/stats/internal/hash"
)

var globalIncarnationRegistry = &incarnationRegistry{key: "incarnation", counters: make(map[counterKey]uint)}

type incarnationRegistry struct {
	countersMu sync.Mutex
	counters   map[counterKey]uint
	key        string
}

func (ir *incarnationRegistry) next(ck counterKey) uint {
	ir.countersMu.Lock()
	defer ir.countersMu.Unlock()

	current, ok := ir.counters[ck]

	next := current + 1

	if !ok {
		next--
	}

	ir.counters[ck] = next

	return next
}

type counterKey struct {
	keySum  uint64
	tagsSum uint64
}

func newCounterKey() counterKey {
	return counterKey{keySum: hash.New()}
}

func (ck counterKey) add(key string, tags map[string]string) counterKey {
	if key != "" {
		if ck.keySum != hash.New() {
			key = "_" + key
		}

		ck.keySum = hash.AddNoScramble(ck.keySum, key)
	}

	for k, v := range tags {
		ck.tagsSum |= hash.Add(hash.Add(hash.New(), k), v)
	}

	return ck
}

func (ck counterKey) appendTag(k, v string) counterKey {
	ck.tagsSum |= hash.Add(hash.Add(hash.New(), k), v)

	return ck
}

type multiIncarnationScope struct {
	scope Scope

	currentKey counterKey
	registry   *incarnationRegistry
}

func newMultiIncarnationScope(sc Scope, r *incarnationRegistry) *multiIncarnationScope {
	return &multiIncarnationScope{
		scope:      sc,
		currentKey: newCounterKey().add(sc.namespace(), sc.tags()),
		registry:   r,
	}
}

func GlobalIncarnationScope(sc Scope) Scope {
	return newMultiIncarnationScope(sc, globalIncarnationRegistry)
}

func LocalIncarnationScope(sc Scope, k string) Scope {
	return newMultiIncarnationScope(
		sc,
		&incarnationRegistry{key: k, counters: make(map[counterKey]uint)},
	)
}

func (mis *multiIncarnationScope) namespace() string       { return mis.scope.namespace() }
func (mis *multiIncarnationScope) tags() map[string]string { return mis.scope.tags() }
func (mis *multiIncarnationScope) rootScope() *rootScope   { return mis.scope.rootScope() }

func (mis *multiIncarnationScope) Scope(k string, vs map[string]string) Scope {
	if _, ok := vs[mis.registry.key]; ok {
		panic(fmt.Sprintf("scope can not include the incarnation key %q", mis.registry.key))
	}

	return &multiIncarnationScope{
		scope:      mis.scope.Scope(k, vs),
		currentKey: mis.currentKey.add(k, vs),
		registry:   mis.registry,
	}
}

func (mis *multiIncarnationScope) RootScope() Scope {
	return &multiIncarnationScope{
		scope:    mis.scope.RootScope(),
		registry: mis.registry,
	}
}

type abstractVector[T any] interface {
	WithLabels(...string) T
}

type multiIncarnationVector[T any] struct {
	cv abstractVector[T]
	ls []string

	currentKey counterKey
	registry   *incarnationRegistry
}

func (micv *multiIncarnationVector[T]) WithLabels(vs ...string) T {
	if len(vs) != len(micv.ls) {
		panic("wrong number of label values")
	}

	ck := micv.currentKey

	for i, l := range micv.ls {
		ck = ck.appendTag(l, vs[i])
	}

	return micv.cv.WithLabels(
		append(
			vs,
			strconv.Itoa(int(micv.registry.next(ck))),
		)...,
	)
}

func (mis *multiIncarnationScope) Counter(k string) Counter {
	return mis.CounterVector(k, nil).WithLabels()
}

func (mis *multiIncarnationScope) CounterVector(k string, ls []string) CounterVector {
	return &multiIncarnationVector[Counter]{
		cv:         mis.scope.CounterVector(k, append(ls, mis.registry.key)),
		ls:         ls,
		currentKey: mis.currentKey.add(k, nil),
		registry:   mis.registry,
	}
}

func (mis *multiIncarnationScope) Gauge(k string) Gauge {
	return mis.GaugeVector(k, nil).WithLabels()
}

func (mis *multiIncarnationScope) GaugeVector(k string, ls []string) GaugeVector {
	return &multiIncarnationVector[Gauge]{
		cv:         mis.scope.GaugeVector(k, append(ls, mis.registry.key)),
		ls:         ls,
		currentKey: mis.currentKey.add(k, nil),
		registry:   mis.registry,
	}
}

func (mis *multiIncarnationScope) Histogram(k string, opts ...HistogramOption) Histogram {
	return mis.HistogramVector(k, nil, opts...).WithLabels()
}

func (mis *multiIncarnationScope) HistogramVector(k string, ls []string, opts ...HistogramOption) HistogramVector {
	return &multiIncarnationVector[Histogram]{
		cv:         mis.scope.HistogramVector(k, append(ls, mis.registry.key), opts...),
		ls:         ls,
		currentKey: mis.currentKey.add(k, nil),
		registry:   mis.registry,
	}
}
