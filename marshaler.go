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
		pool: sync.Pool{New: func() interface{} { return fnv.New64() }},
		st:   make(map[uint64][]string),
	}
}

type hashingMarshaler struct {
	pool sync.Pool

	sync.RWMutex
	st map[uint64][]string
}

func (hm *hashingMarshaler) marshal(vs []string) uint64 {
	var hasher = hm.pool.Get().(hash.Hash64)

	hasher.Reset()

	for _, v := range vs {
		io.WriteString(hasher, v)
	}

	res := hasher.Sum64()

	hm.Lock()
	hm.st[res] = vs
	hm.Unlock()
	hm.pool.Put(hasher)

	return res
}

func (hm *hashingMarshaler) unmarshal(h uint64) []string {
	hm.RLock()
	defer hm.RUnlock()

	return hm.st[h]
}
