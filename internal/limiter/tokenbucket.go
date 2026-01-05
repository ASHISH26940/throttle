package limiter

import (
	"sync/atomic"

	"github.com/ASHISH26940/throttle/internal/types"
)

type TokenState struct {
	tokens     int64
	lastNs     int64
	rateNs     int64
	burst      int64
	lastAccess int64
	_pad       [24]byte
}

func NewTokenState(cfg types.Config, clock *Clock) *TokenState {
	now := clock.Now()
	rateNs := cfg.Window.Nanoseconds() / cfg.Rate

	return &TokenState{
		tokens:     cfg.Burst,
		lastNs:     now,
		rateNs:     rateNs,
		burst:      cfg.Burst,
		lastAccess: now,
	}
}

func (ts *TokenState) RefillAndConsume(clock *Clock) (int64, bool) {
	now := clock.Now()
	lastNs := atomic.LoadInt64(&ts.lastNs)

	elasped := now - lastNs
	if elasped < 0 {
		elasped = 0
	}

	tokensToAdd := elasped / ts.rateNs
	currentTokens := atomic.LoadInt64(&ts.tokens)
	newTokens := currentTokens + tokensToAdd

	if newTokens > ts.burst {
		newTokens = ts.burst
	}

	if newTokens > 0 {
		if atomic.CompareAndSwapInt64(&ts.tokens, currentTokens, newTokens-1) {
			atomic.StoreInt64(&ts.lastNs, now)
			atomic.StoreInt64(&ts.lastAccess, now)
			return newTokens - 1, true
		}
	}

	atomic.StoreInt64(&ts.lastAccess, now)
	return atomic.LoadInt64(&ts.tokens), false
}

func (ts *TokenState) Tokens() int64 {
	return atomic.LoadInt64(&ts.tokens)
}

func (ts *TokenState) Reset(clock *Clock) {
	now := clock.Now()
	atomic.StoreInt64(&ts.tokens, ts.burst)
	atomic.StoreInt64(&ts.lastNs, now)
	atomic.StoreInt64(&ts.lastAccess, now)
}
