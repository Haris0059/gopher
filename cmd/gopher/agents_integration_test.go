package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAgentsSubcommand_Integration builds the binary and verifies that
// `gopher agents` runs successfully and produces expected output.
func TestAgentsSubcommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary into a temp dir.
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "gopher")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = filepath.Dir(selfPath(t))
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	// Run the binary with "agents" subcommand in an empty temp dir so there
	// are no agents to discover.
	cwd := t.TempDir()
	cmd := exec.Command(bin, "agents")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gopher agents failed: %v\n%s", err, out)
	}

	// Built-in agents are always listed, so an empty project directory no
	// longer produces "No agents found." — it lists just the built-ins.
	got := string(out)
	if !strings.Contains(got, "Built-in agents:") {
		t.Errorf("expected 'Built-in agents:' in output, got:\n%s", got)
	}
	if strings.Contains(got, "Project agents:") {
		t.Errorf("expected no 'Project agents:' group in an empty project dir, got:\n%s", got)
	}
}

// TestAgentsSubcommand_WithAgents builds the binary and verifies that it
// discovers and lists agents from a project directory.
func TestAgentsSubcommand_WithAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary.
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "gopher")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = filepath.Dir(selfPath(t))
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	// Set up a project dir with an agent.
	cwd := t.TempDir()
	agentsDir := filepath.Join(cwd, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// name + description frontmatter are required for the agent to be
	// discovered (skills.ParseAgentFromMarkdown rejects files without both).
	agentMd := "---\nname: helper\ndescription: Helps with things\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(agentsDir, "helper.md"), []byte(agentMd), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "agents")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gopher agents failed: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "helper") {
		t.Errorf("expected 'helper' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "Project agents:") {
		t.Errorf("expected 'Project agents:' group, got:\n%s", got)
	}
}

// selfPath returns the directory of this test file.
func selfPath(t *testing.T) string {
	t.Helper()
	// Use the known package path relative to the module root.
	// runtime.Caller would work but is fragile in some test runners.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "agents_integration_test.go")
}
