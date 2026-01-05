# Rate Limiting Algorithm Implementation Tasks

## Phase 1: Core Algorithm Implementations

### 1.1 Token Bucket Algorithm

- [ ] Create `internal/algorithm/tokenbucket_algorithm.go`
  - [ ] Define `TokenBucketAlgorithm` struct
  - [ ] Implement `Allow(key string) error` method
  - [ ] Implement `Type() AlgorithmType` method
  - [ ] Implement `Close() error` method
  - [ ] Implement `Reset(key string)` method
  - [ ] Implement `ResetAll()` method
  - [ ] Implement `Stats() AlgorithmStats` method
  - [ ] Add statistics tracking (allowed/denied counters)
  - [ ] Wrap existing `limiter.ShardedLimiter`

### 1.2 Leaky Bucket Algorithm

- [ ] Create `internal/algorithm/leakybucket.go`
  - [ ] Define `LeakyBucketAlgorithm` struct
  - [ ] Define `LeakyBucketShard` struct
  - [ ] Define `LeakyBucketState` struct with queue
  - [ ] Implement `Allow(key string) error` method
    - [ ] Add request to queue
    - [ ] Process leak at constant rate
    - [ ] Check if queue is full
  - [ ] Implement `Type() AlgorithmType` method
  - [ ] Implement `Close() error` method
  - [ ] Implement `Reset(key string)` method
  - [ ] Implement `ResetAll()` method
  - [ ] Implement `Stats() AlgorithmStats` method
  - [ ] Add queue management logic
  - [ ] Add leak rate calculation
  - [ ] Add statistics tracking

### 1.3 Fixed Window Algorithm

- [ ] Create `internal/algorithm/fixedwindow.go`
  - [ ] Define `FixedWindowAlgorithm` struct
  - [ ] Define `FixedWindowShard` struct
  - [ ] Define `FixedWindowState` struct
  - [ ] Implement `Allow(key string) error` method
    - [ ] Check if window expired
    - [ ] Reset counter if needed
    - [ ] Increment and check counter
  - [ ] Implement `Type() AlgorithmType` method
  - [ ] Implement `Close() error` method
  - [ ] Implement `Reset(key string)` method
  - [ ] Implement `ResetAll()` method
  - [ ] Implement `Stats() AlgorithmStats` method
  - [ ] Add window expiration logic
  - [ ] Add statistics tracking

### 1.4 Sliding Window Algorithm

- [ ] Create `internal/algorithm/slidingwindow.go`
  - [ ] Define `SlidingWindowAlgorithm` struct
  - [ ] Define `SlidingWindowShard` struct
  - [ ] Define `SlidingWindowState` struct with timestamp log
  - [ ] Implement `Allow(key string) error` method
    - [ ] Remove expired timestamps
    - [ ] Check request count
    - [ ] Add new timestamp
  - [ ] Implement `Type() AlgorithmType` method
  - [ ] Implement `Close() error` method
  - [ ] Implement `Reset(key string)` method
  - [ ] Implement `ResetAll()` method
  - [ ] Implement `Stats() AlgorithmStats` method
  - [ ] Add timestamp cleanup logic
  - [ ] Add statistics tracking

## Phase 2: Algorithm Factory

### 2.1 Factory Implementation

- [ ] Create `internal/algorithm/factory.go`
  - [ ] Define `DefaultFactory` struct
  - [ ] Implement `Create(cfg types.Config) (Algorithm, error)` method
  - [ ] Add `NewFactory()` constructor
  - [ ] Add `NewTokenBucket(cfg types.Config)` helper
  - [ ] Add `NewLeakyBucket(cfg types.Config)` helper
  - [ ] Add `NewFixedWindow(cfg types.Config)` helper
  - [ ] Add `NewSlidingWindow(cfg types.Config)` helper
  - [ ] Add configuration validation per algorithm

### 2.2 Configuration Updates

- [ ] Update `internal/types/config.go` if needed
  - [ ] Consider adding `AlgorithmType` field (optional)
  - [ ] Ensure config works for all algorithms

## Phase 3: Testing

### 3.1 Token Bucket Tests

