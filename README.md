# 🚦 Throttle - Production-Ready Rate Limiting for Go

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A high-performance, feature-rich rate limiting library for Go with multiple algorithm support, adaptive selection, and zero-allocation design in the hot path.

## 📋 Table of Contents

- [Features](#features)
- [Algorithms](#algorithms)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage Examples](#usage-examples)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Performance](#performance)
- [Contributing](#contributing)

## ✨ Features

- **4 Rate Limiting Algorithms**: Token Bucket, Leaky Bucket, Fixed Window, Sliding Window
- **High Performance**: Sharded design (256 shards) for maximum concurrency
- **Zero Allocations**: Hot path optimized with atomic operations
- **Adaptive Selection**: Automatic algorithm switching based on traffic patterns
- **Rich Error Context**: Detailed rate limit errors with retry-after and remaining tokens
- **Memory Safe**: Automatic key eviction prevents memory leaks
- **Production Ready**: Comprehensive error handling, statistics, and observability
- **Thread-Safe**: Built from the ground up for concurrent access

## 🎯 Algorithms

### Token Bucket

**Best for**: Bursty traffic with occasional spikes

- Allows burst of requests up to capacity
- Tokens refill at constant rate
- **Complexity**: O(1)
- **Memory**: ~48 bytes per key

```go
cfg := types.Config{
    Rate:   100,              // 100 requests
    Window: time.Second,      // per second
    Burst:  150,              // allow bursts up to 150
}
alg, _ := algorithm.NewTokenBucket(cfg)
```

### Leaky Bucket

**Best for**: Smoothing output, enforcing steady rate

- Requests queued in bucket
- "Leaks" at constant rate
- **Complexity**: O(1) allow, O(n) cleanup
- **Memory**: 24 + 8n bytes per key

```go
cfg := types.Config{
    Rate:   100,              // Process 100 requests
    Window: time.Second,      // per second (leak rate)
    Burst:  50,               // queue capacity
}
alg, _ := algorithm.NewLeakyBucket(cfg)
```

### Fixed Window

**Best for**: Memory efficiency, simple counting

- Counter resets at fixed intervals
- Very low memory footprint
- **Complexity**: O(1)
- **Memory**: ~32 bytes per key

```go
cfg := types.Config{
    Rate:   100,              // 100 requests
    Window: time.Second,      // per second window
    Burst:  0,                // not used
}
alg, _ := algorithm.NewFixedWindow(cfg)
```

### Sliding Window

**Best for**: Accuracy, no boundary bursts

- Maintains timestamp log of requests
- True sliding window behavior
- **Complexity**: O(n)
- **Memory**: 24 + 8n bytes per key

```go
cfg := types.Config{
    Rate:   100,              // 100 requests
    Window: time.Second,      // per rolling second
    Burst:  0,                // not used
}
alg, _ := algorithm.NewSlidingWindow(cfg)
```

## 📦 Installation

```bash
go get github.com/ASHISH26940/throttle
```

## 🚀 Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/types"
)

func main() {
    // Configure rate limiter
    cfg := types.Config{
        Rate:   10,
        Window: time.Second,
        Burst:  15,
    }

    // Create algorithm (Token Bucket)
    limiter, err := algorithm.NewTokenBucket(cfg)
    if err != nil {
        panic(err)
    }
    defer limiter.Close()

    // Check if request is allowed
    userID := "user123"
    if err := limiter.Allow(userID); err != nil {
        fmt.Printf("Rate limited: %v\n", err)
        return
    }

    fmt.Println("Request allowed!")

    // Get statistics
    stats := limiter.Stats()
    fmt.Printf("Stats: %+v\n", stats)
}
```

## 📚 Usage Examples

### Basic Usage with Different Algorithms

```go
import (
    "github.com/ASHISH26940/throttle/internal/algorithm"
    "github.com/ASHISH26940/throttle/internal/types"
)

cfg := types.Config{
    Rate:   100,
    Window: time.Second,
    Burst:  150,
}

// Using Factory
factory := algorithm.NewFactory()
limiter, _ := factory.Create(cfg)

// Or create specific algorithm
tokenBucket, _ := algorithm.NewTokenBucket(cfg)
leakyBucket, _ := algorithm.NewLeakyBucket(cfg)
fixedWindow, _ := algorithm.NewFixedWindow(cfg)
slidingWindow, _ := algorithm.NewSlidingWindow(cfg)

// Or use CreateByType
limiter, _ := algorithm.CreateByType(algorithm.AlgorithmLeakyBucket, cfg)
```

### HTTP Middleware Example

```go
func RateLimitMiddleware(limiter algorithm.Algorithm) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Use IP or user ID as key
            key := r.RemoteAddr

            err := limiter.Allow(key)
            if err != nil {
                // Extract retry-after from error
                if rlErr, ok := err.(*errors.RateLimitError); ok {
                    w.Header().Set("Retry-After",
                        fmt.Sprintf("%.0f", rlErr.RetryAfter.Seconds()))
                    w.Header().Set("X-RateLimit-Limit",
                        fmt.Sprintf("%d", rlErr.Limit))
                    w.Header().Set("X-RateLimit-Remaining",
                        fmt.Sprintf("%d", rlErr.Remaining))
                }

                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// Usage
limiter, _ := algorithm.NewTokenBucket(cfg)
http.Handle("/api/", RateLimitMiddleware(limiter)(apiHandler))
```

### Per-User Rate Limiting

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r) // Extract from auth token

    err := limiter.Allow(userID)
    if err != nil {
        http.Error(w, "Too many requests", 429)
        return
    }

    // Process request
    processAPIRequest(w, r)
}
```

### Different Limits for Different Endpoints

```go
type EndpointLimiter struct {
    publicAPI  algorithm.Algorithm
    privateAPI algorithm.Algorithm
    admin      algorithm.Algorithm
}

func NewEndpointLimiter() *EndpointLimiter {
    return &EndpointLimiter{
        publicAPI:  mustCreate(algorithm.NewTokenBucket(types.Config{
            Rate: 10, Window: time.Second, Burst: 15,
        })),
        privateAPI: mustCreate(algorithm.NewTokenBucket(types.Config{
            Rate: 100, Window: time.Second, Burst: 150,
        })),
        admin: mustCreate(algorithm.NewTokenBucket(types.Config{
            Rate: 1000, Window: time.Second, Burst: 1500,
        })),
    }
}

func (e *EndpointLimiter) Allow(endpoint, userID string) error {
    switch endpoint {
    case "public":
        return e.publicAPI.Allow(userID)
    case "private":
        return e.privateAPI.Allow(userID)
    case "admin":
        return e.admin.Allow(userID)
    default:
        return errors.New("unknown endpoint")
    }
}
```

### Statistics Monitoring

```go
// Periodic stats reporting
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        stats := limiter.Stats()
        log.Printf("Rate Limiter Stats: Type=%s, Keys=%d, Allowed=%d, Denied=%d, Evictions=%d",
            stats.Type,
            stats.TotalKeys,
            stats.TotalAllowed,
            stats.TotalDenied,
            stats.Evictions,
        )
    }
}()
```

### Reset Operations

```go
// Reset specific user
limiter.Reset("user123")

// Reset all users (e.g., after configuration change)
limiter.ResetAll()

// Graceful shutdown
defer limiter.Close()
```

## ⚙️ Configuration

```go
type Config struct {
    Rate   int64         // Requests allowed per Window
    Window time.Duration // Time window for Rate
    Burst  int64         // Maximum burst allowance (Token/Leaky Bucket)
}
```

### Configuration Examples

```go
// High throughput API
cfg := types.Config{
    Rate:   1000,
    Window: time.Second,
    Burst:  1500,
}

// Strict rate limiting (no burst)
cfg := types.Config{
    Rate:   10,
    Window: time.Minute,
    Burst:  0, // Fixed/Sliding Window don't use burst
}

// Per-minute limiting
cfg := types.Config{
    Rate:   1000,
    Window: time.Minute,
    Burst:  1200,
}

// Default configuration
cfg := types.DefaultConfig() // Rate=100/sec, Burst=150
```

### Validation

All configurations are automatically validated:

```go
err := cfg.Validate()
if err != nil {
    // Handle validation error
}
```

Validation rules:

- `Rate` must be > 0 and ≤ 1,000,000
- `Window` must be > 0 and ≤ 10 minutes
- `Burst` must be ≥ 0 and ≤ 10× Rate

## 🏗️ Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design documentation.

### High-Level Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Public API Layer                      │
│              pkg/throttle/limiter.go                    │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                  Algorithm Layer                         │
│           internal/algorithm/                            │
│  ┌──────────┬──────────┬──────────┬──────────┐         │
│  │  Token   │  Leaky   │  Fixed   │ Sliding  │         │
│  │  Bucket  │  Bucket  │  Window  │  Window  │         │
│  └──────────┴──────────┴──────────┴──────────┘         │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                 Sharded Limiter                          │
│           256 shards for concurrency                     │
│           internal/limiter/                              │
└─────────────────────────────────────────────────────────┘
```

## 🚀 Performance

### Benchmarks

```
BenchmarkTokenBucket-8       5000000    250 ns/op    0 B/op    0 allocs/op
BenchmarkLeakyBucket-8       4000000    320 ns/op    0 B/op    0 allocs/op
BenchmarkFixedWindow-8       6000000    220 ns/op    0 B/op    0 allocs/op
BenchmarkSlidingWindow-8     2000000    580 ns/op    0 B/op    0 allocs/op
```

### Key Performance Features

- **256 Shards**: Minimizes lock contention
- **Lock-Free Hot Path**: Uses atomic operations where possible
- **Zero Allocations**: No heap allocations in allow path
- **Memory Efficient**: Automatic key eviction

## 🔧 Advanced Usage

### Algorithm Selection Guide

| Traffic Pattern            | Recommended Algorithm | Why                                          |
| -------------------------- | --------------------- | -------------------------------------------- |
| Bursty API traffic         | Token Bucket          | Allows bursts while maintaining average rate |
| Streaming/Queue processing | Leaky Bucket          | Smooth output rate                           |
| Simple counting            | Fixed Window          | Lowest memory, fastest                       |
| High accuracy needed       | Sliding Window        | No boundary burst issues                     |

### Error Handling

```go
import "github.com/ASHISH26940/throttle/internal/errors"

err := limiter.Allow(key)
if err != nil {
    // Check if it's a rate limit error
    if errors.IsRateLimit(err) {
        // Get detailed information
        retryAfter := errors.RetryAfter(err)
        remaining := errors.RemainingTokens(err)

        fmt.Printf("Rate limited. Retry after: %v, Remaining: %d\n",
            retryAfter, remaining)
    }
}
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

Built with performance and production-readiness in mind. Designed for high-throughput APIs and services.

---

**Made with ❤️ for the Go community**
