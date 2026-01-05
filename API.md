# API Reference

Complete API documentation for the throttle library.

## Core Interfaces

### Algorithm Interface

The main interface that all rate limiting algorithms implement.

```go
type Algorithm interface {
    // Allow checks if a request for the given key is allowed.
    // Returns nil if allowed, or RateLimitError if denied.
    Allow(key string) error

    // Type returns the algorithm type identifier.
    Type() AlgorithmType

    // Close releases resources. Safe to call multiple times.
    Close() error

    // Reset clears the state for a specific key.
    Reset(key string)

    // ResetAll clears all state and statistics.
    ResetAll()

    // Stats returns current statistics snapshot.
    Stats() AlgorithmStats
}
```

### Factory Interface

Factory for creating algorithm instances.

```go
type Factory interface {
    // Create creates an algorithm instance from configuration.
    // Returns error if configuration is invalid.
    Create(cfg types.Config) (Algorithm, error)
}
```

## Types

### AlgorithmType

```go
type AlgorithmType int

const (
    AlgorithmTokenBucket   AlgorithmType = iota  // 0
    AlgorithmLeakyBucket                         // 1
    AlgorithmFixedWindow                         // 2
    AlgorithmSlidingWindow                       // 3
)

func (a AlgorithmType) String() string
```

**Methods**:

- `String()`: Returns human-readable name ("TokenBucket", "LeakyBucket", etc.)

### Config

```go
type Config struct {
    Rate   int64         // Requests allowed per Window
    Window time.Duration // Time window duration
    Burst  int64         // Maximum burst allowance
}
```

**Methods**:

```go
// Validate checks configuration validity
func (c Config) Validate() error

// WithDefaults applies defaults to partial config
func (c Config) WithDefaults() Config

// String returns debug representation
func (c Config) String() string
```

**Factory Functions**:

```go
// DefaultConfig returns safe defaults
// Rate=100/sec, Burst=150
func DefaultConfig() Config
```

**Validation Rules**:

- Rate: 0 < Rate ≤ 1,000,000
- Window: 0 < Window ≤ 10 minutes
- Burst: 0 ≤ Burst ≤ 10× Rate

### AlgorithmStats

Statistics snapshot for an algorithm.

```go
type AlgorithmStats struct {
    Type         AlgorithmType // Algorithm type
    TotalKeys    int64         // Active key count
    TotalAllowed int64         // Total allowed requests
    TotalDenied  int64         // Total denied requests
    Evictions    int64         // Total key evictions
    AvgLatencyNs int64         // Average latency (if tracked)
}
```

## Constructor Functions

### NewTokenBucket

```go
func NewTokenBucket(cfg types.Config) (*TokenBucketAlgorithm, error)
```

Creates a new Token Bucket rate limiter.

**Parameters**:

- `cfg`: Configuration (Rate, Window, Burst)

**Returns**:

- `*TokenBucketAlgorithm`: Algorithm instance
- `error`: Validation error if config is invalid

**Example**:

```go
cfg := types.Config{
    Rate:   100,
    Window: time.Second,
    Burst:  150,
}
alg, err := algorithm.NewTokenBucket(cfg)
if err != nil {
    log.Fatal(err)
}
defer alg.Close()
```

### NewLeakyBucket

```go
func NewLeakyBucket(cfg types.Config) (*LeakyBucketAlgorithm, error)
```

Creates a new Leaky Bucket rate limiter.

**Parameters**:

- `cfg`: Configuration (Rate, Window, Burst as queue capacity)

**Returns**:

- `*LeakyBucketAlgorithm`: Algorithm instance
- `error`: Validation error if config is invalid

**Example**:

```go
cfg := types.Config{
    Rate:   100,    // Leak rate
    Window: time.Second,
    Burst:  50,     // Queue capacity
}
alg, err := algorithm.NewLeakyBucket(cfg)
```

### NewFixedWindow

```go
func NewFixedWindow(cfg types.Config) (*FixedWindowAlgorithm, error)
```

Creates a new Fixed Window rate limiter.

**Parameters**:

- `cfg`: Configuration (Rate, Window; Burst not used)

**Returns**:

- `*FixedWindowAlgorithm`: Algorithm instance
- `error`: Validation error if config is invalid

**Example**:

```go
cfg := types.Config{
    Rate:   100,
    Window: time.Second,
    Burst:  0, // Not used
}
alg, err := algorithm.NewFixedWindow(cfg)
```

### NewSlidingWindow

```go
func NewSlidingWindow(cfg types.Config) (*SlidingWindowAlgorithm, error)
```

Creates a new Sliding Window rate limiter.

**Parameters**:

- `cfg`: Configuration (Rate, Window; Burst not used)

**Returns**:

- `*SlidingWindowAlgorithm`: Algorithm instance
- `error`: Validation error if config is invalid

