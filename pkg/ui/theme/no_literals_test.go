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
