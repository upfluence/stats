package expvar

import (
	"expvar"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upfluence/stats"
)

func TestPublish(t *testing.T) {
	for _, tt := range []struct {
		name     string
		mutate   func(stats.Scope)
		asserMap func(*testing.T, map[string]string)
	}{
		{
			name:   "no stats",
			mutate: func(stats.Scope) {},
			asserMap: func(t *testing.T, res map[string]string) {
				assert.Equal(t, 0, len(res), "%v", res)
			},
		},
		{
			name: "simple counter",
			mutate: func(s stats.Scope) {
				s.Counter("foo").Add(37)
			},
			asserMap: func(t *testing.T, res map[string]string) {
				assert.Equal(
					t,
					"{\"Type\":\"counter\",\"Value\":[{\"Tags\":{},\"Value\":37}]}\n",
					res["foo"],
				)
			},
		},
		{
			name: "simple gauge",
			mutate: func(s stats.Scope) {
				s.Gauge("bar").Update(37)
			},
			asserMap: func(t *testing.T, res map[string]string) {
				assert.Equal(
					t,
					"{\"Type\":\"gauge\",\"Value\":[{\"Tags\":{},\"Value\":37}]}\n",
					res["bar"],
				)
			},
		},
		{
			name: "simple histogram",
			mutate: func(s stats.Scope) {
				s.Histogram("buz").Record(.37)
			},
			asserMap: func(t *testing.T, res map[string]string) {
				assert.Equal(
					t,
					"{\"Type\":\"histogram\",\"Value\":[{\"Tags\":{},\"Count\":1,\"Sum\":0.37,\"Buckets\":[{\"Count\":0,\"UpperBound\":0.005},{\"Count\":0,\"UpperBound\":0.01},{\"Count\":0,\"UpperBound\":0.025},{\"Count\":0,\"UpperBound\":0.05},{\"Count\":0,\"UpperBound\":0.1},{\"Count\":0,\"UpperBound\":0.25},{\"Count\":1,\"UpperBound\":0.5},{\"Count\":0,\"UpperBound\":1},{\"Count\":0,\"UpperBound\":2.5},{\"Count\":0,\"UpperBound\":5},{\"Count\":0,\"UpperBound\":10},{\"Count\":0,\"UpperBound\":0}]}]}\n",
					res["buz"],
				)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.mutate(stats.RootScope(NewCollector()))

			var expvarMap = make(map[string]string)

			expvar.Do(func(kv expvar.KeyValue) {
				if kv.Key == "memstats" || kv.Key == "cmdline" {
					return
				}
				expvarMap[kv.Key] = kv.Value.String()
			})

			tt.asserMap(t, expvarMap)
		})
	}
}
