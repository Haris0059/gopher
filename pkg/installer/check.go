package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SetupMessage describes an installation-health notice surfaced to the user
// after install. It never signals failure by itself.
// Source: installer.ts SetupMessage type, :800 checkInstall()
type SetupMessage struct {
	Type               string // "error" | "path" | "alias" | "info"
	Text               string
	UserActionRequired bool
}

// CheckInstall validates the installed executable is reachable, returning
// advisory messages. It never returns an error: a missing PATH entry is
// reported as a "path" message while the install itself still succeeded.
// Source: installer.ts:800-940 checkInstall()
func CheckInstall() []SetupMessage {
	dir := InstallDir()

	var messages []SetupMessage
	if !dirInPATH(dir, os.Getenv("PATH")) {
		messages = append(messages, SetupMessage{
			Type:               "path",
			Text:               dir + " is not in your PATH.",
			UserActionRequired: true,
		})
	}
	return messages
}

// dirInPATH reports whether dir appears as an entry of the PATH string,
// using the platform's list separator.
func dirInPATH(dir, pathEnv string) bool {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	for _, entry := range strings.Split(pathEnv, sep) {
		if entry == "" {
			continue
		}
		if samePath(entry, dir) {
			return true
		}
	}
	return false
}

// samePath compares two paths after cleaning, without resolving symlinks.
func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
