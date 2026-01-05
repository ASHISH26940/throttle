# Rate Limiting Algorithm Implementation Plan

This plan implements the four core rate limiting algorithms defined in the `algorithm.Algorithm` interface.

## Overview

We need to implement four distinct rate limiting algorithms, each with different characteristics suitable for different traffic patterns:

1. **Token Bucket** - Good for bursty traffic
2. **Leaky Bucket** - Good for smoothing traffic
3. **Fixed Window** - Simple and memory efficient
4. **Sliding Window** - More accurate than fixed window

Each algorithm must implement the `Algorithm` interface:

```go
type Algorithm interface{
    Allow(key string) error
    Type() AlgorithmType
    Close() error
    Reset(key string)
    ResetAll()
    Stats() AlgorithmStats
}
```

## Proposed Changes

### Core Algorithms

#### [NEW] internal/algorithm/tokenbucket_algorithm.go

Wraps the existing `ShardedLimiter` (which uses token bucket) to implement the `Algorithm` interface.

**Features:**

- Delegates to existing `limiter.ShardedLimiter`
- Tracks statistics (allowed/denied counts)
- Thread-safe operations
- Efficient for bursty traffic patterns

**Implementation Details:**

```go
type TokenBucketAlgorithm struct {
    limiter *limiter.ShardedLimiter
    stats   AlgorithmStats
    mu      sync.RWMutex
    closed  atomic.Int32
}
```

---

#### [NEW] internal/algorithm/leakybucket.go

Implements the leaky bucket algorithm using a queue-based approach.

**Features:**

- Request queue with max capacity
- Constant leak rate
- Smooths out traffic bursts
- Sharded for performance
- Statistics tracking

**Algorithm:**

- Incoming requests added to queue
- Queue "leaks" at constant rate
- If queue full, reject request
- Good for enforcing steady output rate

**Implementation Details:**

```go
type LeakyBucketAlgorithm struct {
    config  types.Config
    shards  [256]*LeakyBucketShard
    clock   *limiter.Clock
    closed  atomic.Int32
}

type LeakyBucketShard struct {
    mu        sync.RWMutex
    states    map[uint64]*LeakyBucketState
    evictions atomic.Int64
}

type LeakyBucketState struct {
    queue      []int64  // Timestamps of requests in queue
    capacity   int64
    leakRateNs int64
    lastLeak   int64
}
```

---

#### [NEW] internal/algorithm/fixedwindow.go

Implements fixed window counter algorithm.

**Features:**

- Simple counter per time window
- Window resets at fixed intervals
- Very memory efficient
- Sharded for performance
- Statistics tracking

**Algorithm:**

- Each window has start time and counter
- Allow if counter < limit
- Reset counter when window expires
- Simple but can have burst at window boundaries

**Implementation Details:**

```go
type FixedWindowAlgorithm struct {
    config  types.Config
    shards  [256]*FixedWindowShard
    clock   *limiter.Clock
    closed  atomic.Int32
}

type FixedWindowShard struct {
    mu        sync.RWMutex
    states    map[uint64]*FixedWindowState
    evictions atomic.Int64
}

type FixedWindowState struct {
    windowStart int64
    counter     int64
    limit       int64
}
```

---

#### [NEW] internal/algorithm/slidingwindow.go

Implements sliding window log algorithm.

**Features:**

- Keeps timestamp log of recent requests
- More accurate than fixed window
- Prevents boundary bursts
- Sharded for performance
- Statistics tracking

**Algorithm:**

- Store timestamps of requests in sliding window
- Remove expired timestamps
- Allow if count < limit
- More accurate but higher memory usage

**Implementation Details:**

```go
type SlidingWindowAlgorithm struct {
    config  types.Config
    shards  [256]*SlidingWindowShard
    clock   *limiter.Clock
    closed  atomic.Int32
}

type SlidingWindowShard struct {
    mu        sync.RWMutex
    states    map[uint64]*SlidingWindowState
    evictions atomic.Int64
}

type SlidingWindowState struct {
    timestamps []int64  // Request timestamps within window
    limit      int64
    windowNs   int64
}
```

---

### Factory Pattern

#### [NEW] internal/algorithm/factory.go

Factory for creating algorithm instances.

**Features:**

- Implements `Factory` interface
- Creates algorithms based on config
- Validates configuration per algorithm type
- Returns appropriate algorithm instance

