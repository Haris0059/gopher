package theme

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// no_literals_test.go — guards against color drift: every hex or ANSI-256
// color literal in the tree must live in pkg/ui/theme, not be re-invented
// ad hoc in a component. This is what keeps the four-competing-color-sources
// problem (see the gopher-color-refactor plan) from recurring.

// hexLiteralRe matches a Go string literal that is exactly a hex color, e.g.
// "#6ad4e1" or "#fff". Requires the literal's own quotes so it doesn't flag
// prose containing a "#123"-shaped substring (e.g. "owner/repo#123").
var hexLiteralRe = regexp.MustCompile(`"#[0-9a-fA-F]{3,8}"`)

// ansiLiteralRe matches lipgloss.Color("N") with a bare ANSI-256 numeric
// code (0-255), the pattern used to bypass the theme entirely.
var ansiLiteralRe = regexp.MustCompile(`Color\("[0-9]{1,3}"\)`)

// literalExcludedDirs are directories allowed to contain raw color literals:
// the theme package itself (the source of truth), *_test.go fixtures
// anywhere (checked separately below), and the vendored TS reference tree.
var literalExcludedDirs = []string{
	"claude-code-main",
}

// bannedColors are hex values retired from the palette entirely — not just
// moved into pkg/ui/theme, but forbidden from appearing anywhere in the repo,
// including inside pkg/ui/theme and test fixtures. No exceptions.
//
//   - #1e4976 (old Blue500): the leftover "dark blue" selected-row background
//     in the "/" and "@" autocompletes, unrelated to either brand color.
//     Replaced by the indigo/purple Selection roles (Indigo800/700/100).
var bannedColors = []string{
	"#1e4976",
}

func TestNoColorLiteralsOutsideTheme(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var violations []string

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			for _, excluded := range literalExcludedDirs {
				if name == excluded {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// The theme package is the source of truth for color literals.
		if filepath.Dir(path) == filepath.Join(repoRoot, "pkg", "ui", "theme") {
			return nil
		}
		// Test fixtures are allowed literal colors to assert against
		// (e.g. verifying a raw-color passthrough resolves unchanged).
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)

		rel, _ := filepath.Rel(repoRoot, path)
		for _, m := range hexLiteralRe.FindAllString(content, -1) {
			violations = append(violations, rel+": hex literal "+m+" — add a role to pkg/ui/theme instead")
		}
		for _, m := range ansiLiteralRe.FindAllString(content, -1) {
			violations = append(violations, rel+": ANSI literal "+m+" — reference a theme.ColorScheme role instead")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("found %d color literal(s) outside pkg/ui/theme:\n%s", len(violations), strings.Join(violations, "\n"))
	}
}

// TestBannedColorsNeverAppear checks bannedColors don't reappear anywhere in
// the repo — including pkg/ui/theme and *_test.go, which the general
// no-literals-outside-theme check above exempts. A banned color is retired
// for good, not just relocated.
func TestBannedColorsNeverAppear(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var violations []string

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "claude-code-main" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// This file documents the banned values by name; exclude it from
		// its own scan.
		if filepath.Base(path) == "no_literals_test.go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := strings.ToLower(string(data))

		rel, _ := filepath.Rel(repoRoot, path)
		for _, banned := range bannedColors {
			if strings.Contains(content, strings.ToLower(banned)) {
				violations = append(violations, rel+": banned color "+banned+" must not appear anywhere")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("found %d banned color reference(s):\n%s", len(violations), strings.Join(violations, "\n"))
	}
}

// findRepoRoot walks up from the current package directory to find the
// module root (identified by go.mod).
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (go.mod)")
		}
		dir = parent
	}
}
