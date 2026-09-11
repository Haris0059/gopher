package handlers_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Haris0059/gopher/cmd/gopher/handlers"
	"github.com/Haris0059/gopher/pkg/skills"
)

func TestAgentsHandler_BuiltInsAlwaysListed(t *testing.T) {
	t.Parallel()
	// Use a temp dir that has no .claude/agents anywhere (project side; the
	// real ~/.claude/agents may still contribute on the developer's machine,
	// so this test only asserts on the built-ins, not an exact count).
	tmp := t.TempDir()
	var buf bytes.Buffer
	handlers.AgentsHandler(&buf, tmp)
	got := buf.String()

	if !strings.Contains(got, "Built-in agents:") {
		t.Errorf("expected 'Built-in agents:' group, got:\n%s", got)
	}
	if !strings.Contains(got, "Explore") || !strings.Contains(got, "general-purpose") {
		t.Errorf("expected built-in agent names present, got:\n%s", got)
	}
}

func TestAgentsHandlerWithDirs_NoAgents(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handlers.AgentsHandlerWithDirs(&buf, map[skills.AgentSource]string{})
	got := buf.String()

	// Built-ins are always present via AgentsHandlerWithDirs too, so this
	// exercises the "no custom agents" path, not a truly empty listing.
	if !strings.Contains(got, "Built-in agents:") {
		t.Errorf("expected 'Built-in agents:' group, got:\n%s", got)
	}
	if strings.Contains(got, "Project agents:") || strings.Contains(got, "User agents:") {
		t.Errorf("expected no custom agent groups, got:\n%s", got)
	}
}

func TestAgentsHandler_MarkdownAgents(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// Create project-level agents dir with two agents defined via frontmatter.
	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeAgentMd(t, agentsDir, "code-review.md", "code-review", "Reviews code for issues")
	writeAgentMd(t, agentsDir, "deploy.md", "deploy", "Deploys the project")

	var buf bytes.Buffer
	handlers.AgentsHandler(&buf, tmp)
	got := buf.String()

	// Verify group label.
	if !strings.Contains(got, "Project agents:") {
		t.Errorf("expected 'Project agents:' group header, got:\n%s", got)
	}

	// Verify 2-space indent on agent lines.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "code-review") || strings.Contains(line, "deploy") {
			if !strings.HasPrefix(line, "  ") {
				t.Errorf("expected 2-space indent on agent line, got: %q", line)
			}
		}
	}

	// Verify alphabetical order (code-review before deploy).
	crIdx := strings.Index(got, "code-review")
	dIdx := strings.Index(got, "deploy")
	if crIdx < 0 || dIdx < 0 || crIdx >= dIdx {
		t.Errorf("expected code-review before deploy in output:\n%s", got)
	}
}

func TestAgentsHandler_JSONAgents(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// prompt is required by the schema (loadAgentsDir.ts AgentJsonSchema);
	// entries without one are silently skipped.
	data := `{"tester":{"description":"Runs tests","prompt":"Run the test suite.","model":"claude-sonnet-4-6","memory":"user"}}`
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	handlers.AgentsHandler(&buf, tmp)
	got := buf.String()

	// Verify middle-dot format: "tester · claude-sonnet-4-6 · user memory"
	if !strings.Contains(got, "tester · claude-sonnet-4-6 · user memory") {
		t.Errorf("expected formatted agent line with middle dots, got:\n%s", got)
	}
}

func TestAgentsHandler_JSONAgents_MissingPromptSkipped(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// No "prompt" field: this entry must be rejected.
	data := `{"tester":{"description":"Runs tests","model":"claude-sonnet-4-6"}}`
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	handlers.AgentsHandler(&buf, tmp)
	got := buf.String()

	if strings.Contains(got, "tester") {
		t.Errorf("expected agent missing 'prompt' to be skipped, got:\n%s", got)
	}
}

func TestAgentsHandler_ShadowedAgent(t *testing.T) {
	t.Parallel()

	userDir := t.TempDir()
	projectDir := t.TempDir()

	// Create the same agent in both user and project dirs.
	writeAgentMd(t, userDir, "reviewer.md", "reviewer", "Reviews code")
	writeAgentMd(t, projectDir, "reviewer.md", "reviewer", "Reviews code")

	dirs := map[skills.AgentSource]string{
		skills.AgentSourceUser:    userDir,
		skills.AgentSourceProject: projectDir,
	}

	var buf bytes.Buffer
	handlers.AgentsHandlerWithDirs(&buf, dirs)
	got := buf.String()

	// Project overrides user, so user should be shadowed.
	if !strings.Contains(got, "(shadowed by project) reviewer") {
		t.Errorf("expected user agent to be shadowed by project, got:\n%s", got)
	}
}

func TestAgentsHandler_OutputEndsClean(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeAgentMd(t, agentsDir, "alpha.md", "alpha", "An agent")

	var buf bytes.Buffer
	handlers.AgentsHandler(&buf, tmp)
	got := buf.String()

	// Output should end with a single newline (from Fprintln), not multiple trailing newlines.
	trimmed := strings.TrimRight(got, "\n")
	trailing := got[len(trimmed):]
	if trailing != "\n" {
		t.Errorf("expected output to end with exactly one newline, got %d trailing newlines", len(trailing))
	}
}

// writeAgentMd writes a minimal valid agent markdown file with frontmatter
// (name + description are required by skills.ParseAgentFromMarkdown).
func writeAgentMd(t *testing.T, dir, filename, name, description string) {
	t.Helper()
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
