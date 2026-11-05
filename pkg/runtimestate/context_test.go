package runtimestate

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLifecycleLockContextCancellationRetainsOwnerAndOtherInstances(t *testing.T) {
	registry := New[string, int]()
	owner := registry.LockKey("one")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if unlock, err := registry.LockKeyContext(ctx, "one"); unlock != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued lock = %v, acquired=%v", err, unlock != nil)
	}
	other, err := registry.LockKeyContext(context.Background(), "other")
	if err != nil {
		t.Fatal(err)
	}
	other()
	registry.mu.RLock()
	guard := registry.guards["one"]
	users := guard.users
	registry.mu.RUnlock()
	if users != 1 || guard.mu.TryLock() {
		t.Fatal("canceled waiter changed the active lifecycle lock")
	}
	owner()
	registry.mu.RLock()
	remaining := len(registry.guards)
	registry.mu.RUnlock()
	if remaining != 0 {
		t.Fatal("canceled waiter retained a lifecycle guard")
	}
	next, err := registry.LockKeyContext(context.Background(), "one")
	if err != nil {
		t.Fatal(err)
	}
	next()
}

func TestCanceledGenerationWaitDoesNotPublishReplacement(t *testing.T) {
	registry := New[string, *StopSignal]()
	old := NewStopSignal()
	old.Begin()
	registry.Set("one", old)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	unlock := registry.LockKey("one")
	next, err := NewGenerationLockedContext(ctx, registry, "one", time.Hour)
	unlock()
	if !errors.Is(err, context.DeadlineExceeded) || next != nil || registry.Get("one") != old {
		t.Fatalf("replacement=%v cause=%v", next, err)
	}
	old.Complete()
}

func TestCanceledGenerationRequestDoesNotStopCurrentSession(t *testing.T) {
	registry := New[string, *StopSignal]()
	old := NewStopSignal()
	registry.Set("one", old)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	unlock := registry.LockKey("one")
	_, err := NewGenerationLockedContext(ctx, registry, "one", time.Second)
	unlock()
	if !errors.Is(err, context.Canceled) || registry.Get("one") != old {
		t.Fatalf("current generation changed: %v", err)
	}
	select {
	case <-old.Done():
		t.Fatal("already canceled request stopped the active session")
	default:
	}
}
