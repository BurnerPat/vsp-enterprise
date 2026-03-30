package mcp

import "github.com/oisee/vibing-steampunk/internal/config"

// Config is a type alias for the central config.ResolvedConfig.
// This keeps all existing references to mcp.Config compiling while the
// single source of truth lives in internal/config.
type Config = config.ResolvedConfig
