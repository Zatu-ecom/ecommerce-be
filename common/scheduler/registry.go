package scheduler

import (
	"context"
	"encoding/json"
	"sync"

	"ecommerce-be/common/log"
)

type Handler func(ctx context.Context, payload json.RawMessage) error

var (
	mu       sync.RWMutex
	registry = map[string]Handler{}
)

func Register(command string, handler Handler) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := registry[command]; exists {
		// Already registered — skip silently but emit a warning so it's visible
		// in logs if this ever happens outside of tests (e.g. accidental double init).
		log.Warn("scheduler: command already registered, skipping: " + command)
		return
	}
	registry[command] = handler
}

func Get(command string) (Handler, bool) {
	mu.RLock()
	defer mu.RUnlock()

	h, ok := registry[command]
	return h, ok
}
