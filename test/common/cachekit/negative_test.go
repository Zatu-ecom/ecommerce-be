package cachekit_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"ecommerce-be/common/cachekit"
)

// fakeCache is an in-memory cachekit.Cache for tombstone lifecycle tests
// (no backend, no containers).
type fakeCache struct {
	mu   sync.Mutex
	data map[string][]byte
	vers map[string]uint64
}

func newFakeCache() *fakeCache {
	return &fakeCache{data: map[string][]byte{}, vers: map[string]uint64{}}
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.data[key]
	if !ok {
		return nil, cachekit.ErrMiss
	}
	return b, nil
}

func (f *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = value
	return nil
}

func (f *fakeCache) SetNX(_ context.Context, key string, value []byte, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.data[key]; ok {
		return false, nil
	}
	f.data[key] = value
	return true, nil
}

func (f *fakeCache) CompareAndSet(_ context.Context, key, genKey string, expectedGen uint64, value []byte, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.vers[genKey] != expectedGen {
		return false, nil
	}
	f.data[key] = value
	return true, nil
}

func (f *fakeCache) Incr(_ context.Context, key string) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.vers[key]++
	return f.vers[key], nil
}

func (f *fakeCache) Del(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range keys {
		delete(f.data, k)
	}
	return nil
}

func (f *fakeCache) DelPrefix(_ context.Context, prefix string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k := range f.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(f.data, k)
		}
	}
	return nil
}

// TestIsTombstone proves the sentinel check (whitespace-tolerant) and that
// real DTO bytes never read as negative.
func TestIsTombstone(t *testing.T) {
	if !cachekit.IsTombstone(cachekit.TombstoneBytes) {
		t.Fatal("tombstone bytes must read as tombstone")
	}
	if !cachekit.IsTombstone(append([]byte("  "), append(cachekit.TombstoneBytes, '\n')...)) {
		t.Fatal("padded tombstone must read as tombstone")
	}
	if cachekit.IsTombstone([]byte(`{"id":1}`)) {
		t.Fatal("DTO bytes must not read as tombstone")
	}
	if cachekit.IsTombstone(nil) {
		t.Fatal("nil bytes must not read as tombstone")
	}
}

// TestTombstone_StoreAsMiss proves repeated lookups of missing data do not
// reach the database: the first miss stores the sentinel, later reads
// return ErrMiss without refilling.
func TestTombstone_StoreAsMiss(t *testing.T) {
	ctx := context.Background()
	c := newFakeCache()
	loads := 0
	fill := func(context.Context) ([]byte, error) {
		loads++
		return nil, nil // confirmed absent
	}

	_, err := cachekit.FetchBytes(ctx, c, cachekit.NopRecorder(), &cachekit.Flight{},
		"m", "o", "k", time.Minute, time.Minute, fill)
	if !errors.Is(err, cachekit.ErrMiss) {
		t.Fatalf("first miss must return ErrMiss, got %v", err)
	}
	raw, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("tombstone must be stored, got %v", err)
	}
	if !bytes.Equal(raw, cachekit.TombstoneBytes) {
		t.Fatalf("stored bytes must be the sentinel, got %q", raw)
	}

	_, err = cachekit.FetchBytes(ctx, c, cachekit.NopRecorder(), &cachekit.Flight{},
		"m", "o", "k", time.Minute, time.Minute, fill)
	if !errors.Is(err, cachekit.ErrMiss) {
		t.Fatalf("tombstone hit must return ErrMiss, got %v", err)
	}
	if loads != 1 {
		t.Fatalf("negative hit must not refill: loads = %d", loads)
	}
}

// TestTombstone_DelOnWrite proves any write clears the sentinel through the
// same single-key Del (no second key to remember), and a later create
// refills normally.
func TestTombstone_DelOnWrite(t *testing.T) {
	ctx := context.Background()
	c := newFakeCache()
	loads := 0
	missing := true
	fill := func(context.Context) ([]byte, error) {
		loads++
		if missing {
			return nil, nil
		}
		return []byte(`{"id":9}`), nil
	}
	fetch := func() ([]byte, error) {
		return cachekit.FetchBytes(ctx, c, cachekit.NopRecorder(), &cachekit.Flight{},
			"m", "o", "k", time.Minute, time.Minute, fill)
	}

	if _, err := fetch(); !errors.Is(err, cachekit.ErrMiss) {
		t.Fatalf("expected miss, got %v", err)
	}
	if err := c.Del(ctx, "k"); err != nil { // write path clears
		t.Fatal(err)
	}
	missing = false // row created
	body, err := fetch()
	if err != nil {
		t.Fatalf("post-create read must refill, got %v", err)
	}
	if !bytes.Equal(body, []byte(`{"id":9}`)) {
		t.Fatalf("refill mismatch: %q", body)
	}
	if loads != 2 {
		t.Fatalf("expected exactly 2 fills, got %d", loads)
	}
}
