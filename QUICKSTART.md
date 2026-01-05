# Quick Reference Guide

## Installation

```bash
go get github.com/ASHISH26940/throttle
```

## Imports

```go
import (
    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/types"
    "github.com/ASHISH26940/throttle/internal/errors"
)
```

## Create Algorithm

```go
cfg := types.Config{
    Rate:   100,              // 100 requests
    Window: time.Second,      // per second
    Burst:  150,              // burst capacity
}

// Option 1: Direct creation
limiter, err := algorithm.NewTokenBucket(cfg)

// Option 2: Factory
factory := algorithm.NewFactory()
limiter, err := factory.Create(cfg)

// Option 3: By type
limiter, err := algorithm.CreateByType(algorithm.AlgorithmLeakyBucket, cfg)
```

## Check Rate Limit

```go
err := limiter.Allow("user123")
if err != nil {
    // Rate limited!
    if rlErr, ok := err.(*errors.RateLimitError); ok {
        retryAfter := rlErr.RetryAfter
        remaining := rlErr.Remaining
    }
}
```

## HTTP Middleware

```go
func rateLimitMiddleware(limiter algorithm.Algorithm) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if err := limiter.Allow(r.RemoteAddr); err != nil {
                http.Error(w, "Rate limit exceeded", 429)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Get Statistics

```go
stats := limiter.Stats()
fmt.Printf("Keys: %d, Allowed: %d, Denied: %d\n",
    stats.TotalKeys, stats.TotalAllowed, stats.TotalDenied)
```

## Reset

```go
limiter.Reset("user123")   // Reset specific key
limiter.ResetAll()         // Reset everything
```

## Close

```go
defer limiter.Close()  // Always close when done
```

## Algorithm Comparison

| Algorithm          | Best For        | Complexity | Memory   |
| ------------------ | --------------- | ---------- | -------- |
| **Token Bucket**   | Bursty traffic  | O(1)       | Low      |
| **Leaky Bucket**   | Smooth output   | O(1)       | Medium   |
| **Fixed Window**   | Simple counting | O(1)       | Very Low |
| **Sliding Window** | High accuracy   | O(n)       | High     |

## Configuration Presets

```go
// High throughput API
cfg := types.Config{Rate: 1000, Window: time.Second, Burst: 1500}

// Strict limiting
cfg := types.Config{Rate: 10, Window: time.Minute, Burst: 0}

// Default
cfg := types.DefaultConfig() // 100/sec, burst 150
```

## Error Handling Helpers

```go
import "github.com/ASHISH26940/throttle/internal/errors"

errors.IsRateLimit(err)          // Check if rate limit error
errors.RetryAfter(err)           // Get retry duration
errors.RemainingTokens(err)      // Get remaining tokens
errors.RateLimitKey(err)         // Get rate-limited key
```

## Common Patterns

### Per-User Limiting

```go
limiter.Allow(userID)
```

### Per-IP Limiting

```go
limiter.Allow(req.RemoteAddr)
```

### Per-Endpoint Limiting

```go
limiter.Allow(endpoint + ":" + userID)
```

### Multi-Tier

```go
limiters[tier].Allow(userID)
```

---

**Full Documentation**: See [README.md](README.md) | [ARCHITECTURE.md](ARCHITECTURE.md) | [API.md](API.md) | [EXAMPLES.md](EXAMPLES.md)
