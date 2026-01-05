package algorithm

import (
	"testing"
	"time"

	"github.com/ASHISH26940/throttle/internal/types"
)

func TestSlidingWindowBasic(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: 100 * time.Millisecond,
		Burst:  0,
	}

	sw, err := NewSlidingWindow(cfg)
	if err != nil {
		t.Fatalf("Failed to create sliding window: %v", err)
	}
	defer sw.Close()

	// Should allow 5 requests in window
	for i := 0; i < 5; i++ {
		err := sw.Allow("user1")
		if err != nil {
			t.Errorf("Request %d should be allowed, got error: %v", i, err)
		}
	}

	// 6th request should be denied (limit reached)
	err = sw.Allow("user1")
	if err == nil {
		t.Error("Request 6 should be denied (limit reached)")
	}

	// Check stats
	stats := sw.Stats()
	if stats.Type != AlgorithmSlidingWindow {
		t.Errorf("Expected type SlidingWindow, got %v", stats.Type)
	}
	if stats.TotalAllowed != 5 {
		t.Errorf("Expected 5 allowed, got %d", stats.TotalAllowed)
	}
	if stats.TotalDenied != 1 {
		t.Errorf("Expected 1 denied, got %d", stats.TotalDenied)
	}
}

func TestSlidingWindowSliding(t *testing.T) {
	cfg := types.Config{
		Rate:   3,
		Window: 100 * time.Millisecond,
		Burst:  0,
	}

	sw, err := NewSlidingWindow(cfg)
	if err != nil {
		t.Fatalf("Failed to create sliding window: %v", err)
	}
	defer sw.Close()

	// Use up limit
	for i := 0; i < 3; i++ {
		sw.Allow("user1")
	}

	// Should be denied
	if err := sw.Allow("user1"); err == nil {
		t.Error("Should be denied (limit reached)")
	}

	// Wait for first request to expire (110ms total)
	time.Sleep(110 * time.Millisecond)

	// Now oldest timestamp should be expired, allowing 1 more request
	if err := sw.Allow("user1"); err != nil {
		t.Errorf("Should be allowed after oldest expired: %v", err)
	}
}

func TestSlidingWindowMultipleKeys(t *testing.T) {
	cfg := types.Config{
		Rate:   3,
		Window: time.Second,
		Burst:  0,
	}

	sw, err := NewSlidingWindow(cfg)
	if err != nil {
		t.Fatalf("Failed to create sliding window: %v", err)
	}
	defer sw.Close()

	// Different keys should have separate limits
	for i := 0; i < 3; i++ {
		if err := sw.Allow("user1"); err != nil {
			t.Errorf("user1 request %d failed: %v", i, err)
		}
		if err := sw.Allow("user2"); err != nil {
			t.Errorf("user2 request %d failed: %v", i, err)
		}
	}

	stats := sw.Stats()
	if stats.TotalAllowed != 6 {
		t.Errorf("Expected 6 allowed (3 per user), got %d", stats.TotalAllowed)
	}
}

func TestSlidingWindowReset(t *testing.T) {
	cfg := types.Config{
		Rate:   3,
		Window: time.Second,
		Burst:  0,
	}

	sw, err := NewSlidingWindow(cfg)
	if err != nil {
		t.Fatalf("Failed to create sliding window: %v", err)
	}
	defer sw.Close()

	// Use up limit
	for i := 0; i < 3; i++ {
		sw.Allow("user1")
	}

	// Reset specific key
	sw.Reset("user1")

	// Should be allowed again
	for i := 0; i < 3; i++ {
		if err := sw.Allow("user1"); err != nil {
			t.Errorf("Request %d should be allowed after reset: %v", i, err)
		}
	}
}
