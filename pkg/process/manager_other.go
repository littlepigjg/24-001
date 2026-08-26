// Package process: no-op fallbacks for platforms without process groups.
//go:build !unix

package process

import "os/exec"

func setProcessGroup(_ *exec.Cmd) {}

func killProcessGroup(_ *exec.Cmd) error { return nil }
