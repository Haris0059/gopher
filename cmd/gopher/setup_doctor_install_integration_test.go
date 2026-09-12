package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSetupTokenSubcommand_Integration verifies that `gopher setup-token`
// runs successfully and prints the expected starting message.
func TestSetupTokenSubcommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bin := buildBinary(t)
	cmd := exec.Command(bin, "setup-token")
	cmd.Dir = t.TempDir()
	// Clear env vars that would trigger auth-conflict warnings, so we get
	// a clean starting message.
	cmd.Env = append(os.Environ(),
		"ANTHROPIC_API_KEY=",
		"ANTHROPIC_AUTH_TOKEN=",
		"CLAUDE_CODE_USE_BEDROCK=",
		"CLAUDE_CODE_USE_VERTEX=",
		"CLAUDE_CODE_USE_FOUNDRY=",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gopher setup-token failed: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "long-lived") {
		t.Errorf("expected starting message containing 'long-lived', got:\n%s", got)
	}
}

// TestSetupTokenSubcommand_AuthWarning verifies that `gopher setup-token`
// prints an auth-conflict warning when ANTHROPIC_API_KEY is set.
func TestSetupTokenSubcommand_AuthWarning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bin := buildBinary(t)
	cmd := exec.Command(bin, "setup-token")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY=sk-test-key")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gopher setup-token failed: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "Warning") {
		t.Errorf("expected auth-conflict warning, got:\n%s", got)
	}
}

// TestDoctorSubcommand_Integration verifies that `gopher doctor`
// runs successfully and prints the diagnostics message.
func TestDoctorSubcommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bin := buildBinary(t)
	cmd := exec.Command(bin, "doctor")
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gopher doctor failed: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "Running diagnostics") {
		t.Errorf("expected 'Running diagnostics' in output, got:\n%s", got)
	}
}

// newFakeReleaseServer serves a single fake gopher release: a GitHub
// releases-API "latest" response, the platform asset, and its checksums.txt.
// version is the tag without a leading "v".
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

// installEnv builds an isolated HOME/XDG environment plus the
// GOPHER_INSTALL_*_BASE_URL overrides pointing at srv, so `gopher install`
// never touches the real filesystem or GitHub.
func installEnv(t *testing.T, srv *httptest.Server) []string {
	t.Helper()
	home := t.TempDir()
	return append(os.Environ(),
		"HOME="+home,
		"XDG_DATA_HOME="+filepath.Join(home, "data"),
		"XDG_CACHE_HOME="+filepath.Join(home, "cache"),
		"XDG_STATE_HOME="+filepath.Join(home, "state"),
		"GOPHER_INSTALL_API_BASE_URL="+srv.URL,
		"GOPHER_INSTALL_DOWNLOAD_BASE_URL="+srv.URL,
	)
}

// TestInstallSubcommand_Integration verifies that `gopher install`
// runs successfully with default (no target) and with a target argument,
// against a local fake release server.
func TestInstallSubcommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bin := buildBinary(t)
	srv := newFakeReleaseServer(t, "1.0.0", "fake gopher binary")

	// No target: resolves "latest" against the fake server.
	cmd := exec.Command(bin, "install")
	cmd.Dir = t.TempDir()
	cmd.Env = installEnv(t, srv)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gopher install failed: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "Install complete") {
		t.Errorf("expected 'Install complete' in output, got:\n%s", got)
	}
}

// TestInstallSubcommand_WithTarget verifies install with a specific version target.
func TestInstallSubcommand_WithTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bin := buildBinary(t)
	srv := newFakeReleaseServer(t, "1.2.3", "fake gopher binary")

	cmd := exec.Command(bin, "install", "1.2.3")
	cmd.Dir = t.TempDir()
	cmd.Env = installEnv(t, srv)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gopher install 1.2.3 failed: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "1.2.3") {
		t.Errorf("expected '1.2.3' in output, got:\n%s", got)
	}
}

// TestInstallSubcommand_Force verifies install with --force flag.
func TestInstallSubcommand_Force(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bin := buildBinary(t)
	srv := newFakeReleaseServer(t, "1.0.0", "fake gopher binary")

	cmd := exec.Command(bin, "install", "--force")
	cmd.Dir = t.TempDir()
	cmd.Env = installEnv(t, srv)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gopher install --force failed: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "Install complete") {
		t.Errorf("expected 'Install complete' in output, got:\n%s", got)
	}
}

// TestInstallSubcommand_InvalidChannel verifies an unrecognized channel/target
// is rejected rather than silently accepted.
func TestInstallSubcommand_InvalidChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bin := buildBinary(t)
	srv := newFakeReleaseServer(t, "1.0.0", "fake gopher binary")

	cmd := exec.Command(bin, "install", "beta")
	cmd.Dir = t.TempDir()
	cmd.Env = installEnv(t, srv)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected gopher install beta to fail, got:\n%s", out)
	}

	got := string(out)
	if !strings.Contains(got, "invalid channel") {
		t.Errorf("expected 'invalid channel' in output, got:\n%s", got)
	}
}
