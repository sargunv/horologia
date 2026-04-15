package api

import (
	"context"
	"sync"
	"time"
)

const (
	passwordThrottleBaseDelay = 250 * time.Millisecond
	passwordThrottleMaxDelay  = 4 * time.Second
)

type passwordThrottleEntry struct {
	failures   int
	retryAfter time.Time
}

type passwordThrottle struct {
	mu      sync.Mutex
	entries map[string]passwordThrottleEntry
}

var defaultPasswordThrottle = &passwordThrottle{
	entries: map[string]passwordThrottleEntry{},
}

func (t *passwordThrottle) beforeAttempt(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}

	t.mu.Lock()
	entry, ok := t.entries[key]
	t.mu.Unlock()
	if !ok {
		return nil
	}

	delay := time.Until(entry.retryAfter)
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (t *passwordThrottle) recordFailure(key string) {
	if key == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	entry := t.entries[key]
	entry.failures++
	delay := min(passwordThrottleBaseDelay<<minInt(entry.failures-1, 4), passwordThrottleMaxDelay)
	entry.retryAfter = time.Now().Add(delay)
	t.entries[key] = entry
}

func (t *passwordThrottle) recordSuccess(key string) {
	if key == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, key)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
