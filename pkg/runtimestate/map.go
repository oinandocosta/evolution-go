// Package runtimestate owns the synchronized registries shared by HTTP handlers
// and WhatsApp event goroutines. Locks cover registry access only, never network I/O.
package runtimestate

import (
	"context"
	"sync"
	"time"
)

type Map[K comparable, V comparable] struct {
	mu     sync.RWMutex
	values map[K]V
	guards map[K]*keyGuard
}

func New[K comparable, V comparable]() *Map[K, V] { return &Map[K, V]{} }

func (m *Map[K, V]) Load(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.values[key]
	return value, ok
}

func (m *Map[K, V]) Get(key K) V { value, _ := m.Load(key); return value }

func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = make(map[K]V)
	}
	m.values[key] = value
}

func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
}

// CompareAndDelete prevents an old client teardown from deleting a newer client.
func (m *Map[K, V]) CompareAndDelete(key K, expected V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.values[key]
	if !ok || current != expected {
		return false
	}
	delete(m.values, key)
	return true
}

type keyGuard struct {
	mu    sync.Mutex
	users int
}

// LockKey serializes lifecycle operations for one instance, without holding a
// registry mutex during network I/O. References keep queued users on the same lock.
func (m *Map[K, V]) LockKey(key K) func() {
	guard := m.retainGuard(key)
	guard.mu.Lock()
	return func() { guard.mu.Unlock(); m.releaseGuard(key, guard) }
}

// LockKeyContext shares the same per-instance lock as lifecycle operations, but
// does not leave a queued lock or goroutine behind when an HTTP query expires.
func (m *Map[K, V]) LockKeyContext(ctx context.Context, key K) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	guard := m.retainGuard(key)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if guard.mu.TryLock() {
			if err := ctx.Err(); err != nil {
				guard.mu.Unlock()
				m.releaseGuard(key, guard)
				return nil, err
			}
			return func() { guard.mu.Unlock(); m.releaseGuard(key, guard) }, nil
		}
		select {
		case <-ctx.Done():
			m.releaseGuard(key, guard)
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *Map[K, V]) retainGuard(key K) *keyGuard {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.guards == nil {
		m.guards = make(map[K]*keyGuard)
	}
	guard := m.guards[key]
	if guard == nil {
		guard = &keyGuard{}
		m.guards[key] = guard
	}
	guard.users++
	return guard
}

func (m *Map[K, V]) releaseGuard(key K, guard *keyGuard) {
	m.mu.Lock()
	defer m.mu.Unlock()
	guard.users--
	if guard.users == 0 {
		delete(m.guards, key)
	}
}
