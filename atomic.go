package stats

import "sync/atomic"

type atomicInt64 struct {
	int64
}

func (c *atomicInt64) Inc()           { c.Add(1) }
func (c *atomicInt64) Add(v int64)    { atomic.AddInt64(&c.int64, v) }
func (c *atomicInt64) Get() int64     { return atomic.LoadInt64(&c.int64) }
func (c *atomicInt64) Update(v int64) { atomic.StoreInt64(&c.int64, v) }
