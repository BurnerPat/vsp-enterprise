package mcp

import "github.com/oisee/vibing-steampunk/internal/config"

// Config is a type alias for the per-system resolved configuration.
// It is consumed by newSystemInstance and stored on each System.
type Config = config.SystemResolvedConfig

// GlobalConfig is a type alias for the top-level runtime configuration.
// It holds global settings and the map of all resolved systems.
type GlobalConfig = config.ResolvedConfig
