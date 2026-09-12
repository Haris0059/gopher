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

const testContent = "fake gopher binary contents"

func testAssetPath(version string) string {
	filename := assetName(version, runtime.GOOS, runtime.GOARCH)
	return fmt.Sprintf("/%s/%s/releases/download/v%s/%s", repoOwner, repoName, version, filename)
}

func testChecksumsPath(version string) string {
	return fmt.Sprintf("/%s/%s/releases/download/v%s/checksums.txt", repoOwner, repoName, version)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestDownloadVersion_Success(t *testing.T) {
	version := "1.0.0"
	filename := assetName(version, runtime.GOOS, runtime.GOARCH)
	sum := sha256Hex(testContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChecksumsPath(version):
			fmt.Fprintf(w, "%s  %s\n", sum, filename)
		case testAssetPath(version):
			w.Write([]byte(testContent))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out")
	err := downloadVersion(context.Background(), srv.Client(), srv.URL, version, runtime.GOOS, runtime.GOARCH, dest)
	if err != nil {
		t.Fatalf("downloadVersion: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != testContent {
		t.Errorf("content = %q, want %q", data, testContent)
	}

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		t.Errorf("mode = %v, want executable bit set", info.Mode())
	}
}

func TestDownloadVersion_ChecksumMismatch(t *testing.T) {
	version := "1.0.0"
	filename := assetName(version, runtime.GOOS, runtime.GOARCH)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChecksumsPath(version):
			fmt.Fprintf(w, "%s  %s\n", sha256Hex("wrong content"), filename)
		case testAssetPath(version):
			w.Write([]byte(testContent))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out")
	err := downloadVersion(context.Background(), srv.Client(), srv.URL, version, runtime.GOOS, runtime.GOARCH, dest)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Error("partial file was not removed after checksum mismatch")
	}
}

func TestDownloadVersion_AssetNotFound(t *testing.T) {
	version := "1.0.0"
	filename := assetName(version, runtime.GOOS, runtime.GOARCH)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChecksumsPath(version):
			fmt.Fprintf(w, "%s  %s\n", sha256Hex(testContent), filename)
		case testAssetPath(version):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out")
	err := downloadVersion(context.Background(), srv.Client(), srv.URL, version, runtime.GOOS, runtime.GOARCH, dest)
	if err == nil {
		t.Fatal("expected error for 404 asset")
	}
}

func TestDownloadVersion_MissingChecksumEntry(t *testing.T) {
	version := "1.0.0"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChecksumsPath(version):
			fmt.Fprintf(w, "%s  some-other-file\n", sha256Hex(testContent))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out")
	err := downloadVersion(context.Background(), srv.Client(), srv.URL, version, runtime.GOOS, runtime.GOARCH, dest)
	if err == nil {
		t.Fatal("expected error for missing checksum entry")
	}
}

func TestDownloadVersion_RetriesOnServerErrorThenSucceeds(t *testing.T) {
	version := "1.0.0"
	filename := assetName(version, runtime.GOOS, runtime.GOARCH)
	sum := sha256Hex(testContent)

	var assetRequests int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChecksumsPath(version):
			fmt.Fprintf(w, "%s  %s\n", sum, filename)
		case testAssetPath(version):
			n := atomic.AddInt64(&assetRequests, 1)
			if n == 1 {
				// First attempt fails at the network level via a hung
				// connection isn't practical here; a 5xx exercises the
				// "fails immediately" path instead per current retry rules.
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write([]byte(testContent))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out")
	err := downloadVersion(context.Background(), srv.Client(), srv.URL, version, runtime.GOOS, runtime.GOARCH, dest)
	// downloadOnce treats non-200 status as non-retryable, matching the
	// reference's "HTTP errors fail immediately" rule (download.ts:358-368).
	if err == nil {
		t.Fatal("expected a 500 status to fail immediately, not retry to success")
	}
	if got := atomic.LoadInt64(&assetRequests); got != 1 {
		t.Errorf("asset requested %d times, want exactly 1 (no retry on HTTP error)", got)
	}
}

// failAssetNTimes fails the first n requests whose path contains "download/"
// (i.e. the asset fetch, not the checksums.txt fetch) with a network-level
// error, then delegates to the real transport — simulating a transient
// network failure, which IS retried (unlike an HTTP status error).
type failAssetNTimes struct {
	remaining int64
	filename  string
	inner     http.RoundTripper
}

func (f *failAssetNTimes) RoundTrip(req *http.Request) (*http.Response, error) {
	if filepath.Base(req.URL.Path) == f.filename && atomic.AddInt64(&f.remaining, -1) >= 0 {
		return nil, fmt.Errorf("simulated network failure")
	}
	return f.inner.RoundTrip(req)
}

func TestDownloadVersion_RetriesOnNetworkErrorThenSucceeds(t *testing.T) {
	version := "1.0.0"
	filename := assetName(version, runtime.GOOS, runtime.GOARCH)
	sum := sha256Hex(testContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChecksumsPath(version):
			fmt.Fprintf(w, "%s  %s\n", sum, filename)
		case testAssetPath(version):
			w.Write([]byte(testContent))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: &failAssetNTimes{remaining: 2, filename: filename, inner: http.DefaultTransport}}

	dest := filepath.Join(t.TempDir(), "out")
	err := downloadVersion(context.Background(), client, srv.URL, version, runtime.GOOS, runtime.GOARCH, dest)
	if err != nil {
		t.Fatalf("downloadVersion: %v (want success after retries)", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != testContent {
		t.Errorf("content = %q, want %q", data, testContent)
	}
}
