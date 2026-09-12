//go:build !windows

package installer

import (
	"os"
	"syscall"
)

// isProcessAlive reports whether pid identifies a live process.
// Source: pidLock.ts:82-95 isProcessRunning()
func isProcessAlive(pid int) bool {
	if pid <= 1 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; signal 0 checks liveness without
	// actually signaling.
	return proc.Signal(syscall.Signal(0)) == nil
}
