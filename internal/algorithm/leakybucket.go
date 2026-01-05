package algorithm

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/ASHISH26940/throttle/internal/errors"
	"github.com/ASHISH26940/throttle/internal/limiter"
	"github.com/ASHISH26940/throttle/internal/types"
	"github.com/ASHISH26940/throttle/internal/utils"
)

const (
	leakyShardCount      = 256
	leakyMaxKeysPerShard = 10000
	leakyEvictionTTLMult = 2
)

type LeakyBucketAlgorithm struct {
	config types.Config
	shards [leakyShardCount]*LeakyBucketShard
	clock  *limiter.Clock
	closed atomic.Int32

	totalAllowed atomic.Int64
	totalDenied  atomic.Int64
}

type LeakyBucketShard struct {
	mu        sync.RWMutex
	states    map[uint64]*LeakyBucketState
	evictions atomic.Int64
}

type LeakyBucketState struct {
	queue      []int64 // Timestamps of requests in queue
	capacity   int64   // Max queue size (burst)
	leakRateNs int64   // Time between leaks (nanoseconds)
	lastLeak   int64   // Last leak timestamp
	lastAccess int64   // For eviction
	mu         sync.Mutex
}

func NewLeakyBucket(cfg types.Config) (*LeakyBucketAlgorithm, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	lb := &LeakyBucketAlgorithm{
		config: cfg,
		clock:  limiter.NewClock(),
	}

	for i := 0; i < leakyShardCount; i++ {
		lb.shards[i] = &LeakyBucketShard{
			states: make(map[uint64]*LeakyBucketState, leakyMaxKeysPerShard/10),
		}
	}

	return lb, nil
}

func (lb *LeakyBucketAlgorithm) Allow(key string) error {
	if lb.closed.Load() != 0 {
		return errors.ErrLimiterClosed
	}

	hash := utils.SecureHash(key)
	shardID := hash % leakyShardCount
	shard := lb.shards[shardID]

	shard.mu.RLock()
	state, exists := shard.states[hash]
	shard.mu.RUnlock()

	if !exists {
		shard.mu.Lock()
		state, exists = shard.states[hash]
		if !exists {
			if len(shard.states) >= leakyMaxKeysPerShard {
				lb.evictOldKeys(shard, lb.clock.Now())
			}
			state = newLeakyBucketState(lb.config, lb.clock)
			shard.states[hash] = state
		}
		shard.mu.Unlock()
	}

	now := lb.clock.Now()
	state.mu.Lock()
	defer state.mu.Unlock()

	// Leak (remove old requests from queue)
	state.leak(now)

	// Check if queue has space
	if int64(len(state.queue)) >= state.capacity {
		lb.totalDenied.Add(1)
		retryNs := state.leakRateNs * (int64(len(state.queue)) - state.capacity + 1)
		return errors.NewRateLimitError(
			key,
			errors.ErrRateLimitExceeded,
			lb.config.Rate,
			state.capacity-int64(len(state.queue)),
			time.Duration(retryNs),
			time.Now().Add(lb.config.Window),
			false,
			shardID,
		)
	}

	// Add request to queue
	state.queue = append(state.queue, now)
	state.lastAccess = now
	lb.totalAllowed.Add(1)

	return nil
}

func newLeakyBucketState(cfg types.Config, clock *limiter.Clock) *LeakyBucketState {
	leakRateNs := cfg.Window.Nanoseconds() / cfg.Rate
	now := clock.Now()

	return &LeakyBucketState{
		queue:      make([]int64, 0, cfg.Burst),
		capacity:   cfg.Burst,
		leakRateNs: leakRateNs,
		lastLeak:   now,
		lastAccess: now,
	}
}

func (s *LeakyBucketState) leak(now int64) {
	if len(s.queue) == 0 {
		s.lastLeak = now
		return
	}

	elapsed := now - s.lastLeak
	if elapsed < 0 {
		elapsed = 0
	}

	leaksToProcess := elapsed / s.leakRateNs
	if leaksToProcess > 0 {
		if leaksToProcess >= int64(len(s.queue)) {
			s.queue = s.queue[:0]
		} else {
			s.queue = s.queue[leaksToProcess:]
		}
		s.lastLeak = now
	}
}

func (lb *LeakyBucketAlgorithm) evictOldKeys(shard *LeakyBucketShard, now int64) {
	cutoffNs := now - (lb.config.Window.Nanoseconds() * leakyEvictionTTLMult)
	evicted := 0

	for hash, state := range shard.states {
		state.mu.Lock()
		lastAccess := state.lastAccess
		state.mu.Unlock()

		if lastAccess < cutoffNs {
			delete(shard.states, hash)
			evicted++
		}
	}
	shard.evictions.Add(int64(evicted))
}

func (lb *LeakyBucketAlgorithm) Type() AlgorithmType {
	return AlgorithmLeakyBucket
}

func (lb *LeakyBucketAlgorithm) Close() error {
	if !lb.closed.CompareAndSwap(0, 1) {
		return errors.ErrLimiterClosed
	}
	return nil
}

func (lb *LeakyBucketAlgorithm) Reset(key string) {
	hash := utils.SecureHash(key)
	shardID := hash % leakyShardCount
	shard := lb.shards[shardID]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if state, exists := shard.states[hash]; exists {
		state.mu.Lock()
		state.queue = state.queue[:0]
		state.lastLeak = lb.clock.Now()
		state.mu.Unlock()
	}
}

func (lb *LeakyBucketAlgorithm) ResetAll() {
	for i := 0; i < leakyShardCount; i++ {
		shard := lb.shards[i]
		shard.mu.Lock()
		shard.states = make(map[uint64]*LeakyBucketState)
		shard.evictions.Store(0)
		shard.mu.Unlock()
	}
	lb.totalAllowed.Store(0)
	lb.totalDenied.Store(0)
}

func (lb *LeakyBucketAlgorithm) Stats() AlgorithmStats {
	totalKeys := int64(0)
	totalEvictions := int64(0)

	for i := 0; i < leakyShardCount; i++ {
		shard := lb.shards[i]
		shard.mu.RLock()
		totalKeys += int64(len(shard.states))
		shard.mu.RUnlock()
		totalEvictions += shard.evictions.Load()
	}

	return AlgorithmStats{
		Type:         AlgorithmLeakyBucket,
		TotalKeys:    totalKeys,
		TotalAllowed: lb.totalAllowed.Load(),
		TotalDenied:  lb.totalDenied.Load(),
		Evictions:    totalEvictions,
		AvgLatencyNs: 0,
	}
}