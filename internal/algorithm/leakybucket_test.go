package algorithm

import (
	"testing"
	"time"

	"github.com/ASHISH26940/throttle/internal/types"
)

func TestLeakyBucketBasic(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  10,
	}

	lb, err := NewLeakyBucket(cfg)
	if err != nil {
		t.Fatalf("Failed to create leaky bucket: %v", err)
	}
	defer lb.Close()

	// Should allow requests up to burst capacity
	for i := 0; i < 10; i++ {
		err := lb.Allow("user1")
		if err != nil {
			t.Errorf("Request %d should be allowed, got error: %v", i, err)
		}
	}

	// 11th request should be denied (queue full)
	err = lb.Allow("user1")
	if err == nil {
		t.Error("Request 11 should be denied (queue full)")
	}

	// Check stats
	stats := lb.Stats()
	if stats.Type != AlgorithmLeakyBucket {
		t.Errorf("Expected type LeakyBucket, got %v", stats.Type)
	}
	if stats.TotalAllowed != 10 {
		t.Errorf("Expected 10 allowed, got %d", stats.TotalAllowed)
	}
	if stats.TotalDenied != 1 {
		t.Errorf("Expected 1 denied, got %d", stats.TotalDenied)
	}
}

func TestLeakyBucketLeak(t *testing.T) {
	cfg := types.Config{
		Rate:   10, // 10 per second = leak every 100ms
		Window: time.Second,
		Burst:  5,
	}

	lb, err := NewLeakyBucket(cfg)
	if err != nil {
		t.Fatalf("Failed to create leaky bucket: %v", err)
	}
	defer lb.Close()

	// Fill queue
	for i := 0; i < 5; i++ {
		lb.Allow("user1")
	}

	// Queue should be full
	if err := lb.Allow("user1"); err == nil {
		t.Error("Should be denied when queue full")
	}

	// Wait for some leaking (200ms = 2 leaks at 10/sec rate)
	time.Sleep(220 * time.Millisecond)

	// Should have space for at least 2 more now
	if err := lb.Allow("user1"); err != nil {
		t.Errorf("Should be allowed after leak: %v", err)
	}
	if err := lb.Allow("user1"); err != nil {
		t.Errorf("Should be allowed after leak: %v", err)
	}
}

func TestLeakyBucketMultipleKeys(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  5,
	}

	lb, err := NewLeakyBucket(cfg)
	if err != nil {
		t.Fatalf("Failed to create leaky bucket: %v", err)
	}
	defer lb.Close()

	// Different keys should have separate queues
	for i := 0; i < 5; i++ {
		if err := lb.Allow("user1"); err != nil {
			t.Errorf("user1 request %d failed: %v", i, err)
		}
		if err := lb.Allow("user2"); err != nil {
			t.Errorf("user2 request %d failed: %v", i, err)
		}
	}

	stats := lb.Stats()
	if stats.TotalAllowed != 10 {
		t.Errorf("Expected 10 allowed (5 per user), got %d", stats.TotalAllowed)
	}
}

func TestLeakyBucketReset(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  3,
	}

	lb, err := NewLeakyBucket(cfg)
	if err != nil {
		t.Fatalf("Failed to create leaky bucket: %v", err)
	}
	defer lb.Close()

	// Fill queue
	for i := 0; i < 3; i++ {
		lb.Allow("user1")
	}

	// Reset specific key
	lb.Reset("user1")

	// Should be able to fill queue again
	for i := 0; i < 3; i++ {
		if err := lb.Allow("user1"); err != nil {
			t.Errorf("Request %d should be allowed after reset: %v", i, err)
		}
	}
}
