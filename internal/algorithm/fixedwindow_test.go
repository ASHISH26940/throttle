package algorithm

import (
	"testing"
	"time"

	"github.com/ASHISH26940/throttle/internal/types"
)

func TestFixedWindowBasic(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: 100 * time.Millisecond,
		Burst:  0,
	}

	fw, err := NewFixedWindow(cfg)
	if err != nil {
		t.Fatalf("Failed to create fixed window: %v", err)
	}
	defer fw.Close()

	// Should allow 5 requests in window
	for i := 0; i < 5; i++ {
		err := fw.Allow("user1")
		if err != nil {
			t.Errorf("Request %d should be allowed, got error: %v", i, err)
		}
	}

	// 6th request should be denied (limit reached)
	err = fw.Allow("user1")
	if err == nil {
		t.Error("Request 6 should be denied (limit reached)")
	}

	// Check stats
	stats := fw.Stats()
	if stats.Type != AlgorithmFixedWindow {
		t.Errorf("Expected type FixedWindow, got %v", stats.Type)
	}
	if stats.TotalAllowed != 5 {
		t.Errorf("Expected 5 allowed, got %d", stats.TotalAllowed)
	}
	if stats.TotalDenied != 1 {
		t.Errorf("Expected 1 denied, got %d", stats.TotalDenied)
	}
}

func TestFixedWindowReset(t *testing.T) {
	cfg := types.Config{
		Rate:   3,
		Window: 50 * time.Millisecond,
		Burst:  0,
	}

	fw, err := NewFixedWindow(cfg)
	if err != nil {
		t.Fatalf("Failed to create fixed window: %v", err)
	}
	defer fw.Close()

	// Use up limit
	for i := 0; i < 3; i++ {
		fw.Allow("user1")
	}

	// Should be denied
	if err := fw.Allow("user1"); err == nil {
		t.Error("Should be denied before window reset")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again after window reset
	if err := fw.Allow("user1"); err != nil {
		t.Errorf("Should be allowed after window reset: %v", err)
	}
}

func TestFixedWindowMultipleKeys(t *testing.T) {
	cfg := types.Config{
		Rate:   3,
		Window: time.Second,
		Burst:  0,
	}

	fw, err := NewFixedWindow(cfg)
	if err != nil {
		t.Fatalf("Failed to create fixed window: %v", err)
	}
	defer fw.Close()

	// Different keys should have separate limits
	for i := 0; i < 3; i++ {
		if err := fw.Allow("user1"); err != nil {
			t.Errorf("user1 request %d failed: %v", i, err)
		}
		if err := fw.Allow("user2"); err != nil {
			t.Errorf("user2 request %d failed: %v", i, err)
		}
	}

	stats := fw.Stats()
	if stats.TotalAllowed != 6 {
		t.Errorf("Expected 6 allowed (3 per user), got %d", stats.TotalAllowed)
	}
}

func TestFixedWindowResetKey(t *testing.T) {
	cfg := types.Config{
		Rate:   3,
		Window: time.Second,
		Burst:  0,
	}

	fw, err := NewFixedWindow(cfg)
	if err != nil {
		t.Fatalf("Failed to create fixed window: %v", err)
	}
	defer fw.Close()

	// Use up limit
	for i := 0; i < 3; i++ {
		fw.Allow("user1")
	}

	// Reset specific key
	fw.Reset("user1")

	// Should be allowed again
	if err := fw.Allow("user1"); err != nil {
		t.Errorf("Should be allowed after reset: %v", err)
	}
}
