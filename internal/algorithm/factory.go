package algorithm

import (
	"fmt"

	"github.com/ASHISH26940/throttle/internal/types"
)

type DefaultFactory struct{}

func NewFactory() Factory {
	return &DefaultFactory{}
}

func (f *DefaultFactory) Create(cfg types.Config) (Algorithm, error) {
	// If config has an algorithm type specified, we'd use it here
	// For now, default to TokenBucket
	return NewTokenBucket(cfg)
}

// CreateByType creates an algorithm by explicit type
func CreateByType(algType AlgorithmType, cfg types.Config) (Algorithm, error) {
	switch algType {
	case AlgorithmTokenBucket:
		return NewTokenBucket(cfg)
	case AlgorithmLeakyBucket:
		return NewLeakyBucket(cfg)
	case AlgorithmFixedWindow:
		return NewFixedWindow(cfg)
	case AlgorithmSlidingWindow:
		return NewSlidingWindow(cfg)
	default:
		return nil, fmt.Errorf("unknown algorithm type: %v", algType)
	}
}