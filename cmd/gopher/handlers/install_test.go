package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Haris0059/gopher/pkg/installer"
)

// newFakeReleaseServer serves a single fake release good enough for
// installer.Install: a GitHub-releases-API "latest" response, the platform
// asset, and its checksums.txt. version is the tag without a leading "v".
func newFakeReleaseServer(t *testing.T, version, content string) *httptest.Server {
	t.Helper()
	filename := fmt.Sprintf("gopher_%s_%s_%s", version, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}
	sum := sha256.Sum256([]byte(content))
	sumHex := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Haris0059/gopher/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v%s"}`, version)
	})
	mux.HandleFunc(fmt.Sprintf("/Haris0059/gopher/releases/download/v%s/checksums.txt", version), func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", sumHex, filename)
	})
	mux.HandleFunc(fmt.Sprintf("/Haris0059/gopher/releases/download/v%s/%s", version, filename), func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(content))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func testDirs(t *testing.T) installer.Dirs {
	t.Helper()
	root := t.TempDir()
	return installer.Dirs{
		Versions:   filepath.Join(root, "versions"),
		Staging:    filepath.Join(root, "staging"),
		Locks:      filepath.Join(root, "locks"),
		Executable: filepath.Join(root, "bin", installer.BinaryName()),
	}
}

func TestInstall_ForceFlag(t *testing.T) {
	var calledForce bool
	installer := func(target string, force bool) (string, error) {
		calledForce = force
		return "Install complete (forced)", nil
	}

	var buf bytes.Buffer
	code := Install(InstallOpts{
		Target:    "",
		Force:     true,
		Output:    &buf,
		Installer: installer,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !calledForce {
		t.Error("--force flag was not passed to installer")
	}
}

func TestInstall_TargetPassed(t *testing.T) {
	var capturedTarget string
	installer := func(target string, force bool) (string, error) {
		capturedTarget = target
		return "Install complete", nil
	}

	var buf bytes.Buffer
	Install(InstallOpts{
		Target:    "alpha",
		Force:     false,
		Output:    &buf,
		Installer: installer,
	})
	if capturedTarget != "alpha" {
		t.Errorf("expected target %q, got %q", "alpha", capturedTarget)
	}
}

func TestInstall_FailureDetection(t *testing.T) {
	t.Run("result containing failed returns exit 1", func(t *testing.T) {
		installer := func(target string, force bool) (string, error) {
			return "shell integration failed to install", nil
		}
		var buf bytes.Buffer
		code := Install(InstallOpts{
			Output:    &buf,
			Installer: installer,
		})
		if code != 1 {
			t.Errorf("expected exit 1 for failed result, got %d", code)
		}
	})

	t.Run("error returns exit 1", func(t *testing.T) {
		installer := func(target string, force bool) (string, error) {
			return "", fmt.Errorf("native installer not found")
		}
		var buf bytes.Buffer
		code := Install(InstallOpts{
			Output:    &buf,
			Installer: installer,
		})
		if code != 1 {
			t.Errorf("expected exit 1 for error, got %d", code)
		}
		if !strings.Contains(buf.String(), "Install failed") {
			t.Errorf("expected failure message in output, got: %s", buf.String())
		}
	})

	t.Run("success returns exit 0", func(t *testing.T) {
		installer := func(target string, force bool) (string, error) {
			return "Install complete", nil
		}
		var buf bytes.Buffer
		code := Install(InstallOpts{
			Output:    &buf,
			Installer: installer,
		})
		if code != 0 {
			t.Errorf("expected exit 0, got %d", code)
		}
	})
}

func TestInstall_DefaultInstaller(t *testing.T) {
	srv := newFakeReleaseServer(t, "1.2.3", "fake gopher binary")
	dirs := testDirs(t)

	var buf bytes.Buffer
	code := Install(InstallOpts{
		Target:          "1.2.3",
		Force:           true,
		Output:          &buf,
		APIBaseURL:      srv.URL,
		DownloadBaseURL: srv.URL,
		Dirs:            dirs,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\noutput:\n%s", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "1.2.3") {
		t.Errorf("expected version in output, got: %s", out)
	}
	if !strings.Contains(out, "Install complete") {
		t.Errorf("expected completion message, got: %s", out)
	}
}

func TestInstall_DefaultInstaller_InvalidChannel(t *testing.T) {
	var buf bytes.Buffer
	code := Install(InstallOpts{
		Target: "beta", // not "latest"/"stable"/semver
		Output: &buf,
		Dirs:   testDirs(t),
	})
	if code != 1 {
		t.Fatalf("expected exit 1 for an invalid channel, got %d\noutput:\n%s", code, buf.String())
	}
	if !strings.Contains(buf.String(), "invalid channel") {
		t.Errorf("expected invalid-channel message, got: %s", buf.String())
	}
}
