package prometheus

import (
	"strconv"

	"github.com/upfluence/stats/internal/hash"
)

func hashTags(tags map[string]string) uint64 {
	var res uint64

	for key, val := range tags {
		v := hash.New()

		v = hash.Add(v, key)
		v = hash.Add(v, val)

		res ^= v
	}

	return res
}

func hashSlice(vs []string) uint64 {
	var res uint64

	for _, val := range vs {
		res ^= hash.Add(hash.New(), val)
	}

	return res
}

func hashFloat64Slice(vs []float64) uint64 {
	var res uint64

	for _, val := range vs {
		res ^= hash.Add(hash.New(), strconv.FormatFloat(val, 'E', -1, 64))
	}

	return res
}
