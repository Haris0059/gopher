package installer

import (
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
