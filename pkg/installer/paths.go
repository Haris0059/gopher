// Package installer implements Gopher's self-install/update flow: resolving
// the latest release, downloading and verifying the right asset, installing
// it into a versioned directory, and symlinking it into the user's PATH.
// Source: src/utils/nativeInstaller/ (installer.ts, download.ts, pidLock.ts)
package installer

import (
	"os"
	"path/filepath"
	"runtime"
)

// Dirs holds the directory layout used by the installer.
// Source: installer.ts:115-132 getBaseDirectories()
type Dirs struct {
	// Versions holds one file per installed version (XDG_DATA_HOME/gopher/versions).
	Versions string
	// Staging holds in-progress downloads (XDG_CACHE_HOME/gopher/staging).
	Staging string
	// Locks holds per-version lockfiles (XDG_STATE_HOME/gopher/locks).
	Locks string
	// Executable is the path the user runs, e.g. ~/.local/bin/gopher.
	Executable string
}

// baseDirsFor computes Dirs from a GOOS string, an env lookup function, and
// the resolved home directory. It performs no I/O and no runtime/os reads,
// so it is fully table-testable.
func baseDirsFor(goos string, env func(string) string, home string) Dirs {
	dataHome := xdgOr(env, "XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	cacheHome := xdgOr(env, "XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	stateHome := xdgOr(env, "XDG_STATE_HOME", filepath.Join(home, ".local", "state"))

	return Dirs{
		Versions:   filepath.Join(dataHome, "gopher", "versions"),
		Staging:    filepath.Join(cacheHome, "gopher", "staging"),
		Locks:      filepath.Join(stateHome, "gopher", "locks"),
		Executable: filepath.Join(installDirFor(goos, env, home), binaryNameFor(goos)),
	}
}

// xdgOr returns env(key) if set and non-empty, else def.
func xdgOr(env func(string) string, key, def string) string {
	if v := env(key); v != "" {
		return v
	}
	return def
}

// DefaultDirs returns the directory layout for the current platform and
// environment.
func DefaultDirs() Dirs {
	home, _ := os.UserHomeDir()
	return baseDirsFor(runtime.GOOS, os.Getenv, home)
}

// installDirFor computes InstallDir for an arbitrary GOOS/env/home, the pure
// core behind InstallDir().
func installDirFor(goos string, env func(string) string, home string) string {
	switch goos {
	case "windows":
		appData := env("LOCALAPPDATA")
		if appData == "" {
			appData = home
		}
		return filepath.Join(appData, "Gopher", "bin")
	default:
		// darwin, linux, and everything else: XDG user bin dir.
		// Source: xdg.ts getUserBinDir() — always $HOME/.local/bin, no XDG override.
		return filepath.Join(home, ".local", "bin")
	}
}

// binaryNameFor computes BinaryName for an arbitrary GOOS.
func binaryNameFor(goos string) string {
	if goos == "windows" {
		return "gopher.exe"
	}
	return "gopher"
}

// platformFor maps a GOOS/GOARCH pair to the release asset platform string
// used in download URLs, e.g. "linux-amd64", "darwin-arm64".
func platformFor(goos, goarch string) string {
	return goos + "-" + goarch
}

// InstallDir returns the default installation directory for the platform.
func InstallDir() string {
	home, _ := os.UserHomeDir()
	return installDirFor(runtime.GOOS, os.Getenv, home)
}

// BinaryName returns the binary name for the current platform.
func BinaryName() string {
	return binaryNameFor(runtime.GOOS)
}

// IsInstalled checks if the CLI is installed in the default location.
func IsInstalled() bool {
	path := filepath.Join(InstallDir(), BinaryName())
	_, err := os.Stat(path)
	return err == nil
}
