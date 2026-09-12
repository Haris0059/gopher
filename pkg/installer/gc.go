package installer

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// VersionRetentionCount is how many of the newest unprotected versions GC
// keeps. Source: installer.ts:74 VERSION_RETENTION_COUNT = 2
const VersionRetentionCount = 2

// staleAge is how old a .tmp.<pid>.<ts> leftover or staging entry must be
// before GC removes it.
const staleAge = time.Hour

var tmpLeftoverRe = regexp.MustCompile(`\.tmp\.\d+\.\d+$`)

// GC removes old installed versions, keeping the newest retain unprotected
// versions plus anything currently in use. Errors during GC are non-fatal
// in the caller (Install), matching the reference's fire-and-forget cleanup.
// Source: installer.ts:1184-1438 cleanupOldVersions()
func GC(dirs Dirs, retain int) error {
	now := time.Now()

	removeStaleTmp(dirs.Versions, now)
	removeStaleTmp(dirs.Staging, now)
	removeStaleLocks(dirs.Locks, now)

	entries, err := os.ReadDir(dirs.Versions)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	protected := protectedVersions(dirs)

	type candidate struct {
		version string
		modTime time.Time
	}
	var candidates []candidate
	for _, e := range entries {
		if e.IsDir() || tmpLeftoverRe.MatchString(e.Name()) {
			continue
		}
		if protected[e.Name()] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Size() == 0 {
			// Placeholder file for an in-progress install; leave it alone.
			continue
		}
		candidates = append(candidates, candidate{e.Name(), info.ModTime()})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})

	if retain < 0 {
		retain = 0
	}
	if len(candidates) <= retain {
		return nil
	}
	for _, c := range candidates[retain:] {
		os.Remove(filepath.Join(dirs.Versions, c.version))
		os.Remove(lockPath(dirs, c.version))
	}
	return nil
}

// protectedVersions returns the set of version names GC must never delete:
// the version the current process is running from, and the version the
// executable symlink currently points at.
func protectedVersions(dirs Dirs) map[string]bool {
	protected := map[string]bool{}

	if execPath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
			if v, ok := versionUnder(dirs.Versions, resolved); ok {
				protected[v] = true
			}
		}
	}

	if target, err := os.Readlink(dirs.Executable); err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(dirs.Executable), target)
		}
		if v, ok := versionUnder(dirs.Versions, target); ok {
			protected[v] = true
		}
	}

	// Anything currently locked is in use by another process.
	if lockEntries, err := os.ReadDir(dirs.Locks); err == nil {
		for _, e := range lockEntries {
			version := strings.TrimSuffix(e.Name(), ".lock")
			if info, err := readLockInfo(filepath.Join(dirs.Locks, e.Name())); err == nil {
				if isProcessAlive(info.PID) {
					protected[version] = true
				}
			}
		}
	}

	return protected
}

// versionUnder reports whether path is directly inside versionsDir, and if
// so returns its basename (the version).
func versionUnder(versionsDir, path string) (string, bool) {
	dir, file := filepath.Split(path)
	if filepath.Clean(dir) != filepath.Clean(versionsDir) {
		return "", false
	}
	return file, true
}

// removeStaleTmp deletes *.tmp.<pid>.<ts> leftovers in dir older than
// staleAge.
func removeStaleTmp(dir string, now time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !tmpLeftoverRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > staleAge {
			os.RemoveAll(filepath.Join(dir, e.Name()))
		}
	}
}

// removeStaleLocks deletes lockfiles whose holder process is no longer
// running.
func removeStaleLocks(dir string, now time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".lock") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := readLockInfo(path)
		if err != nil {
			continue
		}
		if !isProcessAlive(info.PID) {
			os.Remove(path)
		}
	}
}
