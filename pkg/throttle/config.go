package throttle

import "github.com/ASHISH26940/throttle/internal/types"

// Config is the public API - re-exported from internal/types
type Config = types.Config

// DefaultConfig returns safe defaults.
var DefaultConfig = types.DefaultConfig
