package scheduler

import (
	"context"
	"encoding/json"
)

type Handler func(ctx context.Context, payload json.RawMessage) error

var registry = map[string]Handler{}

func Register(command string, handler Handler) {
	if _, exists := registry[command]; exists {
		panic("command already registered: " + command)
	}
	registry[command] = handler
}

func Get(command string) (Handler, bool) {
	h, ok := registry[command]
	return h, ok
}
