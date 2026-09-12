package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeVersion creates a fake, non-empty version file with the given mtime.
func writeVersion(t *testing.T, dirs Dirs, version string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(dirs.Versions, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dirs.Versions, version)
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestGC_KeepsNewestRetainCount(t *testing.T) {
	dirs := testDirs(t)
	now := time.Now()

	writeVersion(t, dirs, "1.0.0", now.Add(-4*time.Hour))
	writeVersion(t, dirs, "2.0.0", now.Add(-3*time.Hour))
	writeVersion(t, dirs, "3.0.0", now.Add(-2*time.Hour))
	writeVersion(t, dirs, "4.0.0", now.Add(-1*time.Hour))

	if err := GC(dirs, 2); err != nil {
		t.Fatalf("GC: %v", err)
	}

	assertVersionExists(t, dirs, "4.0.0", true)
	assertVersionExists(t, dirs, "3.0.0", true)
	assertVersionExists(t, dirs, "2.0.0", false)
	assertVersionExists(t, dirs, "1.0.0", false)
}

func TestGC_ProtectsSymlinkTargetEvenWhenOld(t *testing.T) {
	dirs := testDirs(t)
	now := time.Now()

	writeVersion(t, dirs, "1.0.0", now.Add(-10*time.Hour)) // old, but symlinked
	writeVersion(t, dirs, "2.0.0", now.Add(-3*time.Hour))
	writeVersion(t, dirs, "3.0.0", now.Add(-2*time.Hour))
	writeVersion(t, dirs, "4.0.0", now.Add(-1*time.Hour))

	if err := os.MkdirAll(filepath.Dir(dirs.Executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dirs.Versions, "1.0.0"), dirs.Executable); err != nil {
		t.Fatal(err)
	}

	if err := GC(dirs, 2); err != nil {
		t.Fatalf("GC: %v", err)
	}

	assertVersionExists(t, dirs, "1.0.0", true) // protected via symlink
	assertVersionExists(t, dirs, "4.0.0", true)
	assertVersionExists(t, dirs, "3.0.0", true)
	assertVersionExists(t, dirs, "2.0.0", false)
}

func TestGC_ProtectsLiveLockedVersion(t *testing.T) {
	dirs := testDirs(t)
	now := time.Now()

	writeVersion(t, dirs, "1.0.0", now.Add(-10*time.Hour)) // old, but locked
	writeVersion(t, dirs, "2.0.0", now.Add(-3*time.Hour))
	writeVersion(t, dirs, "3.0.0", now.Add(-2*time.Hour))
	writeVersion(t, dirs, "4.0.0", now.Add(-1*time.Hour))

	if err := os.MkdirAll(dirs.Locks, 0o755); err != nil {
		t.Fatal(err)
	}
	info := lockInfo{PID: os.Getpid(), Version: "1.0.0"} // our own pid: always alive
	data, _ := json.Marshal(info)
	if err := os.WriteFile(lockPath(dirs, "1.0.0"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := GC(dirs, 2); err != nil {
		t.Fatalf("GC: %v", err)
	}

	assertVersionExists(t, dirs, "1.0.0", true) // protected via live lock
	assertVersionExists(t, dirs, "2.0.0", false)
}

func TestGC_RemovesStaleTmpLeftoversOnly(t *testing.T) {
	dirs := testDirs(t)
	now := time.Now()

	if err := os.MkdirAll(dirs.Versions, 0o755); err != nil {
		t.Fatal(err)
	}

	old := filepath.Join(dirs.Versions, "1.0.0.tmp.123.456")
	fresh := filepath.Join(dirs.Versions, "1.0.0.tmp.789.999")
	for _, p := range []string{old, fresh} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(old, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// fresh keeps its just-written mtime (< 1h old).

	if err := GC(dirs, 2); err != nil {
		t.Fatalf("GC: %v", err)
	}

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("stale .tmp leftover was not removed")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("fresh .tmp leftover was removed too early")
	}
}

func assertVersionExists(t *testing.T, dirs Dirs, version string, want bool) {
	t.Helper()
	_, err := os.Stat(filepath.Join(dirs.Versions, version))
	got := err == nil
	if got != want {
		t.Errorf("version %s exists = %v, want %v", version, got, want)
	}
}
