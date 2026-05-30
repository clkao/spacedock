// ABOUTME: Unix test helpers for liveness checks used by the cancel-cleanup test.
//go:build unix

package status

import "syscall"

// processAlive reports whether a pid is still alive (signal 0 probes existence).
func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// killPid best-effort terminates a pid (cleanup if an assertion fails).
func killPid(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
