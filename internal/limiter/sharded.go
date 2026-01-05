package limiter

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/ASHISH26940/throttle/internal/errors"
	"github.com/ASHISH26940/throttle/internal/types"
	"github.com/ASHISH26940/throttle/internal/utils"
)

const (
	shardCount      = 256   // Fixed shard count (optimal for most workloads)
	maxKeysPerShard = 10000 // OOM protection: 256 * 10K = 2.56M max keys
	evictionTTLMult = 2
)

type ShardedLimiter struct {
	config types.Config
	shards [shardCount]*LimiterShard
	clock  *Clock
	closed atomic.Int32
}

type LimiterShard struct {
	mu        sync.RWMutex
	states    map[uint64]*TokenState
	evictions atomic.Int64
}

func New(cfg types.Config, clock *Clock) (*ShardedLimiter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	sl := &ShardedLimiter{
		config: cfg,
		clock:  clock,
	}

	for i := 0; i < shardCount; i++ {
		sl.shards[i] = &LimiterShard{states: make(map[uint64]*TokenState, maxKeysPerShard/10)}
	}

	return sl, nil
}

func (sl *ShardedLimiter) Allow(key string) error {
	if sl.closed.Load() != 0 {
		return errors.ErrLimiterClosed
	}

	hash := utils.SecureHash(key)
	shardID := hash % shardCount
	shard := sl.shards[shardID]

	shard.mu.RLock()
	state, exists := shard.states[hash]
	shard.mu.RUnlock()

	if !exists {
		shard.mu.Lock()
		state, exists = shard.states[hash]
		if !exists {
			if len(shard.states) >= maxKeysPerShard {
				sl.evictOldKeys(shard, sl.clock.Now())
			}
			state = NewTokenState(sl.config, sl.clock)
			shard.states[hash] = state
		}
		shard.mu.Unlock()
	}

	remaining, allowed := state.RefillAndConsume(sl.clock)
	if allowed {
		return nil
	}

	retryNs := (1 - remaining) * state.rateNs
	if retryNs < 0 {
		retryNs = 0
	}

	return errors.NewRateLimitError(
		key,
		errors.ErrRateLimitExceeded,
		sl.config.Rate,
		remaining,
		time.Duration(retryNs),
		time.Now().Add(sl.config.Window),
		false, // burst detection TODO: v1.1
		shardID,
	)
}

func (sl *ShardedLimiter) evictOldKeys(shard *LimiterShard, now int64) {
	cutoffNs := now - (sl.config.Window.Nanoseconds() * evictionTTLMult)
	evicted := 0

	for hash, state := range shard.states {
		lastAccess := atomic.LoadInt64(&state.lastAccess)
		if lastAccess < cutoffNs {
			delete(shard.states, hash)
			evicted++
		}
	}
	shard.evictions.Add(int64(evicted))
}

func (sl *ShardedLimiter) KeyCount() int {
	total := 0
	for i := 0; i < shardCount; i++ {
		shard := sl.shards[i]
		shard.mu.RLock()
		total += len(shard.states)
		shard.mu.RUnlock()
	}
	return total
}

func (sl *ShardedLimiter) Evictions() int64 {
	total := int64(0)
	for i := 0; i < shardCount; i++ {
		total += sl.shards[i].evictions.Load()
	}
	return total
}

func (sl *ShardedLimiter) Close() error {
	if !sl.closed.CompareAndSwap(0, 1) {
		return errors.ErrLimiterClosed
	}
	return nil
}

func (sl *ShardedLimiter) Reset() {
	for i := 0; i < shardCount; i++ {
		shard := sl.shards[i]
		shard.mu.Lock()
		shard.states = make(map[uint64]*TokenState)
		shard.evictions.Store(0)
		shard.mu.Unlock()
	}
}
