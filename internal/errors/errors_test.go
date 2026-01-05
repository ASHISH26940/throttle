package errors

import (
	"testing"
	"time"
)

func TestRateLimitError(t *testing.T) {
	rle := NewRateLimitError("user1", ErrInvalidTokenCount, 100, 5, 30*time.Second, time.Now(), true, 42)
	
	if !IsRateLimit(rle) {
		t.Fatal("should detect rate limit")
	}
	if RetryAfter(rle) != 30*time.Second {
		t.Fatal("wrong retry")
	}
	if !BurstDetected(rle) {
		t.Fatal("should detect burst")
	}
	if ShardID(rle) != 42 {
		t.Fatal("wrong shard")
	}
}

func TestConfigError(t *testing.T) {
	err := NewConfigError("Rate", int64(0), "must be positive")
	if field := ConfigField(err); field != "Rate" {
		t.Fatalf("expected Rate, got %q", field)
	}
}

func TestHelpers_NoPanic(t *testing.T) {
	// Nil safe
	if RetryAfter(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}
