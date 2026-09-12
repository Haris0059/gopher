package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockInfo is the on-disk shape of a per-version lockfile.
// Source: pidLock.ts:54-62
type lockInfo struct {
	PID        int    `json:"pid"`
	Version    string `json:"version"`
	ExecPath   string `json:"execPath"`
	AcquiredAt int64  `json:"acquiredAt"` // epoch ms
}

// lockPath returns the lockfile path for a version.
func lockPath(dirs Dirs, version string) string {
	return filepath.Join(dirs.Locks, version+".lock")
}

// acquireLock takes an exclusive lock on version, stealing it if the holder
// PID is no longer running. It returns a release function to call when done.
// Source: pidLock.ts tryAcquireLock (:238) / isProcessRunning (:82-95),
// reduced from the reference's PID-reuse cross-checks and GrowthBook gating
// to a plain liveness probe.
func acquireLock(dirs Dirs, version string) (release func(), err error) {
	if err := os.MkdirAll(dirs.Locks, 0o755); err != nil {
		return nil, err
	}
	path := lockPath(dirs, version)

	info := lockInfo{
		PID:        os.Getpid(),
		Version:    version,
		AcquiredAt: time.Now().UnixMilli(),
	}
	if execPath, err := os.Executable(); err == nil {
		info.ExecPath = execPath
	}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	if err := tryCreateLockfile(path, data); err != nil {
		if !os.IsExist(err) {
			return nil, err
		}
		// Lock already exists — steal it if stale, else report the holder.
		existing, readErr := readLockInfo(path)
		if readErr == nil && isProcessAlive(existing.PID) {
			return nil, fmt.Errorf("version %s is locked by another install (pid %d)", version, existing.PID)
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove stale lock: %w", err)
		}
		if err := tryCreateLockfile(path, data); err != nil {
			return nil, fmt.Errorf("failed to acquire lock for version %s: %w", version, err)
		}
	}

	return func() { os.Remove(path) }, nil
}

// tryCreateLockfile atomically creates path with the given contents,
// failing with an os.IsExist error if it already exists.
func tryCreateLockfile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		os.Remove(path)
		return writeErr
	}
	return closeErr
}

// readLockInfo reads and parses a lockfile.
func readLockInfo(path string) (lockInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return lockInfo{}, err
	}
	var info lockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return lockInfo{}, err
	}
	return info, nil
}
