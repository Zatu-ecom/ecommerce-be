package cachekit_test

import (
	"ecommerce-be/common/cachekit"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestFlight_Collapse proves concurrent identical misses run the fill once
// per process (M2/M3): exactly one leader, every waiter shares the result.
func TestFlight_Collapse(t *testing.T) {
	var f cachekit.Flight
	var runs atomic.Int64
	const callers = 16
	var wg sync.WaitGroup
	results := make([]any, callers)
	shared := make([]bool, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, sh, err := f.Do("same-key", func() (any, error) {
				runs.Add(1)
				time.Sleep(20 * time.Millisecond)
				return "db-row", nil
			})
			if err != nil {
				t.Errorf("fill error: %v", err)
				return
			}
			results[i], shared[i] = v, sh
		}(i)
	}
	wg.Wait()
	if got := runs.Load(); got != 1 {
		t.Fatalf("fill ran %d times, want exactly 1", got)
	}
	// runs == 1 means every caller attached to the single in-flight fill,
	// so all of them must report a shared result.
	for i := range results {
		if results[i] != "db-row" {
			t.Fatalf("caller %d got %v, want db-row", i, results[i])
		}
		if !shared[i] {
			t.Fatalf("caller %d unshared despite a single fill", i)
		}
	}
}

// TestFlight_Inflight proves the cache_fill_inflight signal tracks blocked
// stampede waiters and returns to zero (FR-021).
func TestFlight_Inflight(t *testing.T) {
	var f cachekit.Flight
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
