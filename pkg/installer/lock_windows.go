//go:build windows

package installer

import "os"

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
	// os.FindProcess on Windows actually opens a handle; a nil Signal check
	// isn't meaningful there, so probe via Wait with a zero-length timeout
	// substitute: attempting to signal 0 is unsupported, so fall back to
	// treating a successfully found process as alive. This mirrors the
	// conservative behavior of the reference implementation's Windows path.
	return proc != nil
}
