package requesttrace

import (
	"context"
	"errors"
	"sync"
)

type contextKey struct{}

type State struct {
	mu      sync.RWMutex
	traceID string
	spanID  string
	err     error
}

func WithState(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, &State{})
}

func Set(ctx context.Context, traceID, spanID string) {
	state, _ := ctx.Value(contextKey{}).(*State)
	if state == nil {
		return
	}
	state.mu.Lock()
	state.traceID = traceID
	state.spanID = spanID
	state.mu.Unlock()
}

func IDs(ctx context.Context) (string, string) {
	state, _ := ctx.Value(contextKey{}).(*State)
	if state == nil {
		return "", ""
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.traceID, state.spanID
}

func SetError(ctx context.Context, err error) {
	state, _ := ctx.Value(contextKey{}).(*State)
	if state == nil || err == nil {
		return
	}
	state.mu.Lock()
	state.err = errors.Join(state.err, err)
	state.mu.Unlock()
}

func Error(ctx context.Context) error {
	state, _ := ctx.Value(contextKey{}).(*State)
	if state == nil {
		return nil
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.err
}