- [ ] Create `internal/algorithm/tokenbucket_algorithm_test.go`
  - [ ] Test basic allow/deny behavior
  - [ ] Test burst handling
  - [ ] Test token refill over time
  - [ ] Test `Reset(key)` functionality
  - [ ] Test `ResetAll()` functionality
  - [ ] Test `Stats()` accuracy
  - [ ] Test concurrent access (race conditions)
  - [ ] Test `Close()` behavior

### 3.2 Leaky Bucket Tests

- [ ] Create `internal/algorithm/leakybucket_test.go`
  - [ ] Test queue filling and draining
  - [ ] Test leak rate accuracy
  - [ ] Test capacity limits
  - [ ] Test queue overflow behavior
  - [ ] Test `Reset(key)` functionality
  - [ ] Test `ResetAll()` functionality
  - [ ] Test `Stats()` accuracy
  - [ ] Test concurrent access

### 3.3 Fixed Window Tests

- [ ] Create `internal/algorithm/fixedwindow_test.go`
  - [ ] Test window transitions
  - [ ] Test counter increments
  - [ ] Test window reset behavior
  - [ ] Test boundary conditions
  - [ ] Test `Reset(key)` functionality
  - [ ] Test `ResetAll()` functionality
  - [ ] Test `Stats()` accuracy
  - [ ] Test concurrent access

### 3.4 Sliding Window Tests

- [ ] Create `internal/algorithm/slidingwindow_test.go`
  - [ ] Test timestamp addition and removal
  - [ ] Test window sliding behavior
  - [ ] Test accuracy vs fixed window
  - [ ] Test memory management
  - [ ] Test `Reset(key)` functionality
  - [ ] Test `ResetAll()` functionality
  - [ ] Test `Stats()` accuracy
  - [ ] Test concurrent access

### 3.5 Factory Tests

- [ ] Create `internal/algorithm/factory_test.go`
  - [ ] Test creating Token Bucket
  - [ ] Test creating Leaky Bucket
  - [ ] Test creating Fixed Window
  - [ ] Test creating Sliding Window
  - [ ] Test configuration validation
  - [ ] Test error handling

### 3.6 Benchmark Tests

- [ ] Create `internal/algorithm/benchmark_test.go`
  - [ ] Benchmark Token Bucket throughput
  - [ ] Benchmark Leaky Bucket throughput
  - [ ] Benchmark Fixed Window throughput
  - [ ] Benchmark Sliding Window throughput
  - [ ] Benchmark memory usage for each
  - [ ] Benchmark concurrent access
  - [ ] Compare algorithms under different traffic patterns

## Phase 4: Integration

### 4.1 Selector Integration

- [ ] Update `internal/selector/selector.go`
  - [ ] Use factory to create algorithms
  - [ ] Test algorithm switching
  - [ ] Verify metrics collection works with all algorithms
  - [ ] Test pattern detection triggers correct algorithm

### 4.2 Documentation

- [ ] Add godoc comments to all public types and methods
- [ ] Add usage examples in comments
- [ ] Create example files if needed
- [ ] Update main README.md with algorithm descriptions

## Phase 5: Verification

### 5.1 Automated Testing

- [ ] Run all tests: `go test ./internal/algorithm/... -v`
- [ ] Run race detector: `go test ./internal/algorithm/... -race`
- [ ] Run benchmarks: `go test ./internal/algorithm/... -bench=. -benchmem`
- [ ] Check coverage: `go test ./internal/algorithm/... -cover`
- [ ] Ensure >80% coverage

### 5.2 Performance Verification

- [ ] Verify Token Bucket performance meets requirements
- [ ] Verify Leaky Bucket performance meets requirements
- [ ] Verify Fixed Window performance meets requirements
- [ ] Verify Sliding Window performance meets requirements
- [ ] Compare memory usage across algorithms

### 5.3 Integration Verification

- [ ] Test with selector in automatic mode
- [ ] Test algorithm switching based on traffic patterns
- [ ] Verify no memory leaks
- [ ] Test under high concurrency

## Notes

- All algorithms use 256 shards for performance
- All algorithms must be thread-safe
- Statistics must be updated atomically
- Use existing `limiter.Clock` for time operations
- Use existing error types from `internal/errors`
- Follow existing code style and conventions
