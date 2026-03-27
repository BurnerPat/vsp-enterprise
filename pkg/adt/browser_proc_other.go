//go:build !windows

package adt

import "os/exec"

// setBrowserProcessAttrs is a no-op on non-Windows platforms.
// On Linux/macOS, child processes are not affected by MCP host job objects.
func setBrowserProcessAttrs(cmd *exec.Cmd) {
	// no-op
}