**Implementation Details:**

```go
type DefaultFactory struct{}

func NewFactory() Factory {
    return &DefaultFactory{}
}

func (f *DefaultFactory) Create(cfg types.Config) (Algorithm, error) {
    // Create based on algorithm type in config
    // For now, we'll add an AlgorithmType field to config
}

// Convenience functions
func NewTokenBucket(cfg types.Config) (*TokenBucketAlgorithm, error)
func NewLeakyBucket(cfg types.Config) (*LeakyBucketAlgorithm, error)
func NewFixedWindow(cfg types.Config) (*FixedWindowAlgorithm, error)
func NewSlidingWindow(cfg types.Config) (*SlidingWindowAlgorithm, error)
```

---

### Testing

#### [NEW] internal/algorithm/tokenbucket_algorithm_test.go

Tests for Token Bucket implementation including:

- Basic allow/deny behavior
- Burst handling
- Reset functionality
- Statistics accuracy
- Concurrent access

#### [NEW] internal/algorithm/leakybucket_test.go

Tests for Leaky Bucket implementation including:

- Queue behavior
- Leak rate accuracy
- Capacity limits
- Statistics tracking

#### [NEW] internal/algorithm/fixedwindow_test.go

Tests for Fixed Window implementation including:

- Window transitions
- Counter resets
- Boundary conditions
- Statistics tracking

#### [NEW] internal/algorithm/slidingwindow_test.go

Tests for Sliding Window implementation including:

- Timestamp management
- Window sliding behavior
- Accuracy vs fixed window
- Statistics tracking

#### [NEW] internal/algorithm/factory_test.go

Tests for algorithm factory including:

- Creating each algorithm type
- Configuration validation
- Error handling

#### [NEW] internal/algorithm/benchmark_test.go

Benchmarks comparing all algorithms:

- Throughput comparison
- Memory usage
- Concurrency performance
- Different traffic patterns

## Verification Plan

### Automated Tests

1. **Unit Tests** - Run all algorithm tests:

   ```powershell
   cd d:\throttle
   go test ./internal/algorithm/... -v
   ```

2. **Race Detection** - Ensure thread safety:

   ```powershell
   go test ./internal/algorithm/... -race
   ```

3. **Benchmarks** - Compare performance:

   ```powershell
   go test ./internal/algorithm/... -bench=. -benchmem
   ```

4. **Coverage** - Check test coverage:
   ```powershell
   go test ./internal/algorithm/... -cover
   ```

### Manual Verification

1. **Integration Testing** - Test with selector package
2. **Performance Testing** - Verify each algorithm handles expected load
3. **Memory Testing** - Check memory usage patterns
4. **Traffic Pattern Testing** - Test with different traffic patterns (bursty, steady, spikey)

## Implementation Notes

### Design Decisions

1. **Sharding Strategy**: All algorithms use 256 shards for optimal performance
2. **Statistics**: Each algorithm maintains its own atomic counters
3. **Clock Abstraction**: Uses `limiter.Clock` for testability
4. **Error Handling**: Leverages existing error types from `internal/errors`
5. **Thread Safety**: Combination of atomic operations and mutexes

### Performance Considerations

- **Token Bucket**: O(1) for allow operation, best overall performance
- **Leaky Bucket**: O(1) for allow, O(n) for cleanup where n = queue size
- **Fixed Window**: O(1) for allow operation, very low memory
- **Sliding Window**: O(n) for allow where n = requests in window, higher memory

### Memory Usage

- **Token Bucket**: ~48 bytes per key
- **Leaky Bucket**: ~24 bytes + (8 bytes × queue size) per key
- **Fixed Window**: ~32 bytes per key
- **Sliding Window**: ~24 bytes + (8 bytes × requests in window) per key

### Configuration

Each algorithm will respect the `types.Config` structure:

- `Rate`: Requests allowed per window
- `Window`: Time window duration
- `Burst`: Maximum burst allowance (applies to Token Bucket and Leaky Bucket)

### Integration with Selector

The selector will use the factory to create algorithms and can switch between them based on traffic patterns:

- **Bursty traffic** → Token Bucket
- **Need smooth output** → Leaky Bucket
- **Low memory requirement** → Fixed Window
- **High accuracy requirement** → Sliding Window
