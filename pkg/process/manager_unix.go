// Package process: unix-specific helpers for process-group management.
//go:build unix

package process

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts the child in its own process group (PGID == child PID)
// so the entire group can be signalled at once when a timeout fires. This
// prevents orphaned grandchildren (e.g. `sleep` under `/bin/sh -c`) from
// outliving the cancelled parent and blocking cmd.Run().
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends SIGKILL to the process group led by cmd. A negative
// PID targets the whole group. Safe to call only after the process has started
// (cmd.Process != nil).
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Negative PID signals the process group whose PGID == |pid|.
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
