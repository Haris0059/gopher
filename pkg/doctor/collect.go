// Collectors that populate the sections of CollectOptions that pkg/ui/app.go
// (and now cmd/gopher/handlers/doctor.go) previously left nil.
// Source: Doctor.tsx useEffect body — gathers agentInfo, contextWarnings,
// versionLockInfo, settings/keybinding/MCP warnings on mount.
package doctor

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Haris0059/gopher/pkg/config"
	"github.com/Haris0059/gopher/pkg/installer"
	"github.com/Haris0059/gopher/pkg/keybindings"
	"github.com/Haris0059/gopher/pkg/mcp"
	"github.com/Haris0059/gopher/pkg/skills"
	uidoctor "github.com/Haris0059/gopher/pkg/ui/doctor"
)

// CollectAgentInfo scans the user and project agent directories for cwd and
// returns their state for the Agents section.
// Source: Doctor.tsx — agentInfo effect (userAgentsDir/projectAgentsDir + pathExists)
func CollectAgentInfo(cwd string) *uidoctor.AgentInfo {
	home, _ := os.UserHomeDir()
	userDir := filepath.Join(home, ".claude", "agents")
	projectDir := filepath.Join(cwd, ".claude", "agents")

	result := skills.LoadAllAgents(cwd)

	info := &uidoctor.AgentInfo{
		UserAgentsDir:    userDir,
		ProjectAgentsDir: projectDir,
		UserDirExists:    dirExists(userDir),
		ProjectDirExists: dirExists(projectDir),
	}
	for _, a := range result.ActiveAgents {
		info.ActiveAgents = append(info.ActiveAgents, uidoctor.AgentEntry{
			AgentType: a.AgentType,
			Source:    string(a.Source),
		})
	}
	for _, f := range result.FailedFiles {
		info.FailedFiles = append(info.FailedFiles, uidoctor.FailedAgentFile{
			Path:  f.Path,
			Error: f.Error,
		})
	}
	return info
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// CollectSettingsErrors validates the global and project settings.json files
// for cwd, returning a SettingsError per schema violation found.
// Source: Doctor.tsx — useSettingsErrors, filtered to non-MCP errors
func CollectSettingsErrors(cwd string) []uidoctor.SettingsError {
	var errs []uidoctor.SettingsError

	home, _ := os.UserHomeDir()
	paths := []string{filepath.Join(home, ".claude", "settings.json")}
	if cwd != "" {
		paths = append(paths, filepath.Join(cwd, ".claude", "settings.json"))
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue // missing settings file is not an error
		}
		for _, e := range config.ValidateSettingsJSON(data) {
			errs = append(errs, uidoctor.SettingsError{
				Path:    e.Path,
				Message: e.Message,
			})
		}
	}
	return errs
}

// CollectKeybindingWarnings loads user keybindings and returns any
// validation warnings raised while parsing/merging them.
// Source: Doctor.tsx — KeybindingWarnings component
func CollectKeybindingWarnings() []uidoctor.KeybindingWarning {
	loader := keybindings.NewLoader()
	var out []uidoctor.KeybindingWarning
	for _, w := range loader.Warnings() {
		out = append(out, uidoctor.KeybindingWarning{
			Key:     w.Key,
			Message: w.Message,
		})
	}
	return out
}

// CollectMCPWarnings checks the user and project MCP config files for cwd,
// returning a warning for each file that fails to parse or each server name
// that fails validation. It does not attempt to connect to any server.
// Source: Doctor.tsx — McpParsingWarnings component (parse-time only)
func CollectMCPWarnings(cwd string) []uidoctor.MCPParsingWarning {
	var warnings []uidoctor.MCPParsingWarning

	home, _ := os.UserHomeDir()
	paths := []string{filepath.Join(home, ".claude", "mcp.json")}
	if cwd != "" {
		paths = append(paths, filepath.Join(cwd, ".mcp.json"))
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue // missing MCP config file is not an error
		}
		var cfg mcp.MCPConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			warnings = append(warnings, uidoctor.MCPParsingWarning{
				Server:  filepath.Base(path),
				Message: "invalid JSON: " + err.Error(),
			})
			continue
		}
		for name := range cfg.Servers {
			if err := mcp.ValidateServerName(name); err != nil {
				warnings = append(warnings, uidoctor.MCPParsingWarning{
					Server:  name,
					Message: err.Error(),
				})
			}
		}
	}
	return warnings
}

// CollectVersionLocks reports the state of pkg/installer's PID-based version
// locks: which versions are currently held, and by whom.
// Source: Doctor.tsx — versionLockInfo effect (getAllLockInfo/cleanupStaleLocks)
func CollectVersionLocks(dirs installer.Dirs) *uidoctor.VersionLockInfo {
	cleaned := installer.CleanupStaleLocks(dirs)
	locks := installer.ListLocks(dirs)

	info := &uidoctor.VersionLockInfo{
		Enabled:           true,
		LocksDir:          dirs.Locks,
		StaleLocksCleaned: cleaned,
	}
	for _, l := range locks {
		info.Locks = append(info.Locks, uidoctor.LockInfo{
			Version:          l.Version,
			PID:              l.PID,
			IsProcessRunning: l.IsRunning,
		})
	}
	return info
}

// DefaultFetchDistTags resolves the "latest" and "stable" channels against
// GitHub releases via pkg/installer.ResolveVersion.
// Source: Doctor.tsx — distTagsPromise (getGcsDistTags/getNpmDistTags)
func DefaultFetchDistTags() (*uidoctor.DistTags, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	latest, err := installer.ResolveVersion(ctx, http.DefaultClient, "", "latest")
	if err != nil {
		return nil, err
	}
	stable, err := installer.ResolveVersion(ctx, http.DefaultClient, "", "stable")
	if err != nil {
		stable = "" // stable is best-effort; latest alone is still useful
	}
	return &uidoctor.DistTags{Stable: stable, Latest: latest}, nil
}
