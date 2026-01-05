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
	slidingShardCount      = 256
	slidingMaxKeysPerShard = 10000
	slidingEvictionTTLMult = 2
)

type SlidingWindowAlgorithm struct {
	config types.Config
	shards [slidingShardCount]*SlidingWindowShard
	clock  *limiter.Clock
	closed atomic.Int32

	totalAllowed atomic.Int64
	totalDenied  atomic.Int64
}

type SlidingWindowShard struct {
	mu        sync.RWMutex
	states    map[uint64]*SlidingWindowState
	evictions atomic.Int64
}

type SlidingWindowState struct {
	timestamps []int64 // Request timestamps within window
	limit      int64
	windowNs   int64
	lastAccess int64
	mu         sync.Mutex
}

func NewSlidingWindow(cfg types.Config) (*SlidingWindowAlgorithm, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	sw := &SlidingWindowAlgorithm{
		config: cfg,
		clock:  limiter.NewClock(),
	}

	for i := 0; i < slidingShardCount; i++ {
		sw.shards[i] = &SlidingWindowShard{
			states: make(map[uint64]*SlidingWindowState, slidingMaxKeysPerShard/10),
		}
	}

	return sw, nil
}

func (sw *SlidingWindowAlgorithm) Allow(key string) error {
	if sw.closed.Load() != 0 {
		return errors.ErrLimiterClosed
	}

	hash := utils.SecureHash(key)
	shardID := hash % slidingShardCount
	shard := sw.shards[shardID]

	shard.mu.RLock()
	state, exists := shard.states[hash]
	shard.mu.RUnlock()

	if !exists {
		shard.mu.Lock()
		state, exists = shard.states[hash]
		if !exists {
			if len(shard.states) >= slidingMaxKeysPerShard {
				sw.evictOldKeys(shard, sw.clock.Now())
			}
			state = newSlidingWindowState(sw.config, sw.clock)
			shard.states[hash] = state
		}
		shard.mu.Unlock()
	}

	now := sw.clock.Now()
	state.mu.Lock()
	defer state.mu.Unlock()

	// Remove expired timestamps
	state.removeExpired(now)

	// Check if limit exceeded
	if int64(len(state.timestamps)) >= state.limit {
		sw.totalDenied.Add(1)
		
		// Calculate retry after based on oldest timestamp
		var retryAfter time.Duration
		if len(state.timestamps) > 0 {
			oldestTimestamp := state.timestamps[0]
			retryAfter = time.Duration(oldestTimestamp + state.windowNs - now)
			if retryAfter < 0 {
				retryAfter = 0
			}
		}

		return errors.NewRateLimitError(
			key,
			errors.ErrRateLimitExceeded,
			sw.config.Rate,
			state.limit-int64(len(state.timestamps)),
			retryAfter,
			time.Now().Add(retryAfter),
			false,
			shardID,
		)
	}

	// Add new timestamp
	state.timestamps = append(state.timestamps, now)
	state.lastAccess = now
	sw.totalAllowed.Add(1)

	return nil
}

func newSlidingWindowState(cfg types.Config, clock *limiter.Clock) *SlidingWindowState {
	now := clock.Now()
	return &SlidingWindowState{
		timestamps: make([]int64, 0, cfg.Rate),
		limit:      cfg.Rate,
		windowNs:   cfg.Window.Nanoseconds(),
		lastAccess: now,
	}
}

func (s *SlidingWindowState) removeExpired(now int64) {
	cutoff := now - s.windowNs
	
	// Find first non-expired index
	firstValid := 0
	for i, ts := range s.timestamps {
		if ts > cutoff {
			firstValid = i
			break
		}
	}

	// Remove expired timestamps
	if firstValid > 0 {
		if firstValid >= len(s.timestamps) {
			s.timestamps = s.timestamps[:0]
		} else {
			s.timestamps = s.timestamps[firstValid:]
		}
	}
}

func (sw *SlidingWindowAlgorithm) evictOldKeys(shard *SlidingWindowShard, now int64) {
	cutoffNs := now - (sw.config.Window.Nanoseconds() * slidingEvictionTTLMult)
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

func (sw *SlidingWindowAlgorithm) Type() AlgorithmType {
	return AlgorithmSlidingWindow
}

func (sw *SlidingWindowAlgorithm) Close() error {
	if !sw.closed.CompareAndSwap(0, 1) {
		return errors.ErrLimiterClosed
	}
	return nil
}

func (sw *SlidingWindowAlgorithm) Reset(key string) {
	hash := utils.SecureHash(key)
	shardID := hash % slidingShardCount
	shard := sw.shards[shardID]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if state, exists := shard.states[hash]; exists {
		state.mu.Lock()
		state.timestamps = state.timestamps[:0]
		state.mu.Unlock()
	}
}

func (sw *SlidingWindowAlgorithm) ResetAll() {
	for i := 0; i < slidingShardCount; i++ {
		shard := sw.shards[i]
		shard.mu.Lock()
		shard.states = make(map[uint64]*SlidingWindowState)
		shard.evictions.Store(0)
		shard.mu.Unlock()
	}
	sw.totalAllowed.Store(0)
	sw.totalDenied.Store(0)
}

func (sw *SlidingWindowAlgorithm) Stats() AlgorithmStats {
	totalKeys := int64(0)
	totalEvictions := int64(0)

	for i := 0; i < slidingShardCount; i++ {
		shard := sw.shards[i]
		shard.mu.RLock()
		totalKeys += int64(len(shard.states))
		shard.mu.RUnlock()
		totalEvictions += shard.evictions.Load()
	}

	return AlgorithmStats{
		Type:         AlgorithmSlidingWindow,
		TotalKeys:    totalKeys,
		TotalAllowed: sw.totalAllowed.Load(),
		TotalDenied:  sw.totalDenied.Load(),
		Evictions:    totalEvictions,
		AvgLatencyNs: 0,
	}
}