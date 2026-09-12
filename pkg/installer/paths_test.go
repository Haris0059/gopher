package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestBaseDirsFor_XDGUnset(t *testing.T) {
	home := "/home/u"
	got := baseDirsFor("linux", envMap(nil), home)

	want := Dirs{
		Versions:   filepath.Join(home, ".local", "share", "gopher", "versions"),
		Staging:    filepath.Join(home, ".cache", "gopher", "staging"),
		Locks:      filepath.Join(home, ".local", "state", "gopher", "locks"),
		Executable: filepath.Join(home, ".local", "bin", "gopher"),
	}
	if got != want {
		t.Errorf("baseDirsFor(linux, no XDG) = %+v, want %+v", got, want)
	}
}

func TestBaseDirsFor_XDGSet(t *testing.T) {
	home := "/home/u"
	env := envMap(map[string]string{
		"XDG_DATA_HOME":  "/data",
		"XDG_CACHE_HOME": "/cache",
		"XDG_STATE_HOME": "/state",
	})
	got := baseDirsFor("linux", env, home)

	want := Dirs{
		Versions:   filepath.Join("/data", "gopher", "versions"),
		Staging:    filepath.Join("/cache", "gopher", "staging"),
		Locks:      filepath.Join("/state", "gopher", "locks"),
		Executable: filepath.Join(home, ".local", "bin", "gopher"),
	}
	if got != want {
		t.Errorf("baseDirsFor(linux, XDG set) = %+v, want %+v", got, want)
	}
}

func TestBaseDirsFor_Darwin(t *testing.T) {
	home := "/Users/u"
	got := baseDirsFor("darwin", envMap(nil), home)

	if want := filepath.Join(home, ".local", "bin", "gopher"); got.Executable != want {
		t.Errorf("darwin Executable = %s, want %s", got.Executable, want)
	}
}

func TestBaseDirsFor_Windows(t *testing.T) {
	home := `C:\Users\u`

	t.Run("LOCALAPPDATA set", func(t *testing.T) {
		env := envMap(map[string]string{"LOCALAPPDATA": `C:\Users\u\AppData\Local`})
		got := baseDirsFor("windows", env, home)
		want := filepath.Join(`C:\Users\u\AppData\Local`, "Gopher", "bin", "gopher.exe")
		if got.Executable != want {
			t.Errorf("windows Executable = %s, want %s", got.Executable, want)
		}
	})

	t.Run("LOCALAPPDATA unset falls back to home", func(t *testing.T) {
		got := baseDirsFor("windows", envMap(nil), home)
		want := filepath.Join(home, "Gopher", "bin", "gopher.exe")
		if got.Executable != want {
			t.Errorf("windows Executable = %s, want %s", got.Executable, want)
		}
	})
}

func TestBaseDirsFor_UnknownGOOS(t *testing.T) {
	home := "/home/u"
	got := baseDirsFor("plan9", envMap(nil), home)
	want := filepath.Join(home, ".local", "bin", "gopher")
	if got.Executable != want {
		t.Errorf("unknown GOOS Executable = %s, want %s", got.Executable, want)
	}
}

func TestBinaryNameFor(t *testing.T) {
	if got := binaryNameFor("windows"); got != "gopher.exe" {
		t.Errorf("binaryNameFor(windows) = %s, want gopher.exe", got)
	}
	for _, goos := range []string{"linux", "darwin", "freebsd"} {
		if got := binaryNameFor(goos); got != "gopher" {
			t.Errorf("binaryNameFor(%s) = %s, want gopher", goos, got)
		}
	}
}

func TestPlatformFor(t *testing.T) {
	cases := []struct{ goos, goarch, want string }{
		{"linux", "amd64", "linux-amd64"},
		{"linux", "arm64", "linux-arm64"},
		{"darwin", "arm64", "darwin-arm64"},
		{"darwin", "amd64", "darwin-amd64"},
		{"windows", "amd64", "windows-amd64"},
	}
	for _, c := range cases {
		if got := platformFor(c.goos, c.goarch); got != c.want {
			t.Errorf("platformFor(%s, %s) = %s, want %s", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestIsInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if IsInstalled() {
		t.Error("IsInstalled() = true before binary exists")
	}

	dir := InstallDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, BinaryName()), []byte("bin"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !IsInstalled() {
		t.Error("IsInstalled() = false after binary created")
	}
}
