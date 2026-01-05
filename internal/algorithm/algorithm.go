package algorithm

import (
	"github.com/ASHISH26940/throttle/internal/types"
)

type AlgorithmType int

const (
	AlgorithmTokenBucket AlgorithmType = iota
	AlgorithmLeakyBucket
	AlgorithmFixedWindow
	AlgorithmSlidingWindow
)

func (a AlgorithmType) String() string{
	switch a {
	case AlgorithmTokenBucket:
		return "TokenBucket"
	case AlgorithmLeakyBucket:
		return "LeakyBucket"
	case AlgorithmFixedWindow:
		return "FixedWindow"
	case AlgorithmSlidingWindow:
		return "SlidingWindow"
	default:
		return "Unknown"
	}
}

type Algorithm interface{
	Allow(key string)error
	Type()AlgorithmType
	Close()error
	Reset(key string)
	ResetAll()
	Stats()AlgorithmStats
}

type AlgorithmStats struct{
	Type AlgorithmType
	TotalKeys int64
	TotalAllowed int64
	TotalDenied int64
	Evictions int64
	AvgLatencyNs int64
}

type Factory interface{
	Create(cfg types.Config) (Algorithm,error)
}

type AlgorithmMetrics struct{
	Type AlgorithmType
	LastEvalNs int64
	P50LatencyNs int64
	P99LatencyNs int64
	ThroughputRPS float64
	MemoryBytes int64
	Score float64
}