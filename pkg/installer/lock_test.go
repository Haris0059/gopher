package installer

import (
	"encoding/json"
	"os"
	"testing"
)

func TestAcquireLock_ReleaseAllowsReacquire(t *testing.T) {
	dirs := testDirs(t)

	release, err := acquireLock(dirs, "1.0.0")
	if err != nil {
		t.Fatalf("acquireLock: %v", err)
	}
	release()

	release2, err := acquireLock(dirs, "1.0.0")
	if err != nil {
		t.Fatalf("acquireLock after release: %v", err)
	}
	release2()
}

func TestAcquireLock_WritesExpectedShape(t *testing.T) {
	dirs := testDirs(t)

	release, err := acquireLock(dirs, "1.0.0")
	if err != nil {
		t.Fatalf("acquireLock: %v", err)
	}
	defer release()

	info, err := readLockInfo(lockPath(dirs, "1.0.0"))
	if err != nil {
		t.Fatalf("readLockInfo: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", info.PID, os.Getpid())
	}
	if info.Version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", info.Version)
	}
	if info.AcquiredAt == 0 {
		t.Error("AcquiredAt = 0, want a timestamp")
	}
}

func TestIsProcessAlive(t *testing.T) {
	if !isProcessAlive(os.Getpid()) {
		t.Error("isProcessAlive(own pid) = false, want true")
	}
	if isProcessAlive(0) {
		t.Error("isProcessAlive(0) = true, want false")
	}
}

func TestListLocks_ReportsLiveAndStale(t *testing.T) {
	dirs := testDirs(t)

	release, err := acquireLock(dirs, "1.0.0")
	if err != nil {
		t.Fatalf("acquireLock: %v", err)
	}
	defer release()

	stale := lockInfo{PID: 999999999, Version: "0.9.0"}
	data, _ := json.Marshal(stale)
	if err := tryCreateLockfile(lockPath(dirs, "0.9.0"), data); err != nil {
		t.Fatalf("tryCreateLockfile: %v", err)
	}

	locks := ListLocks(dirs)
	if len(locks) != 2 {
		t.Fatalf("expected 2 locks, got %d: %+v", len(locks), locks)
	}
	for _, l := range locks {
		switch l.Version {
		case "1.0.0":
			if !l.IsRunning {
				t.Error("expected 1.0.0 lock to be reported as running")
			}
		case "0.9.0":
			if l.IsRunning {
				t.Error("expected 0.9.0 lock to be reported as stale")
			}
		default:
			t.Errorf("unexpected lock version %q", l.Version)
		}
	}
}

func TestListLocks_EmptyDir(t *testing.T) {
	dirs := testDirs(t)
	if locks := ListLocks(dirs); locks != nil {
		t.Errorf("expected no locks in an empty dir, got %+v", locks)
	}
}

func TestCleanupStaleLocks_RemovesOnlyDead(t *testing.T) {
	dirs := testDirs(t)

	release, err := acquireLock(dirs, "1.0.0")
	if err != nil {
		t.Fatalf("acquireLock: %v", err)
	}
	defer release()

	stale := lockInfo{PID: 999999999, Version: "0.9.0"}
	data, _ := json.Marshal(stale)
	if err := tryCreateLockfile(lockPath(dirs, "0.9.0"), data); err != nil {
		t.Fatalf("tryCreateLockfile: %v", err)
	}

	cleaned := CleanupStaleLocks(dirs)
	if cleaned != 1 {
		t.Errorf("expected 1 stale lock cleaned, got %d", cleaned)
	}
	if _, err := os.Stat(lockPath(dirs, "0.9.0")); !os.IsNotExist(err) {
		t.Error("expected stale lockfile to be removed")
	}
	if _, err := os.Stat(lockPath(dirs, "1.0.0")); err != nil {
		t.Error("expected live lockfile to remain")
	}
}
