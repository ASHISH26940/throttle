// internal/errors/errors.go
// Production-ready error suite for throttle v1.1.0.
// Zero allocations in happy path, rich context + JSON on error path.

package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ===== SENTINEL ERRORS (12) =====
var (
	// Configuration validation
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrInvalidRate   = errors.New("rate must be positive")
	ErrInvalidWindow = errors.New("window duration must be positive")
	ErrInvalidBurst  = errors.New("burst cannot be negative")

	// Operational
	ErrLimiterClosed     = errors.New("limiter is closed")
	ErrInvalidTokenCount = errors.New("token count must be positive")

	// Context
	ErrContextCanceled = errors.New("context was canceled")
	ErrContextDeadline = errors.New("context deadline exceeded")

	// Internal/Adaptive (future)
	ErrAdaptiveDisabled = errors.New("adaptive mode disabled")
	ErrNoAdjustmentData = errors.New("insufficient traffic data")

	// DoS Protection
	ErrKeyTooLong = errors.New("key too long (max 256 bytes)")

	// Testing
	TestErrBurstDetected = errors.New("test: burst detected")
	TestErrNoTokens      = errors.New("test: no tokens")
)

// ===== RICH ERROR TYPES (4) =====

// RateLimitError - primary error with full context + JSON
type RateLimitError struct {
	Cause      error         `json:"cause,omitempty"`
	Key        string        `json:"key"`
	Limit      int64         `json:"limit"`
	Remaining  int64         `json:"remaining"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	ResetAt    time.Time     `json:"reset_at,omitempty"`
	BurstSeen  bool          `json:"burst_seen"`
	ShardID    uint64        `json:"shard_id"`
}

func (e *RateLimitError) Error() string {
	msg := fmt.Sprintf("rate limit exceeded: %v (key=%s limit=%d remaining=%d)",
		e.Cause, e.Key, e.Limit, e.Remaining)
	if e.RetryAfter > 0 {
		msg += fmt.Sprintf(" retry_after=%v", e.RetryAfter.Round(time.Second))
	}
	if e.BurstSeen {
		msg += " [burst]"
	}
	return msg
}

func (e *RateLimitError) Unwrap() error { return e.Cause }
func (e *RateLimitError) Is(target error) bool {
	_, ok := target.(*RateLimitError)
	return ok
}

func (e *RateLimitError) MarshalJSON() ([]byte, error) {
	type Alias RateLimitError
	return json.Marshal(&struct {
		*Alias
		Cause      string `json:"cause"`
		RetryAfter string `json:"retry_after"`
	}{
		Alias:      (*Alias)(e),
		Cause:      fmt.Sprintf("%v", e.Cause),
		RetryAfter: e.RetryAfter.String(),
	})
}

// ConfigError - detailed validation errors + JSON
type ConfigError struct {
	Field  string      `json:"field"`
	Value  interface{} `json:"value"`
	Reason string      `json:"reason"`
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("invalid config[%s=%v]: %s", e.Field, e.Value, e.Reason)
}

func (e *ConfigError) Is(target error) bool {
	_, ok := target.(*ConfigError)
	return ok
}

// EvictionError - memory pressure + JSON
type EvictionError struct {
	Key       string `json:"key"`
	Evictions int64  `json:"evictions"`
	MaxKeys   int64  `json:"max_keys"`
	ShardID   uint64 `json:"shard_id"`
}

func (e *EvictionError) Error() string {
	return fmt.Sprintf("shard %d memory pressure: evicted %s (total=%d/%d)",
		e.ShardID, e.Key, e.Evictions, e.MaxKeys)
}

func (e *EvictionError) Is(target error) bool {
	_, ok := target.(*EvictionError)
	return ok
}

// ClockError - time skew + JSON
type ClockError struct {
	Cause   error  `json:"cause"`
	SkewNs  int64  `json:"skew_ns"`
	ClockID string `json:"clock_id"`
}

func (e *ClockError) Error() string {
	return fmt.Sprintf("clock %s error: %v (skew=%dns)", e.ClockID, e.Cause, e.SkewNs)
}

func (e *ClockError) Unwrap() error { return e.Cause }
func (e *ClockError) Is(target error) bool {
	_, ok := target.(*ClockError)
	return ok
}

// ===== FACTORY FUNCTIONS =====

func NewRateLimitError(key string, cause error, limit, remaining int64,
	retryAfter time.Duration, resetAt time.Time, burst bool, shardID uint64) *RateLimitError {
	return &RateLimitError{
		Cause:      cause,
		Key:        key,
		Limit:      limit,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		ResetAt:    resetAt,
		BurstSeen:  burst,
		ShardID:    shardID,
	}
}

func WrapRateLimit(key string, err error) *RateLimitError {
	return &RateLimitError{
		Cause:     err,
		Key:       key,
		Limit:     -1,
		Remaining: -1,
	}
}

func NewConfigError(field string, value interface{}, reason string) *ConfigError {
	return &ConfigError{Field: field, Value: value, Reason: reason}
}

func NewEvictionError(key string, evictions, maxKeys int64, shardID uint64) *EvictionError {
	return &EvictionError{
		Key:       key,
		Evictions: evictions,
		MaxKeys:   maxKeys,
		ShardID:   shardID,
	}
}

func NewClockError(cause error, skewNs int64, clockID string) *ClockError {
	return &ClockError{Cause: cause, SkewNs: skewNs, ClockID: clockID}
}

// ===== HELPER FUNCTIONS =====

func IsRateLimit(err error) bool {
	var rle *RateLimitError
	return errors.As(err, &rle)
}

func IsConfigError(err error) bool {
	var ce *ConfigError
	return errors.As(err, &ce)
}

func RetryAfter(err error) time.Duration {
	var rle *RateLimitError
	if errors.As(err, &rle) {
		return rle.RetryAfter
	}
	return 0
}

func RemainingTokens(err error) int64 {
	var rle *RateLimitError
	if errors.As(err, &rle) {
		return rle.Remaining
	}
	return -1
}

func RateLimitKey(err error) string {
	var rle *RateLimitError
	if errors.As(err, &rle) {
		return rle.Key
	}
	return ""
}

func BurstDetected(err error) bool {
	var rle *RateLimitError
	if errors.As(err, &rle) {
		return rle.BurstSeen
	}
	return false
}

func ShardID(err error) uint64 {
	var rle *RateLimitError
	if errors.As(err, &rle) {
		return rle.ShardID
	}
	return 0
}

func Evictions(err error) int64 {
	var ee *EvictionError
	if errors.As(err, &ee) {
		return ee.Evictions
	}
	return 0
}

func ConfigField(err error) string {
	var ce *ConfigError
	if errors.As(err, &ce) {
		return ce.Field
	}
	return ""
}

// MultiError for batch validation
func MultiError(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("multi-error (%d): %w", len(errs), errors.Join(errs...))
}

// Duration is time.Duration alias for JSON marshaling
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}
