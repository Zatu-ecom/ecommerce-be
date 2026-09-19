package cachekit

import (
	"golang.org/x/sync/singleflight"
)

// Flight collapses concurrent identical misses within one process so a hot
// key expiry costs one database fill per pod, not one per goroutine.
// Cross-pod herds stay bounded by pod count (accepted in P1, pre-spec §8.4).
type Flight struct {
	group singleflight.Group
}

// Do executes fn for key, sharing one in-flight call across concurrent
// callers. Shared results report shared=true.
func (f *Flight) Do(key string, fn func() (any, error)) (any, bool, error) {
	v, err, shared := f.group.Do(key, fn)
	return v, shared, err
}

// Forget drops an in-flight call so a poisoned fill (e.g. context cancelled
// mid-DB-read) is not shared further. Rarely needed; available for strategies.
func (f *Flight) Forget(key string) {
	f.group.Forget(key)
}
