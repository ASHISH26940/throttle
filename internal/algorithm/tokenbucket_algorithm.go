package algorithm

import (
	"sync"
	"sync/atomic"

	"github.com/ASHISH26940/throttle/internal/limiter"
	"github.com/ASHISH26940/throttle/internal/types"
)

type TokenBucketAlgorithm struct{
	limiter *limiter.ShardedLimiter

	totalKeys atomic.Int64
	totalAllowed atomic.Int64
	totalDenied atomic.Int64

	closed atomic.Int32
	mu sync.RWMutex
}

func NewTokenBucket(cfg types.Config)(*TokenBucketAlgorithm,error){
	clock:=limiter.NewClock()
	shardedLimiter,err:=limiter.New(cfg,clock)

	if err!=nil{
		return nil,err
	}

	return &TokenBucketAlgorithm{
		limiter:shardedLimiter,
	},nil
}

func (t *TokenBucketAlgorithm) Allow(key string) error{
	if t.closed.Load()!=0{
		return nil
	}

	err:=t.limiter.Allow(key)
	if err!=nil{
		t.totalDenied.Add(1)
		return err
	}

	t.totalAllowed.Add(1)
	return nil
}

func (t *TokenBucketAlgorithm) Type()AlgorithmType{
	return AlgorithmTokenBucket
}

func (t *TokenBucketAlgorithm) Close() error{
	if !t.closed.CompareAndSwap(0,1){
		return nil
	}
	return t.limiter.Close()
}

func (t *TokenBucketAlgorithm) Reset(key string) {
	// For token bucket, we don't have per-key reset in ShardedLimiter
	// This would require extending ShardedLimiter or implementing here
	// For now, we'll leave it as a no-op
}

func (t *TokenBucketAlgorithm) ResetAll(){
	t.limiter.Reset()
	t.totalAllowed.Store(0)
	t.totalDenied.Store(0)
}

func (t *TokenBucketAlgorithm) Stats() AlgorithmStats{
	return AlgorithmStats{
		Type:         AlgorithmTokenBucket,
		TotalKeys:    int64(t.limiter.KeyCount()),
		TotalAllowed: t.totalAllowed.Load(),
		TotalDenied:  t.totalDenied.Load(),
		Evictions:    t.limiter.Evictions(),
		AvgLatencyNs: 0,
	}
}