package stats

import (
	"sync"

	"github.com/upfluence/stats/internal/hash"
)

type labelMarshaler interface {
	marshal([]string) uint64
	unmarshal(uint64, int) []string
}

func newDefaultMarshaler() labelMarshaler {
	return &hashingMarshaler{
		st: make(map[hashingKey][]string),
	}
}

type hashingKey struct {
	hash uint64
	len  int
}

type hashingMarshaler struct {
	sync.RWMutex
	st map[hashingKey][]string
}

func (hm *hashingMarshaler) marshal(vs []string) uint64 {
	res := hash.New()

	for _, v := range vs {
		res = hash.Add(res, v)
	}

	hm.Lock()
	hm.st[hashingKey{hash: res, len: len(vs)}] = vs
	hm.Unlock()

	return res
}

func (hm *hashingMarshaler) unmarshal(h uint64, len int) []string {
	hm.RLock()
	defer hm.RUnlock()

	return hm.st[hashingKey{hash: h, len: len}]
}
