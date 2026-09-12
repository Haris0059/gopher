// Source: src/cli/handlers/util.tsx — installHandler
package handlers

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Haris0059/gopher/pkg/installer"
)

// InstallOpts configures the install handler.
type InstallOpts struct {
	Target string    // positional: install target (e.g. a version, "latest", "stable")
	Force  bool      // --force flag
	Output io.Writer // defaults to os.Stdout

	// APIBaseURL and DownloadBaseURL override where the default installer
	// resolves and downloads releases from. "" uses the real GitHub hosts.
	// Only meaningful when Installer is nil.
	APIBaseURL      string
	DownloadBaseURL string
	// Dirs overrides the installer's directory layout. Zero value uses
	// installer.DefaultDirs(). Only meaningful when Installer is nil.
	Dirs installer.Dirs

	// Installer is the pluggable install engine. Nil uses the default,
	// which calls pkg/installer.Install.
	// Returns a result string; if it contains "failed" the handler exits 1.
	// Source: src/commands/install.js — install.call(result => ...)
	Installer func(target string, force bool) (string, error)
}

// Install handles `claude install [target] [--force]`.
// Source: src/cli/handlers/util.tsx — installHandler
func Install(opts InstallOpts) int {
	w := output(opts.Output)

	install := opts.Installer
	if install == nil {
		install = defaultInstaller(w, opts)
	}

	result, err := install(opts.Target, opts.Force)
	if err != nil {
		fmt.Fprintf(w, "Install failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(w, result)

	// Source: process.exit(result.includes('failed') ? 1 : 0)
	if strings.Contains(result, "failed") {
		return 1
	}
	return 0
}

// defaultInstaller builds the real install engine backed by pkg/installer,
// writing progress to w as it goes.
func defaultInstaller(w io.Writer, opts InstallOpts) func(target string, force bool) (string, error) {
	return func(target string, force bool) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		result, err := installer.Install(ctx, installer.Options{
			Version:         target,
			Force:           force,
			Dirs:            opts.Dirs,
			APIBaseURL:      opts.APIBaseURL,
			DownloadBaseURL: opts.DownloadBaseURL,
			Log: func(msg string) {
				fmt.Fprintln(w, msg)
			},
		})
		if err != nil {
			return "", err
		}

		fmt.Fprintf(w, "Installed version %s to %s\n", result.Version, result.InstalledPath)
		fmt.Fprintf(w, "Executable: %s\n", result.ExecutablePath)

		for _, msg := range installer.CheckInstall() {
			fmt.Fprintf(w, "Warning: %s\n", msg.Text)
		}

		return "Install complete", nil
	}
}
