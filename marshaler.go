package stats

import (
	"hash"
	"hash/fnv"
	"io"
	"sync"
)

type labelMarshaler interface {
	marshal([]string) uint64
	unmarshal(uint64) []string
}

func newDefaultMarshaler() labelMarshaler {
	return &hashingMarshaler{
		pool: sync.Pool{
			New: func() interface{} { return fnv.New64() },
		},
		st: make(map[uint64][]string),
	}
}

// WARNING: this struct is not thread safe. It has to be safely guarded by a
// mutex in the structure using it.
type hashingMarshaler struct {
	pool sync.Pool

	st map[uint64][]string
}

func (hm *hashingMarshaler) marshal(vs []string) uint64 {
	var hasher = hm.pool.Get().(hash.Hash64)

	hasher.Reset()

	for _, v := range vs {
		io.WriteString(hasher, v)
	}

	res := hasher.Sum64()

	hm.st[res] = vs
	hm.pool.Put(hasher)

	return res
}

func (hm *hashingMarshaler) unmarshal(h uint64) []string {
	return hm.st[h]
}
