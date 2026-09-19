package cachekit

import (
	"sync"
	"testing"
	"time"
)

// TestFlight_Inflight proves the cache_fill_inflight signal tracks blocked
// stampede waiters and returns to zero (FR-021).
func TestFlight_Inflight(t *testing.T) {
	var f Flight
	release := make(chan struct{})
	var wg sync.WaitGroup
	const waiters = 8
	for i := 0; i < waiters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = f.Do("hot-key", func() (any, error) {
				<-release
				return []byte("v"), nil
			})
		}()
	}

	deadline := time.Now().Add(2 * time.Second)
	for f.Inflight() != waiters && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := f.Inflight(); got != waiters {
		t.Fatalf("inflight = %d, want %d", got, waiters)
	}
	close(release)
	wg.Wait()
	if got := f.Inflight(); got != 0 {
		t.Fatalf("inflight after drain = %d, want 0", got)
	}
}
