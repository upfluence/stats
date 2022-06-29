package stats

import "sync"

const (
	offset64 = 14695981039346656037
	prime64  = 1099511628211

	magicRune = '\U0010FFFF' + 1
)

// hashNew initializies a new fnv64a hash value.
func hashNew() uint64 {
	return offset64
}

// hashAdd adds a string to a fnv64a hash value, returning the updated hash.
func hashAdd(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}

	h ^= magicRune
	h *= prime64

	return h
}

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
	res := hashNew()

	for _, v := range vs {
		res = hashAdd(res, v)
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
