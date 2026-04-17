package adt

import "github.com/oisee/vibing-steampunk/pkg/adt/transport"

// Compile-time checks: SidecarManager satisfies the transport-layer interfaces
// so it can be passed to NewAdtJcoConnection without modification.
//
// SidecarLifecycle requires Stop() error.
// SidecarIO requires SendSTDIO(msg map[string]interface{}) (map[string]interface{}, error).
var (
	_ transport.SidecarLifecycle = (*SidecarManager)(nil)
	_ transport.SidecarIO        = (*SidecarManager)(nil)
)

// NOTE: SidecarManager, SidecarConfig, and related types currently live in
// pkg/adt/sidecar.go for backward compatibility. They are planned to move to
// pkg/adt/transport/sidecar.go in a follow-up PR. The interfaces above ensure
// the existing types are already compatible with the new transport layer.
//
// Similarly, JCo discovery types (DiscoverJCoLibs, ValidateJava, etc.) in
// pkg/adt/jco_discovery.go are planned to move to pkg/adt/transport/jco_discovery.go.
