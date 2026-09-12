package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Haris0059/gopher/pkg/installer"
)

func TestCollectAgentInfo_ReportsMissingDirs(t *testing.T) {
	cwd := t.TempDir()
	info := CollectAgentInfo(cwd)
	if info == nil {
		t.Fatal("expected non-nil AgentInfo")
	}
	if info.ProjectDirExists {
		t.Error("expected project agents dir to not exist in a fresh temp dir")
	}
	if info.ProjectAgentsDir != filepath.Join(cwd, ".claude", "agents") {
		t.Errorf("unexpected project agents dir: %q", info.ProjectAgentsDir)
	}
}

func TestCollectSettingsErrors_MalformedProjectSettings(t *testing.T) {
	cwd := t.TempDir()
	dir := filepath.Join(cwd, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// max_turns must be an integer >= 1 per the schema.
	bad := `{"max_turns": "not-a-number"}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	errs := CollectSettingsErrors(cwd)
	if len(errs) == 0 {
		t.Error("expected at least one settings error for malformed settings.json")
	}
}

func TestCollectSettingsErrors_NoSettingsFile(t *testing.T) {
	cwd := t.TempDir()
	errs := CollectSettingsErrors(cwd)
	if len(errs) != 0 {
		t.Errorf("expected no errors when no settings file exists, got %v", errs)
	}
}

func TestCollectMCPWarnings_InvalidJSON(t *testing.T) {
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, ".mcp.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	warnings := CollectMCPWarnings(cwd)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for invalid JSON, got %d", len(warnings))
	}
	if warnings[0].Server != ".mcp.json" {
		t.Errorf("expected warning to name the file, got %q", warnings[0].Server)
	}
}

func TestCollectMCPWarnings_InvalidServerName(t *testing.T) {
	cwd := t.TempDir()
	cfg := map[string]any{
		"mcpServers": map[string]any{
			"bad name!": map[string]any{"command": "echo"},
		},
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(cwd, ".mcp.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	warnings := CollectMCPWarnings(cwd)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for invalid server name, got %d", len(warnings))
	}
	if warnings[0].Server != "bad name!" {
		t.Errorf("expected warning for %q, got %q", "bad name!", warnings[0].Server)
	}
}

func TestCollectMCPWarnings_NoConfigFiles(t *testing.T) {
	cwd := t.TempDir()
	if warnings := CollectMCPWarnings(cwd); len(warnings) != 0 {
		t.Errorf("expected no warnings when no MCP config exists, got %v", warnings)
	}
}

func TestCollectVersionLocks_LiveAndStale(t *testing.T) {
	dirs := installer.Dirs{Locks: t.TempDir()}

	writeLock := func(name string, pid int) {
		data, _ := json.Marshal(map[string]any{
			"pid":     pid,
			"version": name,
		})
		if err := os.WriteFile(filepath.Join(dirs.Locks, name+".lock"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A lock held by a definitely-dead PID (0 is never a real process on
	// any platform we run on) and one held by this test process itself.
	writeLock("stale", 999999999)
	writeLock("live", os.Getpid())

	info := CollectVersionLocks(dirs)
	if info == nil || !info.Enabled {
		t.Fatal("expected an enabled VersionLockInfo")
	}
	if info.StaleLocksCleaned != 1 {
		t.Errorf("expected 1 stale lock cleaned, got %d", info.StaleLocksCleaned)
	}
	if len(info.Locks) != 1 || info.Locks[0].Version != "live" {
		t.Fatalf("expected only the live lock to remain, got %+v", info.Locks)
	}
	if !info.Locks[0].IsProcessRunning {
		t.Error("expected the live lock's process to be reported as running")
	}
}

func TestCollectVersionLocks_EmptyDir(t *testing.T) {
	dirs := installer.Dirs{Locks: t.TempDir()}
	info := CollectVersionLocks(dirs)
	if info == nil || !info.Enabled {
		t.Fatal("expected an enabled VersionLockInfo")
	}
	if len(info.Locks) != 0 {
		t.Errorf("expected no locks, got %+v", info.Locks)
	}
}

func TestCollectKeybindingWarnings_DoesNotPanic(t *testing.T) {
	// Exercises the real loader; just verify it returns without panicking.
	_ = CollectKeybindingWarnings()
}