**Example**:

```go
cfg := types.Config{
    Rate:   100,
    Window: time.Second,
    Burst:  0, // Not used
}
alg, err := algorithm.NewSlidingWindow(cfg)
```

### NewFactory

```go
func NewFactory() Factory
```

Creates a new default algorithm factory.

**Returns**:

- `Factory`: Factory instance

**Example**:

```go
factory := algorithm.NewFactory()
limiter, err := factory.Create(cfg)
```

### CreateByType

```go
func CreateByType(algType AlgorithmType, cfg types.Config) (Algorithm, error)
```

Creates an algorithm of a specific type.

**Parameters**:

- `algType`: Desired algorithm type
- `cfg`: Configuration

**Returns**:

- `Algorithm`: Algorithm instance
- `error`: Error if type unknown or config invalid

**Example**:

```go
alg, err := algorithm.CreateByType(
    algorithm.AlgorithmLeakyBucket,
    cfg,
)
```

## Error Types

### RateLimitError

Rich error type returned when rate limit is exceeded.

```go
type RateLimitError struct {
    Cause      error         // Underlying cause
    Key        string        // Rate-limited key
    Limit      int64         // Configured limit
    Remaining  int64         // Remaining capacity
    RetryAfter time.Duration // Time to retry
    ResetAt    time.Time     // Window reset time
    BurstSeen  bool          // Burst detected
    ShardID    uint64        // Shard identifier
}
```

**Methods**:

```go
func (e *RateLimitError) Error() string
func (e *RateLimitError) Unwrap() error
func (e *RateLimitError) Is(target error) bool
func (e *RateLimitError) MarshalJSON() ([]byte, error)
```

**Helper Functions**:

```go
// Check if error is a rate limit error
func IsRateLimit(err error) bool

// Extract retry-after duration
func RetryAfter(err error) time.Duration

// Extract remaining tokens
func RemainingTokens(err error) int64

// Extract rate-limited key
func RateLimitKey(err error) string

// Check if burst was detected
func BurstDetected(err error) bool

// Extract shard ID
func ShardID(err error) uint64
```

**Example**:

```go
err := limiter.Allow("user123")
if errors.IsRateLimit(err) {
    retryAfter := errors.RetryAfter(err)
    remaining := errors.RemainingTokens(err)

    fmt.Printf("Retry in %v, %d remaining\n", retryAfter, remaining)
}
```

### ConfigError

Error for invalid configuration.

```go
type ConfigError struct {
    Field  string      // Invalid field name
    Value  interface{} // Invalid value
    Reason string      // Reason for invalidity
}
```

**Methods**:

```go
func (e *ConfigError) Error() string
func (e *ConfigError) Is(target error) bool
```

**Helper Functions**:

```go
func IsConfigError(err error) bool
func ConfigField(err error) string
```

## Sentinel Errors

Pre-defined error values for common conditions.

```go
var (
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
    ErrLimiterClosed     = errors.New("limiter is closed")
    ErrInvalidRate       = errors.New("rate must be positive")
    ErrInvalidWindow     = errors.New("window duration must be positive")
    ErrInvalidBurst      = errors.New("burst cannot be negative")
)
```

## Best Practices

### Resource Management

Always close algorithm instances:

```go
limiter, err := algorithm.NewTokenBucket(cfg)
if err != nil {
    return err
}
defer limiter.Close() // Always close!
```

### Error Handling

Check for specific error types:

```go
err := limiter.Allow(key)
if err != nil {
    if errors.IsRateLimit(err) {
        // Handle rate limit
        return http.StatusTooManyRequests
    }
    // Handle other errors
    return http.StatusInternalServerError
}
```

### Key Selection

Use stable, unique identifiers:

```go
// Good: User ID
limiter.Allow(userID)

// Good: IP address
limiter.Allow(req.RemoteAddr)

// Good: Composite key
limiter.Allow(fmt.Sprintf("%s:%s", userID, endpoint))

// Bad: Random values
limiter.Allow(uuid.New().String()) // Will never hit limit!
```

### Configuration

Validate configuration before using:

```go
cfg := types.Config{
    Rate:   100,
    Window: time.Second,
    Burst:  150,
}

if err := cfg.Validate(); err != nil {
    log.Fatal("Invalid config:", err)
}
```

### Statistics

Monitor periodically, not on every request:

```go
// Good: Periodic monitoring
ticker := time.NewTicker(1 * time.Minute)
defer ticker.Stop()
for range ticker.C {
    stats := limiter.Stats()
    log.Printf("Stats: %+v", stats)
}

// Bad: Per-request monitoring
stats := limiter.Stats() // Don't do this on every request!
```

---

For more examples, see [EXAMPLES.md](EXAMPLES.md).
