// Package doctor aggregates all diagnostic data for the /doctor health-check screen.
// Source: Doctor.tsx — getDoctorDiagnostic aggregator
package doctor

import (
	"os"
	"runtime"

	"github.com/Haris0059/gopher/pkg/installer"
	uidoctor "github.com/Haris0059/gopher/pkg/ui/doctor"
)

// Version is injected at build time or set by caller.
// Patch segment is the count of Haris0059's commits on this project.
var Version = "0.3.047"

// Warning is a diagnostic warning with a suggested fix.
// Source: Doctor.tsx — DiagnosticInfo.warnings
type Warning struct {
	Issue string
	Fix   string
}

// DiagnosticData holds all collected diagnostic information.
// Source: Doctor.tsx — getDoctorDiagnostic return type
type DiagnosticData struct {
	// Core diagnostic info (T61)
	Version             string
	InstallationType    string
	InstallationPath    string
	InvokedBinary       string
	ConfigInstallMethod string
	PackageManager      string
	Warnings            []Warning
	Ripgrep             RipgrepStatus

	// Dist tags (T62)
	DistTags    *uidoctor.DistTags
	DistTagsErr error

	// Context warnings (T63)
	ContextWarnings *uidoctor.ContextWarnings

	// PID lock info (T64)
	VersionLocks *uidoctor.VersionLockInfo

	// Agent info (T65)
	AgentInfo *uidoctor.AgentInfo

	// Env-var validation (T66)
	EnvValidation []uidoctor.EnvValidationResult

	// Settings/keybinding/MCP warnings (T67)
	SettingsErrors     []uidoctor.SettingsError
	KeybindingWarnings []uidoctor.KeybindingWarning
	MCPWarnings        []uidoctor.MCPParsingWarning

	// Sandbox status (T68)
	Sandbox uidoctor.SandboxStatus

	// Update settings
	AutoUpdates   string
	UpdateChannel string
}

// CollectOptions configures which diagnostic sections to collect.
type CollectOptions struct {
	// Cwd is the project directory used to locate project-scoped agents,
	// settings, and MCP config. Empty uses the process's working directory.
	Cwd string

	// InstallationType, PackageManager and ConfigInstallMethod override the
	// package defaults ("go-binary", "", "direct"). Callers populate these
	// from real detectors (see cmd/gopher/handlers.DefaultDetectInstallType
	// and DefaultDetectPackageManager) — pkg/doctor has no detector of its
	// own to avoid importing the CLI handlers package.
	InstallationType    string
	PackageManager      string
	ConfigInstallMethod string

	// FetchDistTags is an optional callback for fetching dist tags.
	// If nil, dist tags are skipped.
	FetchDistTags uidoctor.FetchDistTagsFunc

	// EnvBounds is the list of env-var bounds to validate.
	// If nil, DefaultEnvBounds() is used.
	EnvBounds []uidoctor.EnvBound

	// InstallerDirs is the directory layout used to look up version locks.
	// If zero, installer.DefaultDirs() is used.
	InstallerDirs installer.Dirs

	// SettingsErrors are pre-collected settings validation errors. If nil,
	// CollectSettingsErrors(Cwd) is used.
	SettingsErrors []uidoctor.SettingsError

	// KeybindingWarnings are pre-collected keybinding parse warnings. If
	// nil, CollectKeybindingWarnings() is used.
	KeybindingWarnings []uidoctor.KeybindingWarning

	// MCPWarnings are pre-collected MCP config parse warnings. If nil,
	// CollectMCPWarnings(Cwd) is used.
	MCPWarnings []uidoctor.MCPParsingWarning

	// ContextWarnings are pre-collected context warnings. There is no
	// default collector for these yet (see TASKS.md).
	ContextWarnings *uidoctor.ContextWarnings

	// VersionLocks are pre-collected PID lock info. If nil,
	// CollectVersionLocks(InstallerDirs) is used.
	VersionLocks *uidoctor.VersionLockInfo

	// AgentInfo is pre-collected agent directory scan results. If nil,
	// CollectAgentInfo(Cwd) is used.
	AgentInfo *uidoctor.AgentInfo
}

// Collect gathers all diagnostic data.
// Source: Doctor.tsx — getDoctorDiagnostic function
func Collect(opts CollectOptions) *DiagnosticData {
	cwd := opts.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	installationType := opts.InstallationType
	if installationType == "" {
		installationType = "go-binary"
	}
	configInstallMethod := opts.ConfigInstallMethod
	if configInstallMethod == "" {
		configInstallMethod = "direct"
	}

	data := &DiagnosticData{
		Version:             Version,
		InstallationType:    installationType,
		InstallationPath:    executablePath(),
		InvokedBinary:       os.Args[0],
		ConfigInstallMethod: configInstallMethod,
		PackageManager:      opts.PackageManager,
		AutoUpdates:         "enabled",
		UpdateChannel:       "latest",
		Ripgrep:             DetectRipgrep(),
	}

	// Local, low-cost checks: PATH reachability and Linux glob footguns.
	for _, m := range installer.CheckInstall() {
		data.Warnings = append(data.Warnings, Warning{Issue: m.Text})
	}
	for _, w := range DetectLinuxGlobPatternWarnings() {
		data.Warnings = append(data.Warnings, Warning{Issue: w.Issue, Fix: w.Fix})
	}

	// T62: Dist tags
	if opts.FetchDistTags != nil {
		data.DistTags, data.DistTagsErr = opts.FetchDistTags()
	}

	// T63: Context warnings — no default collector yet (see TASKS.md).
	data.ContextWarnings = opts.ContextWarnings

	// T64: PID locks
	if opts.VersionLocks != nil {
		data.VersionLocks = opts.VersionLocks
	} else {
		dirs := opts.InstallerDirs
		if dirs == (installer.Dirs{}) {
			dirs = installer.DefaultDirs()
		}
		data.VersionLocks = CollectVersionLocks(dirs)
	}

	// T65: Agent info
	if opts.AgentInfo != nil {
		data.AgentInfo = opts.AgentInfo
	} else {
		data.AgentInfo = CollectAgentInfo(cwd)
	}

	// T66: Env-var validation
	bounds := opts.EnvBounds
	if bounds == nil {
		bounds = uidoctor.DefaultEnvBounds()
	}
	data.EnvValidation = uidoctor.ValidateEnvBounds(bounds)

	// T67: Settings/keybinding/MCP warnings
	if opts.SettingsErrors != nil {
		data.SettingsErrors = opts.SettingsErrors
	} else {
		data.SettingsErrors = CollectSettingsErrors(cwd)
	}
	if opts.KeybindingWarnings != nil {
		data.KeybindingWarnings = opts.KeybindingWarnings
	} else {
		data.KeybindingWarnings = CollectKeybindingWarnings()
	}
	if opts.MCPWarnings != nil {
		data.MCPWarnings = opts.MCPWarnings
	} else {
		data.MCPWarnings = CollectMCPWarnings(cwd)
	}

	// T68: Sandbox
	data.Sandbox = uidoctor.DetectSandboxStatus()

	return data
}

func executablePath() string {
	path, err := os.Executable()
	if err != nil {
		return runtime.GOARCH + "-unknown"
	}
	return path
}
