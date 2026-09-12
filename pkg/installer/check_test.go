package installer

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDirInPATH(t *testing.T) {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	dir := filepath.Join("home", "u", ".local", "bin")
	pathEnv := "/usr/bin" + sep + dir + sep + "/usr/local/bin"

	if !dirInPATH(dir, pathEnv) {
		t.Error("dirInPATH() = false, want true when dir is a PATH entry")
	}
	if dirInPATH(filepath.Join("home", "u", "nowhere"), pathEnv) {
		t.Error("dirInPATH() = true for a directory not in PATH")
	}
}

func TestCheckInstall_DirInPATH(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	t.Setenv("PATH", InstallDir()+sep+"/usr/bin")

	msgs := CheckInstall()
	for _, m := range msgs {
		if m.Type == "path" {
			t.Errorf("unexpected path warning when InstallDir is in PATH: %+v", m)
		}
	}
}

func TestCheckInstall_DirNotInPATH(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin")

	msgs := CheckInstall()

	found := false
	for _, m := range msgs {
		if m.Type == "path" {
			found = true
			if !m.UserActionRequired {
				t.Error("path warning should have UserActionRequired = true")
			}
		}
	}
	if !found {
		t.Error("expected a path warning when InstallDir is not in PATH")
	}
}
