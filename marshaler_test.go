package stats

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	marshalerBar = "bar"
	marshalerFoo = "foo"
)

func TestHashingMarshalerCopiesLabels(t *testing.T) {
	marshaler := newDefaultMarshaler()
	labels := []string{marshalerFoo, marshalerBar}
	key := marshaler.marshal(labels)
	labels[0] = "changed"

	assert.Equal(t, []string{marshalerFoo, marshalerBar}, marshaler.unmarshal(key, len(labels)))
}

func TestHashingMarshalerConcurrentMarshal(t *testing.T) {
	marshaler := newDefaultMarshaler()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			marshaler.marshal([]string{marshalerFoo, marshalerBar})
		}()
	}

	wg.Wait()

	key := marshaler.marshal([]string{marshalerFoo, marshalerBar})

	assert.Equal(t, []string{marshalerFoo, marshalerBar}, marshaler.unmarshal(key, 2))
}
