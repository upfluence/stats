package stats

import "strings"

type limitedScope interface {
	namespace() string
	tags() map[string]string

	collector() Collector
}

type Scope interface {
	limitedScope

	Counter(string) Counter
	CounterVector(string, []string) CounterVector

	Gauge(string) Gauge
	GaugeVector(string, []string) GaugeVector

	Scope(string, map[string]string) Scope
}

func RootScope(c Collector) Scope {
	return scopeWrapper{&rootScope{c: c}}
}

type scopeWrapper struct {
	limitedScope
}

type scopedInt64VectorGetter struct {
	s limitedScope
	g Int64VectorGetter
}

func (slcg *scopedInt64VectorGetter) Labels() []string {
	var ls = slcg.g.Labels()

	for k := range slcg.s.tags() {
		ls = append(ls, k)
	}

	return ls
}

func (slcg *scopedInt64VectorGetter) Get() []*Int64Value {
	var res = slcg.g.Get()

	for _, cv := range res {
		for k, v := range slcg.s.tags() {
			cv.Tags[k] = v
		}
	}

	return res
}

func (sw scopeWrapper) Gauge(name string) Gauge {
	var c = &atomicInt64{}

	sw.collector().RegisterGauge(
		joinStrings(sw.namespace(), name),
		&scopedInt64VectorGetter{
			s: sw.limitedScope,
			g: &atomicInt64Vector{
				cs:        map[string]*atomicInt64{"": c},
				marshaler: defaultLabelMarshaler{},
			},
		},
	)

	return c
}

func (sw scopeWrapper) GaugeVector(name string, labels []string) GaugeVector {
	var lc = &atomicInt64Vector{
		labels:    labels,
		cs:        make(map[string]*atomicInt64),
		marshaler: defaultLabelMarshaler{},
	}

	sw.collector().RegisterGauge(
		joinStrings(sw.namespace(), name),
		&scopedInt64VectorGetter{s: sw.limitedScope, g: lc},
	)

	return &gaugeVector{atomicInt64Vector: lc}
}

func (sw scopeWrapper) Counter(name string) Counter {
	var c = &atomicInt64{}

	sw.collector().RegisterCounter(
		joinStrings(sw.namespace(), name),
		&scopedInt64VectorGetter{
			s: sw.limitedScope,
			g: &atomicInt64Vector{
				cs:        map[string]*atomicInt64{"": c},
				marshaler: defaultLabelMarshaler{},
			},
		},
	)

	return c
}

func (sw scopeWrapper) CounterVector(name string, labels []string) CounterVector {
	var lc = &atomicInt64Vector{
		labels:    labels,
		cs:        make(map[string]*atomicInt64),
		marshaler: defaultLabelMarshaler{},
	}

	sw.collector().RegisterCounter(
		joinStrings(sw.namespace(), name),
		&scopedInt64VectorGetter{s: sw.limitedScope, g: lc},
	)

	return &counterVector{atomicInt64Vector: lc}
}

func (sw scopeWrapper) Scope(ns string, tags map[string]string) Scope {
	return scopeWrapper{
		limitedScope: &subScope{parent: sw.limitedScope, ns: ns, ts: tags},
	}
}

type rootScope struct {
	c Collector
}

func (*rootScope) namespace() string       { return "" }
func (*rootScope) tags() map[string]string { return nil }
func (rs *rootScope) collector() Collector { return rs.c }

type subScope struct {
	parent limitedScope

	ns string
	ts map[string]string
}

func (ss *subScope) namespace() string {
	return joinStrings(ss.parent.namespace(), ss.ns)
}

func (ss *subScope) tags() map[string]string {
	return mergeStringMaps(ss.parent.tags(), ss.ts)
}

func (ss *subScope) collector() Collector { return ss.parent.collector() }

func joinStrings(ss ...string) string {
	var (
		res []string

		append = func(s string) {
			if s != "" {
				res = append(res, s)
			}
		}
	)

	for _, s := range ss {
		append(s)
	}

	return strings.Join(res, "_")
}

func mergeStringMaps(kvs ...map[string]string) map[string]string {
	var (
		res = make(map[string]string)

		merge = func(kv map[string]string) {
			for k, v := range kv {
				res[k] = v
			}
		}
	)

	for _, kv := range kvs {
		merge(kv)
	}

	return res
}
