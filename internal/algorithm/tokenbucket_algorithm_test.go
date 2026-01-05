package algorithm

import (
	"testing"
	"time"

	"github.com/ASHISH26940/throttle/internal/types"
)

func TestTokenBucketBasic(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  10,
	}

	tb, err := NewTokenBucket(cfg)
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}
	defer tb.Close()

	// Should allow first 10 requests (burst)
	for i := 0; i < 10; i++ {
		err := tb.Allow("user1")
		if err != nil {
			t.Errorf("Request %d should be allowed, got error: %v", i, err)
		}
	}

	// 11th request should be denied (burst exhausted)
	err = tb.Allow("user1")
	if err == nil {
		t.Error("Request 11 should be denied (burst exhausted)")
	}

	// Check stats
	stats := tb.Stats()
	if stats.Type != AlgorithmTokenBucket {
		t.Errorf("Expected type TokenBucket, got %v", stats.Type)
	}
	if stats.TotalAllowed != 10 {
		t.Errorf("Expected 10 allowed, got %d", stats.TotalAllowed)
	}
	if stats.TotalDenied != 1 {
		t.Errorf("Expected 1 denied, got %d", stats.TotalDenied)
	}
}

func TestTokenBucketMultipleKeys(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  5,
	}

	tb, err := NewTokenBucket(cfg)
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}
	defer tb.Close()

	// Different keys should have separate limits
	for i := 0; i < 5; i++ {
		if err := tb.Allow("user1"); err != nil {
			t.Errorf("user1 request %d failed: %v", i, err)
		}
		if err := tb.Allow("user2"); err != nil {
			t.Errorf("user2 request %d failed: %v", i, err)
		}
	}

	stats := tb.Stats()
	if stats.TotalAllowed != 10 {
		t.Errorf("Expected 10 allowed (5 per user), got %d", stats.TotalAllowed)
	}
	if stats.TotalKeys < 2 {
		t.Errorf("Expected at least 2 keys, got %d", stats.TotalKeys)
	}
}

func TestTokenBucketResetAll(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  5,
	}

	tb, err := NewTokenBucket(cfg)
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}
	defer tb.Close()

	// Use up all tokens
	for i := 0; i < 5; i++ {
		tb.Allow("user1")
	}

	// Reset
	tb.ResetAll()

	stats := tb.Stats()
	if stats.TotalAllowed != 0 {
		t.Errorf("After reset, expected 0 allowed, got %d", stats.TotalAllowed)
	}
}

func TestTokenBucketClose(t *testing.T) {
	cfg := types.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  5,
	}

	tb, err := NewTokenBucket(cfg)
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	// Close
	err = tb.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Second close should not error
	err = tb.Close()
	if err != nil {
		t.Errorf("Second close should not error: %v", err)
	}
}
