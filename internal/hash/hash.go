package hash

const (
	offset64 = 14695981039346656037
	prime64  = 1099511628211

	magicRune = '\U0010FFFF' + 1
)

// New initializies a new fnv64a hash value.
func New() uint64 {
	return offset64
}

// Add adds a string to a fnv64a hash value, returning the updated hash.
func Add(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}

	h ^= magicRune
	h *= prime64

	return h
}
