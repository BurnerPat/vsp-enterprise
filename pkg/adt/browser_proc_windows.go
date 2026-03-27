package adt

import (
	"os/exec"
	"syscall"
)

// setBrowserProcessAttrs configures Windows-specific process creation flags.
//
// CREATE_NEW_PROCESS_GROUP (0x200): Detaches from the parent's console group
// so Ctrl+C signals aren't forwarded to Chrome.
//
// CREATE_BREAKAWAY_FROM_JOB (0x1000000): Escapes the parent's Windows Job Object.
// VS Code assigns all child processes to a job object and terminates them when
// managing process lifecycles. Without this flag, Edge is killed immediately
// after spawning because VS Code's job object terminates it.
func setBrowserProcessAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x01000000, // CREATE_BREAKAWAY_FROM_JOB
	}
}
