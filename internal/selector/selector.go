package selector

import (
	"sync/atomic"
	"time"
	
	"github.com/ASHISH26940/throttle/internal/algorithm"
	"github.com/ASHISH26940/throttle/internal/metrics"
	"github.com/ASHISH26940/throttle/internal/types"
)

type SelectionStrategy int

const (
	StrategyAutomatic SelectionStrategy = iota // Adaptive based on metrics
	StrategyManual                              // User-specified, no switching
	StrategyHybrid                              // Auto with manual override
)

type SelectorConfig struct{
	Strategy SelectionStrategy
	EvaluationWindow time.Duration
	SwitchThreshold float64
	ManualAlgorithm algorithm.AlgorithmType
}

func DefaultSelectorConfig() SelectorConfig{
	return SelectorConfig{
		Strategy:         StrategyAutomatic,
		EvaluationWindow: 10 * time.Second,
		SwitchThreshold:  0.15,
		ManualAlgorithm:  algorithm.AlgorithmTokenBucket,
	}
}

type Selector struct{
	config      SelectorConfig
	current     atomic.Value 
	
	metrics     *metrics.TrafficMetrics
	
	algorithms  map[algorithm.AlgorithmType]algorithm.Algorithm

	lastEval    atomic.Int64
	switchCount atomic.Int64 
}

func NewSelector(cfg SelectorConfig,limiterCfg types.Config)(*Selector,error){
	s:=&Selector{
		config: cfg,
		metrics:metrics.NewTrafficMetrics(cfg.EvaluationWindow),
		algorithms: make(map[algorithm.AlgorithmType]algorithm.Algorithm),
	}

	if cfg.Strategy ==StrategyManual{
		s.current.Store(cfg.ManualAlgorithm)
	}else{
		s.current.Store(algorithm.AlgorithmTokenBucket)
	}

	s.lastEval.Store(time.Now().UnixNano())

	return s,nil
}