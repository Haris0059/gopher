package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
)

// newTestServer serves the GitHub releases API and download endpoints for
// a single version with fixed content.
func newTestServer(t *testing.T, version, content string) *httptest.Server {
	t.Helper()
	filename := assetName(version, runtime.GOOS, runtime.GOARCH)
	sum := sha256.Sum256([]byte(content))
	sumHex := hex.EncodeToString(sum[:])

	var assetRequests int64
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/releases/latest", repoOwner, repoName), func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v%s"}`, version)
	})
	mux.HandleFunc(testChecksumsPath(version), func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", sumHex, filename)
	})
	mux.HandleFunc(testAssetPath(version), func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&assetRequests, 1)
		w.Write([]byte(content))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// testDirs builds a Dirs rooted in separate temp directories for versions
// and staging, so the copy-then-rename path (not a same-filesystem rename)
// is exercised.
func testDirs(t *testing.T) Dirs {
	t.Helper()
	root := t.TempDir()
	return Dirs{
		Versions:   filepath.Join(root, "versions"),
		Staging:    filepath.Join(root, "staging"),
		Locks:      filepath.Join(root, "locks"),
		Executable: filepath.Join(root, "bin", BinaryName()),
	}
}

func TestInstall_FullFlow(t *testing.T) {
	version := "1.0.0"
	content := "gopher-binary-v1"
	srv := newTestServer(t, version, content)
	dirs := testDirs(t)

	result, err := Install(context.Background(), Options{
		Version:         version,
		Dirs:            dirs,
		HTTPClient:      srv.Client(),
		APIBaseURL:      srv.URL,
		DownloadBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if result.Version != version {
		t.Errorf("Version = %s, want %s", result.Version, version)
	}
	if !result.Downloaded {
		t.Error("Downloaded = false on first install, want true")
	}
	if result.InstalledPath != filepath.Join(dirs.Versions, version) {
		t.Errorf("InstalledPath = %s", result.InstalledPath)
	}
	if result.ExecutablePath != dirs.Executable {
		t.Errorf("ExecutablePath = %s, want %s", result.ExecutablePath, dirs.Executable)
	}

	info, err := os.Stat(dirs.Versions + "/" + version)
	if err != nil {
		t.Fatalf("Stat installed version: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o755 {
		t.Errorf("installed file mode = %v, want 0755", info.Mode().Perm())
	}

	if runtime.GOOS != "windows" {
		target, err := os.Readlink(dirs.Executable)
		if err != nil {
			t.Fatalf("Readlink: %v", err)
		}
		if target != result.InstalledPath {
			t.Errorf("symlink target = %s, want %s", target, result.InstalledPath)
		}
	}

	data, err := os.ReadFile(dirs.Executable)
	if err != nil {
		t.Fatalf("ReadFile executable: %v", err)
	}
	if string(data) != content {
		t.Errorf("executable content = %q, want %q", data, content)
	}
}

func TestInstall_SecondInstallSkipsDownload(t *testing.T) {
	version := "1.0.0"
	srv := newTestServer(t, version, "content")
	dirs := testDirs(t)
	opts := Options{Version: version, Dirs: dirs, HTTPClient: srv.Client(), APIBaseURL: srv.URL, DownloadBaseURL: srv.URL}

	if _, err := Install(context.Background(), opts); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	result, err := Install(context.Background(), opts)
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}
	if result.Downloaded {
		t.Error("Downloaded = true on second install of same version, want false")
	}
}

func TestInstall_ForceRedownloads(t *testing.T) {
	version := "1.0.0"
	srv := newTestServer(t, version, "content")
	dirs := testDirs(t)
	opts := Options{Version: version, Dirs: dirs, HTTPClient: srv.Client(), APIBaseURL: srv.URL, DownloadBaseURL: srv.URL}

	if _, err := Install(context.Background(), opts); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	opts.Force = true
	result, err := Install(context.Background(), opts)
	if err != nil {
		t.Fatalf("forced Install: %v", err)
	}
	if !result.Downloaded {
		t.Error("Downloaded = false with Force: true, want true")
	}
}

func TestInstall_ZeroBytePlaceholderTriggersDownload(t *testing.T) {
	version := "1.0.0"
	srv := newTestServer(t, version, "content")
	dirs := testDirs(t)

	if err := os.MkdirAll(dirs.Versions, 0o755); err != nil {
		t.Fatal(err)
	}
	// Simulate the reference's zero-byte placeholder for an incomplete install.
	if err := os.WriteFile(filepath.Join(dirs.Versions, version), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Install(context.Background(), Options{
		Version: version, Dirs: dirs, HTTPClient: srv.Client(), APIBaseURL: srv.URL, DownloadBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !result.Downloaded {
		t.Error("Downloaded = false for a zero-byte placeholder, want true")
	}
}

func TestInstall_LockHeldByLiveProcess(t *testing.T) {
	version := "1.0.0"
	srv := newTestServer(t, version, "content")
	dirs := testDirs(t)

	release, err := acquireLock(dirs, version)
	if err != nil {
		t.Fatalf("acquireLock: %v", err)
	}
	defer release()

	_, err = Install(context.Background(), Options{
		Version: version, Dirs: dirs, HTTPClient: srv.Client(), APIBaseURL: srv.URL, DownloadBaseURL: srv.URL,
	})
	if err == nil {
		t.Fatal("expected error when lock is held by a live process")
	}
}

func TestInstall_LockHeldByDeadProcessIsStolen(t *testing.T) {
	version := "1.0.0"
	srv := newTestServer(t, version, "content")
	dirs := testDirs(t)

	if err := os.MkdirAll(dirs.Locks, 0o755); err != nil {
		t.Fatal(err)
	}
	// A PID that (almost certainly) doesn't correspond to a running process.
	stale := []byte(`{"pid":999999,"version":"1.0.0","execPath":"","acquiredAt":0}`)
	if err := os.WriteFile(lockPath(dirs, version), stale, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Install(context.Background(), Options{
		Version: version, Dirs: dirs, HTTPClient: srv.Client(), APIBaseURL: srv.URL, DownloadBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("Install with stale lock: %v", err)
	}
	if !result.Downloaded {
		t.Error("expected download to proceed after stealing stale lock")
	}
}

func TestInstall_StrayEmptyDirAtExecutablePathIsCleared(t *testing.T) {
	version := "1.0.0"
	srv := newTestServer(t, version, "content")
	dirs := testDirs(t)

	if err := os.MkdirAll(dirs.Executable, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Install(context.Background(), Options{
		Version: version, Dirs: dirs, HTTPClient: srv.Client(), APIBaseURL: srv.URL, DownloadBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if result.ExecutablePath != dirs.Executable {
		t.Errorf("ExecutablePath = %s", result.ExecutablePath)
	}
}

func TestInstall_UnwritableBinDirReturnsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, permission checks don't apply")
	}
	version := "1.0.0"
	srv := newTestServer(t, version, "content")
	dirs := testDirs(t)

	binParent := filepath.Dir(dirs.Executable)
	if err := os.MkdirAll(binParent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(binParent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(binParent, 0o755) })

	_, err := Install(context.Background(), Options{
		Version: version, Dirs: dirs, HTTPClient: srv.Client(), APIBaseURL: srv.URL, DownloadBaseURL: srv.URL,
	})
	if err == nil {
		t.Fatal("expected error installing into an unwritable bin dir")
	}
}
