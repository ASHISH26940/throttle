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
	fixedShardCount      = 256
	fixedMaxKeysPerShard = 10000
	fixedEvictionTTLMult = 2
)

type FixedWindowAlgorithm struct {
	config types.Config
	shards [fixedShardCount]*FixedWindowShard
	clock  *limiter.Clock
	closed atomic.Int32

	totalAllowed atomic.Int64
	totalDenied  atomic.Int64
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
	windowNs    int64
	lastAccess  int64
	mu          sync.Mutex
}

func NewFixedWindow(cfg types.Config) (*FixedWindowAlgorithm, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	fw := &FixedWindowAlgorithm{
		config: cfg,
		clock:  limiter.NewClock(),
	}

	for i := 0; i < fixedShardCount; i++ {
		fw.shards[i] = &FixedWindowShard{
			states: make(map[uint64]*FixedWindowState, fixedMaxKeysPerShard/10),
		}
	}

	return fw, nil
}

func (fw *FixedWindowAlgorithm) Allow(key string) error {
	if fw.closed.Load() != 0 {
		return errors.ErrLimiterClosed
	}

	hash := utils.SecureHash(key)
	shardID := hash % fixedShardCount
	shard := fw.shards[shardID]

	shard.mu.RLock()
	state, exists := shard.states[hash]
	shard.mu.RUnlock()

	if !exists {
		shard.mu.Lock()
		state, exists = shard.states[hash]
		if !exists {
			if len(shard.states) >= fixedMaxKeysPerShard {
				fw.evictOldKeys(shard, fw.clock.Now())
			}
			state = newFixedWindowState(fw.config, fw.clock)
			shard.states[hash] = state
		}
		shard.mu.Unlock()
	}

	now := fw.clock.Now()
	state.mu.Lock()
	defer state.mu.Unlock()

	// Check if window expired and reset if needed
	if now >= state.windowStart+state.windowNs {
		state.windowStart = now
		state.counter = 0
	}

	// Check if limit exceeded
	if state.counter >= state.limit {
		fw.totalDenied.Add(1)
		resetTime := state.windowStart + state.windowNs
		retryAfter := time.Duration(resetTime - now)
		
		return errors.NewRateLimitError(
			key,
			errors.ErrRateLimitExceeded,
			fw.config.Rate,
			state.limit-state.counter,
			retryAfter,
			time.Now().Add(retryAfter),
			false,
			shardID,
		)
	}

	state.counter++
	state.lastAccess = now
	fw.totalAllowed.Add(1)

	return nil
}

func newFixedWindowState(cfg types.Config, clock *limiter.Clock) *FixedWindowState {
	now := clock.Now()
	return &FixedWindowState{
		windowStart: now,
		counter:     0,
		limit:       cfg.Rate,
		windowNs:    cfg.Window.Nanoseconds(),
		lastAccess:  now,
	}
}

func (fw *FixedWindowAlgorithm) evictOldKeys(shard *FixedWindowShard, now int64) {
	cutoffNs := now - (fw.config.Window.Nanoseconds() * fixedEvictionTTLMult)
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

func (fw *FixedWindowAlgorithm) Type() AlgorithmType {
	return AlgorithmFixedWindow
}

func (fw *FixedWindowAlgorithm) Close() error {
	if !fw.closed.CompareAndSwap(0, 1) {
		return errors.ErrLimiterClosed
	}
	return nil
}

func (fw *FixedWindowAlgorithm) Reset(key string) {
	hash := utils.SecureHash(key)
	shardID := hash % fixedShardCount
	shard := fw.shards[shardID]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if state, exists := shard.states[hash]; exists {
		state.mu.Lock()
		state.counter = 0
		state.windowStart = fw.clock.Now()
		state.mu.Unlock()
	}
}

func (fw *FixedWindowAlgorithm) ResetAll() {
	for i := 0; i < fixedShardCount; i++ {
		shard := fw.shards[i]
		shard.mu.Lock()
		shard.states = make(map[uint64]*FixedWindowState)
		shard.evictions.Store(0)
		shard.mu.Unlock()
	}
	fw.totalAllowed.Store(0)
	fw.totalDenied.Store(0)
}

func (fw *FixedWindowAlgorithm) Stats() AlgorithmStats {
	totalKeys := int64(0)
	totalEvictions := int64(0)

	for i := 0; i < fixedShardCount; i++ {
		shard := fw.shards[i]
		shard.mu.RLock()
		totalKeys += int64(len(shard.states))
		shard.mu.RUnlock()
		totalEvictions += shard.evictions.Load()
	}

	return AlgorithmStats{
		Type:         AlgorithmFixedWindow,
		TotalKeys:    totalKeys,
		TotalAllowed: fw.totalAllowed.Load(),
		TotalDenied:  fw.totalDenied.Load(),
		Evictions:    totalEvictions,
		AvgLatencyNs: 0,
	}
}