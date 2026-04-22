package adt

import "github.com/oisee/vibing-steampunk/pkg/adt/connection"

// Compile-time checks: SidecarManager satisfies the connection-layer interfaces
// so it can be passed to NewJcoConnection without modification.
//
// SidecarLifecycle requires Stop() error.
// SidecarIO requires SendSTDIO(msg map[string]interface{}) (map[string]interface{}, error).
var (
	_ connection.SidecarLifecycle = (*SidecarManager)(nil)
	_ connection.SidecarIO        = (*SidecarManager)(nil)
)

// NOTE: SidecarManager, SidecarConfig, and related types currently live in
// pkg/adt/sidecar.go. They could move to pkg/adt/connection/sidecar.go in a
// future cleanup. The interfaces above ensure the existing types are compatible
// with the connection layer.
