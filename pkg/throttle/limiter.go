package throttle

import (
	"context"
	"sync/atomic"

	"github.com/ASHISH26940/throttle/internal/errors"
)

type RateLimiter interface {
	Allow(key string) error
	Wait(ctx context.Context, key string) error
	Close() error
	Stats() Stats
}

type Stats struct {
	TotalKeys    int64
	TotalAllowed int64
	TotalDenied  int64
	Evictions    int64
}

func New(cfg Config) (RateLimiter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// TODO: Implement when internal/limiter package is created
	return nil, errors.NewConfigError("limiter", "not implemented", "sharded limiter not yet implemented")
}

type rateLimiter struct {
	// TODO: Will be populated when internal/limiter is implemented
	closed       atomic.Int32
	totalAllowed atomic.Int64
	totalDenied  atomic.Int64
}
