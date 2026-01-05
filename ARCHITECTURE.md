# Architecture Documentation

## Table of Contents

1. [Overview](#overview)
2. [Design Principles](#design-principles)
3. [System Architecture](#system-architecture)
4. [Component Details](#component-details)
5. [Data Flow](#data-flow)
6. [Algorithm Implementations](#algorithm-implementations)
7. [Performance Optimizations](#performance-optimizations)
8. [Memory Management](#memory-management)
9. [Concurrency Model](#concurrency-model)

## Overview

Throttle is a production-ready rate limiting library designed for high-throughput Go applications. The architecture prioritizes:

- **Zero-allocation hot path** for maximum performance
- **Pluggable algorithms** for different use cases
- **Horizontal scalability** through sharding
- **Type safety** with compile-time guarantees
- **Observability** through rich metrics and errors

## Design Principles

### 1. Separation of Concerns

```
┌─────────────────────────────────────────────────────┐
│  Layer 1: Public API (pkg/throttle)                 │
│  - User-facing interface                            │
│  - Configuration types                              │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│  Layer 2: Algorithm Interface (internal/algorithm)  │
│  - Algorithm abstraction                            │
│  - Factory pattern                                  │
│  - Statistics aggregation                           │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│  Layer 3: Implementations                           │
│  - Token Bucket, Leaky Bucket                       │
│  - Fixed Window, Sliding Window                     │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│  Layer 4: Sharded Storage (internal/limiter)        │
│  - 256-shard hash map                               │
│  - Lock-per-shard for concurrency                   │
└─────────────────────────────────────────────────────┘
```

### 2. Performance First

- **Sharding**: 256 shards minimize lock contention
- **Atomics**: Lock-free operations where possible
- **Zero Allocations**: Hot path uses pre-allocated structures
- **Cache-Friendly**: State structures aligned to cache lines

### 3. Production Ready

- **Error Context**: Rich error types with retry-after and metadata
- **Memory Bounds**: Automatic eviction prevents unbounded growth
- **Observability**: Built-in statistics and metrics
- **Thread Safety**: Safe for concurrent access from the ground up

## System Architecture

### High-Level Component Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                         Application                             │
└──────────────────────────┬─────────────────────────────────────┘
                           │
                           │ import "throttle/pkg/throttle"
                           │
┌──────────────────────────▼─────────────────────────────────────┐
│                      Public API Layer                           │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  type RateLimiter interface                              │  │
│  │    Allow(key string) error                               │  │
│  │    Wait(ctx, key) error                                  │  │
│  │    Close() error                                         │  │
│  │    Stats() Stats                                         │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────────────────┬─────────────────────────────────────┘
                           │
┌──────────────────────────▼─────────────────────────────────────┐
│                   Algorithm Interface                           │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  type Algorithm interface                                │  │
│  │    Allow(key string) error                               │  │
│  │    Type() AlgorithmType                                  │  │
│  │    Close() error                                         │  │
│  │    Reset(key string)                                     │  │
│  │    ResetAll()                                            │  │
│  │    Stats() AlgorithmStats                                │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  type Factory interface                                  │  │
│  │    Create(cfg Config) (Algorithm, error)                 │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────┬─────────┬─────────┬─────────┬────────────────────────┘
         │         │         │         │
    ┌────▼───┐ ┌──▼───┐ ┌───▼────┐ ┌─▼──────┐
    │ Token  │ │Leaky │ │ Fixed  │ │Sliding │
    │ Bucket │ │Bucket│ │ Window │ │ Window │
    └────┬───┘ └──┬───┘ └───┬────┘ └─┬──────┘
         │        │         │         │
         └────────┴─────────┴─────────┘
                     │
┌────────────────────▼─────────────────────────────────────────┐
│                  Sharded Limiter Core                         │
│                                                                │
│  ┌────┐ ┌────┐ ┌────┐           ┌────┐                       │
│  │Sh 0│ │Sh 1│ │Sh 2│    ...    │S255│                       │
│  └────┘ └────┘ └────┘           └────┘                       │
│                                                                │
│  Each shard: map[uint64]*State + RWMutex                      │
└────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Algorithm Interface (`internal/algorithm/algorithm.go`)

**Purpose**: Define common contract for all rate limiting algorithms

```go
type Algorithm interface {
    Allow(key string) error      // Check and consume
    Type() AlgorithmType          // Return algorithm type
    Close() error                 // Cleanup resources
    Reset(key string)             // Reset single key
    ResetAll()                    // Reset everything
    Stats() AlgorithmStats        // Get statistics
}
```

**Design Rationale**:

- Single responsibility: each method has one job
- Idempotent operations (Close, Reset can be called multiple times)
- Statistics are read-only snapshots

### 2. Token Bucket Implementation

**File**: `internal/algorithm/tokenbucket_algorithm.go`

**State Structure**:

```go
type TokenBucketAlgorithm struct {
    limiter      *limiter.ShardedLimiter
    totalAllowed atomic.Int64
    totalDenied  atomic.Int64
    closed       atomic.Int32
}
```

**Algorithm**:

1. Calculate tokens to add based on time elapsed
2. Cap at burst limit
3. Try to consume 1 token (CAS operation)
4. Update statistics atomically

**Characteristics**:

- Allows bursts up to `Burst` capacity
- Refills at `Rate / Window` per nanosecond
- O(1) time complexity
- ~48 bytes per key memory

### 3. Leaky Bucket Implementation

**File**: `internal/algorithm/leakybucket.go`

**State Structure**:

```go
type LeakyBucketState struct {
    queue      []int64  // Request timestamps
    capacity   int64    // Max queue size
    leakRateNs int64    // Nanoseconds between leaks
    lastLeak   int64    // Last leak timestamp
    lastAccess int64    // For eviction
    mu         sync.Mutex
}
```

**Algorithm**:

1. Calculate leaks since last check
2. Remove leaked requests from queue
3. Check if queue has capacity
4. Add new request if space available

**Characteristics**:

- Smooths output rate
- Queue-based FIFO
- O(1) allow, O(n) cleanup
- 24 + 8n bytes per key

### 4. Fixed Window Implementation

**File**: `internal/algorithm/fixedwindow.go`

**State Structure**:

```go
type FixedWindowState struct {
    windowStart int64
    counter     int64
    limit       int64
    windowNs    int64
    lastAccess  int64
    mu          sync.Mutex
}
```

**Algorithm**:

1. Check if current window expired
2. Reset counter if expired
3. Increment counter
4. Allow if under limit

**Characteristics**:

- Simple counter-based
- Window boundary issues
- O(1) time complexity
- ~32 bytes per key
- Very fast

### 5. Sliding Window Implementation

**File**: `internal/algorithm/slidingwindow.go`

**State Structure**:

```go
type SlidingWindowState struct {
    timestamps []int64  // Request timestamps
    limit      int64
    windowNs   int64
    lastAccess int64
    mu         sync.Mutex
}
```

**Algorithm**:

1. Remove timestamps older than window
2. Check count against limit
3. Add new timestamp if allowed

**Characteristics**:

- True sliding window
- No boundary issues
- O(n) time complexity (n = requests in window)
- 24 + 8n bytes per key
- Most accurate

## Data Flow

### Request Flow (Allow Operation)

```
User Code
    │
    │ limiter.Allow("user123")
    ▼
┌─────────────────────────┐
│  Algorithm.Allow(key)   │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Hash key → shard ID    │  SecureHash(key) % 256
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Acquire shard lock     │  RWMutex (Read for lookup)
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Lookup/Create state    │  map[hash]*State
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Algorithm-specific     │  Token refill, leak, etc.
│  state update           │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Update statistics      │  Atomic increment
└───────────┬─────────────┘
            │
            ├─ Allowed ──→ Return nil
            │
            └─ Denied ───→ Return RateLimitError
```

### Statistics Flow

```
User Code
    │
    │ limiter.Stats()
    ▼
┌─────────────────────────┐
│  Iterate all shards     │  for i := 0; i < 256; i++
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Lock shard (read)      │  RWMutex.RLock()
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Read shard stats       │  len(states), evictions
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Aggregate totals       │  Sum across shards
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Load atomic counters   │  allowed.Load(), denied.Load()
└───────────┬─────────────┘
            │
            ▼
        Return AlgorithmStats
```

## Performance Optimizations

### 1. Sharding

**Problem**: Single lock becomes bottleneck under high concurrency

**Solution**: 256 shards with independent locks

```go
shardID := hash(key) % 256
shard := shards[shardID]
shard.mu.Lock()
// Work on shard
shard.mu.Unlock()
```

**Result**: 256× reduction in lock contention

### 2. Double-Checked Locking

**Pattern**: Minimize write lock acquisitions

```go
// First check with read lock
shard.mu.RLock()
state, exists := shard.states[hash]
shard.mu.RUnlock()

if !exists {
    // Only acquire write lock if needed
    shard.mu.Lock()
    state, exists = shard.states[hash] // Check again!
    if !exists {
        state = createNew()
        shard.states[hash] = state
    }
    shard.mu.Unlock()
}
```

### 3. Atomic Operations

**Hot path**: Use atomics instead of locks where possible

```go
totalAllowed.Add(1)  // Instead of mutex-protected counter
closed.Load()        // Lockless status check
```

### 4. Memory Alignment

**Cache line padding**: Prevent false sharing

```go
type TokenState struct {
    tokens     int64
    lastNs     int64
    rateNs     int64
    burst      int64
    lastAccess int64
    _pad       [24]byte  // Pad to 64 bytes (cache line)
}
```

## Memory Management

### Key Eviction Strategy

**Problem**: Unbounded key growth leads to OOM

**Solution**: TTL-based eviction

```go
evictionTTL := 2 × window
cutoffTime := now - evictionTTL

for hash, state := range shard.states {
    if state.lastAccess < cutoffTime {
        delete(shard.states, hash)
    }
}
```

**Trigger**: When shard exceeds 10,000 keys

**Memory Bounds**:

- Max keys per shard: 10,000
- Total max keys: 256 × 10,000 = 2.56M
- Memory cap: ~120MB - ~200MB depending on algorithm

### Pre-allocation

Shards pre-allocate space for 1,000 keys each:

```go
states: make(map[uint64]*State, maxKeysPerShard/10)
```

Reduces initial allocations and map resizes.

## Concurrency Model

### Thread Safety Guarantees

1. **All operations are thread-safe**
2. **No data races** (verified with `-race`)
3. **Consistent reads** via proper memory ordering

### Lock Hierarchy

```
Level 1: Shard-level RWMutex
    ├─ Read locks for state lookup
    └─ Write locks for state creation

Level 2: State-level Mutex (Leaky/Fixed/Sliding)
    └─ Protects per-key state mutations

Level 3: Atomic variables (all algorithms)
    └─ Statistics, closed flag
```

**Rule**: Never hold multiple shard locks simultaneously (deadlock prevention)

### Goroutine Safety

- **Safe for concurrent Allow() calls** from multiple goroutines
- **Safe to call Stats()** while processing requests
- **Safe to Reset()** specific keys without blocking others
- **ResetAll()** acquires all shard locks sequentially

## Error Handling

### Rich Error Context

```go
type RateLimitError struct {
    Cause      error
    Key        string
    Limit      int64
    Remaining  int64
    RetryAfter time.Duration
    ResetAt    time.Time
    BurstSeen  bool
    ShardID    uint64
}
```

**Usage**:

```go
err := limiter.Allow(key)
if rlErr, ok := err.(*errors.RateLimitError); ok {
    // Extract metadata
    retryAfter := rlErr.RetryAfter
    remaining := rlErr.Remaining
}
```

## Testing Strategy

### Unit Tests

- Algorithm correctness
- Boundary conditions
- Concurrent access
- Reset operations

### Benchmark Tests

```bash
go test ./internal/algorithm/... -bench=. -benchmem
```

Measures:

- Throughput (ops/sec)
- Latency (ns/op)
- Allocations (allocs/op)

### Race Detection

```bash
go test ./internal/algorithm/... -race
```

Verifies thread safety.

## Future Enhancements

1. **Distributed Mode**: Redis-backed state for multi-instance deployments
2. **Adaptive Switching**: Auto-select algorithm based on traffic patterns
3. **Prometheus Metrics**: Built-in exporter
4. **Latency Tracking**: P50, P99, P999 metrics
5. **Circuit Breaker**: Integration with failure detection

---

**Design Philosophy**: "Make the common case fast, make all cases correct."
