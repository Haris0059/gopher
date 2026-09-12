package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Options configures Install.
type Options struct {
	// Version is a concrete version, "latest"/"stable", or "" (== latest).
	Version string
	// Force reinstalls even if the version is already present.
	Force bool

	Dirs            Dirs // zero value -> DefaultDirs()
	HTTPClient      *http.Client
	APIBaseURL      string // "" -> DefaultAPIBaseURL
	DownloadBaseURL string // "" -> DefaultDownloadBaseURL
	GOOS, GOARCH    string // "" -> runtime values

	Log func(string)
}

// Result describes the outcome of a successful Install.
type Result struct {
	Version        string
	InstalledPath  string // versions/<version>
	ExecutablePath string // the symlink/copy the user runs
	// Downloaded is false when the version was already installed and Force
	// was not set.
	Downloaded bool
}

var installMu sync.Mutex

// Install resolves, downloads (if needed), verifies, and installs the
// requested version, then points the user-facing executable at it and runs
// GC. Concurrent non-forced calls for the same process are serialized rather
// than raced. Source: installer.ts:441-488 performVersionUpdate(),
// :955-974 installLatest() singleflight wrapper.
func Install(ctx context.Context, opts Options) (Result, error) {
	installMu.Lock()
	defer installMu.Unlock()

	dirs := opts.Dirs
	if dirs == (Dirs{}) {
		dirs = DefaultDirs()
	}
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	goos := opts.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := opts.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	logf := opts.Log
	if logf == nil {
		logf = func(string) {}
	}

	version, err := ResolveVersion(ctx, client, opts.APIBaseURL, opts.Version)
	if err != nil {
		return Result{}, fmt.Errorf("failed to resolve version: %w", err)
	}

	for _, dir := range []string{dirs.Versions, dirs.Staging, dirs.Locks, filepath.Dir(dirs.Executable)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Result{}, fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	release, err := acquireLock(dirs, version)
	if err != nil {
		return Result{}, err
	}
	defer release()

	installPath := filepath.Join(dirs.Versions, version)

	downloaded := false
	if opts.Force || !isUsableBinary(installPath) {
		logf(fmt.Sprintf("installing version %s", version))

		stagingPath := filepath.Join(dirs.Staging, fmt.Sprintf("%s.%d.%d", version, os.Getpid(), time.Now().UnixNano()))
		if err := downloadVersion(ctx, client, opts.DownloadBaseURL, version, goos, goarch, stagingPath); err != nil {
			return Result{}, err
		}
		defer os.Remove(stagingPath)

		if err := atomicMoveToInstallPath(stagingPath, installPath); err != nil {
			return Result{}, err
		}
		downloaded = true
	} else {
		logf(fmt.Sprintf("version %s already installed, updating symlink", version))
	}

	if err := removeEmptyDir(dirs.Executable); err != nil {
		return Result{}, err
	}
	if err := updateSymlink(dirs.Executable, installPath); err != nil {
		return Result{}, err
	}

	if !isUsableBinary(dirs.Executable) {
		_, statErr := os.Stat(installPath)
		return Result{}, fmt.Errorf(
			"failed to create executable at %s: source exists=%t; check write permissions",
			dirs.Executable, statErr == nil,
		)
	}

	if err := GC(dirs, VersionRetentionCount); err != nil {
		logf(fmt.Sprintf("gc failed (non-fatal): %v", err))
	}

	return Result{
		Version:        version,
		InstalledPath:  installPath,
		ExecutablePath: dirs.Executable,
		Downloaded:     downloaded,
	}, nil
}

// isUsableBinary reports whether path is a non-empty, executable regular
// file. A zero-byte file is the placeholder written for an in-progress or
// never-completed install and does not count.
// Source: installer.ts:134-150 isPossibleClaudeBinary()
func isUsableBinary(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

// atomicMoveToInstallPath copies src to dest via a temp-file-then-rename, so
// a rename failure never leaves a partial file at dest. A plain os.Rename
// isn't safe here because src (staging) and dest (versions) can live under
// different XDG base dirs, i.e. different filesystems (EXDEV).
// Source: installer.ts:300-315 atomicMoveToInstallPath()
func atomicMoveToInstallPath(src, dest string) error {
	tmp := fmt.Sprintf("%s.tmp.%d.%d", dest, os.Getpid(), time.Now().UnixNano())

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open staged file: %w", err)
	}
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		in.Close()
		return fmt.Errorf("failed to create %s: %w", tmp, err)
	}
	_, copyErr := io.Copy(out, in)
	closeInErr := in.Close()
	closeOutErr := out.Close()

	if copyErr != nil || closeInErr != nil || closeOutErr != nil {
		os.Remove(tmp)
		if copyErr != nil {
			return copyErr
		}
		if closeInErr != nil {
			return closeInErr
		}
		return closeOutErr
	}

	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to install to %s: %w", dest, err)
	}
	return nil
}

// removeEmptyDir removes path if it exists and is an empty directory,
// clearing the way for a symlink/binary to be placed there. It is a no-op
// if path doesn't exist or isn't a directory.
// Source: installer.ts:623-637 removeDirectoryIfEmpty()
func removeEmptyDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return nil // doesn't exist
	}
	if !info.IsDir() {
		return nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("executable path %s is a non-empty directory: %w", path, err)
	}
	return nil
}
