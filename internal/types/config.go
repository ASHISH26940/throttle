package types

import (
	"fmt"
	"time"

	"github.com/ASHISH26940/throttle/internal/errors"
)

type Config struct {
	Rate   int64         `json:"rate"`   // Requests allowed per Window
	Window time.Duration `json:"window"` // Time window for Rate
	Burst  int64         `json:"burst"`  // Maximum burst allowance

	// Internal (unexported)
	MaxKeysPerShard int // Max states per shard before eviction
}

// DefaultConfig returns safe defaults.
func DefaultConfig() Config {
	return Config{
		Rate:            100,
		Window:          time.Second,
		Burst:           150,
		MaxKeysPerShard: 10000,
	}
}

// Validate performs comprehensive configuration validation.
func (c Config) Validate() error {
	if c.Rate <= 0 {
		return errors.NewConfigError("Rate", c.Rate, "must be > 0")
	}
	if c.Rate > 1000000 {
		return errors.NewConfigError("Rate", c.Rate, "max 1M/sec")
	}

	if c.Window <= 0 {
		return errors.NewConfigError("Window", c.Window, "must be > 0")
	}
	if c.Window > 10*time.Minute {
		return errors.NewConfigError("Window", c.Window, "max 10m (memory safety)")
	}

	if c.Burst < 0 {
		return errors.NewConfigError("Burst", c.Burst, "must be >= 0")
	}
	if c.Burst > c.Rate*10 {
		return errors.NewConfigError("Burst", c.Burst, "max 10x Rate")
	}

	return nil
}

// WithDefaults applies safe defaults to partial config.
func (c Config) WithDefaults() Config {
	cfg := DefaultConfig()
	if c.Rate > 0 {
		cfg.Rate = c.Rate
	}
	if c.Window > 0 {
		cfg.Window = c.Window
	}
	if c.Burst >= 0 {
		cfg.Burst = c.Burst
	}
	return cfg
}

// String for debug/logging.
func (c Config) String() string {
	return fmt.Sprintf("Rate=%d/%v Burst=%d", c.Rate, c.Window, c.Burst)
}
