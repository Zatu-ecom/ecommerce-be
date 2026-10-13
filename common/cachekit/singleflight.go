package cachekit

import (
	"sync/atomic"

	"golang.org/x/sync/singleflight"
)

// Flight collapses concurrent identical misses within one process so a hot
// key expiry costs one database fill per pod, not one per goroutine.
// Cross-pod herds stay bounded by pod count (accepted in P1, pre-spec §8.4).
type Flight struct {
	group    singleflight.Group
	inflight atomic.Int64
}

// Do executes fn for key, sharing one in-flight call across concurrent
// callers. Shared results report shared=true.
func (f *Flight) Do(key string, fn func() (any, error)) (any, bool, error) {
	f.inflight.Add(1)
	defer f.inflight.Add(-1)
	v, err, shared := f.group.Do(key, fn)
	return v, shared, err
}

// Inflight reports the current number of goroutines inside Do — executing
// fills plus shared waiters. It is the cache_fill_inflight signal (FR-021):
// sustained highs mean stampede pressure (one hot key or many cold ones)
// is pinning database fills behind singleflight.
func (f *Flight) Inflight() int64 {
	return f.inflight.Load()
}

// Forget drops an in-flight call so a poisoned fill (e.g. context cancelled
// mid-DB-read) is not shared further. Rarely needed; available for strategies.
func (f *Flight) Forget(key string) {
	f.group.Forget(key)
}
