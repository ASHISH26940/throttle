package throttle

import (
	"context"
	"time"

	"github.com/ASHISH26940/throttle/internal/algorithm"
	"github.com/ASHISH26940/throttle/internal/types"
)

// Re-export algorithm types for public use
type AlgorithmType = algorithm.AlgorithmType

const (
	TokenBucket   AlgorithmType = algorithm.AlgorithmTokenBucket
	LeakyBucket   AlgorithmType = algorithm.AlgorithmLeakyBucket
	FixedWindow   AlgorithmType = algorithm.AlgorithmFixedWindow
	SlidingWindow AlgorithmType = algorithm.AlgorithmSlidingWindow
)

// RateLimiter is the public interface for rate limiting
type RateLimiter interface {
	Allow(key string) error
	Close() error
	Reset(key string)
	ResetAll()
	Stats() Stats
}

// Stats represents rate limiter statistics
type Stats struct {
	Type         string
	TotalKeys    int64
	TotalAllowed int64
	TotalDenied  int64
	Evictions    int64
}

type rateLimiter struct {
	algo algorithm.Algorithm
}

// New creates a new rate limiter with the default algorithm (Token Bucket)
func New(cfg Config) (RateLimiter, error) {
	return NewWithAlgorithm(cfg, TokenBucket)
}

// NewWithAlgorithm creates a rate limiter with a specific algorithm
func NewWithAlgorithm(cfg Config, algType AlgorithmType) (RateLimiter, error) {
	internalCfg := types.Config{
		Rate:   cfg.Rate,
		Window: cfg.Window,
		Burst:  cfg.Burst,
	}

	algo, err := algorithm.CreateByType(algType, internalCfg)
	if err != nil {
		return nil, err
	}

	return &rateLimiter{algo: algo}, nil
}

func (rl *rateLimiter) Allow(key string) error {
	return rl.algo.Allow(key)
}

func (rl *rateLimiter) Close() error {
	return rl.algo.Close()
}

func (rl *rateLimiter) Reset(key string) {
	rl.algo.Reset(key)
}

func (rl *rateLimiter) ResetAll() {
	rl.algo.ResetAll()
}

func (rl *rateLimiter) Stats() Stats {
	s := rl.algo.Stats()
	return Stats{
		Type:         s.Type.String(),
		TotalKeys:    s.TotalKeys,
		TotalAllowed: s.TotalAllowed,
		TotalDenied:  s.TotalDenied,
		Evictions:    s.Evictions,
	}
}

// Wait blocks until rate limit allows or context is done (future implementation)
func (rl *rateLimiter) Wait(ctx context.Context, key string) error {
	// Simple implementation: keep trying
	for {
		err := rl.algo.Allow(key)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Retry
		}
	}
}