package installer

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

// updateSymlink points executablePath at target. On Unix it creates (or
// replaces) a symlink atomically via temp-then-rename; on Windows, where
// symlinks require elevated privileges, it copies the file instead. It is a
// no-op if executablePath is already a symlink resolving to target.
// Source: installer.ts:639-796 updateSymlink()
func updateSymlink(executablePath, target string) error {
	if runtime.GOOS == "windows" {
		return copyFile(target, executablePath)
	}

	if current, err := os.Readlink(executablePath); err == nil && current == target {
		return nil // already correct
	}

	tmp := fmt.Sprintf("%s.tmp.%d.%d", executablePath, os.Getpid(), time.Now().UnixNano())
	os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	if err := os.Rename(tmp, executablePath); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to activate symlink at %s: %w", executablePath, err)
	}
	return nil
}

// copyFile copies src to dest, replacing dest if it exists, preserving an
// executable mode.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := fmt.Sprintf("%s.tmp.%d.%d", dest, os.Getpid(), time.Now().UnixNano())
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
