package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAgentOverrides_Basic(t *testing.T) {
	all := []AgentDefinition{
		{AgentType: "Explore", Source: AgentSourceBuiltIn},
		{AgentType: "Explore", Source: AgentSourceProject}, // overrides built-in
		{AgentType: "Plan", Source: AgentSourceBuiltIn},
	}
	active := []AgentDefinition{
		{AgentType: "Explore", Source: AgentSourceProject},
		{AgentType: "Plan", Source: AgentSourceBuiltIn},
	}

	resolved := ResolveAgentOverrides(all, active)
	if len(resolved) != 3 {
		t.Fatalf("expected 3 resolved, got %d", len(resolved))
	}

	// Built-in Explore should be overridden by project
	if resolved[0].OverriddenBy != AgentSourceProject {
		t.Errorf("built-in Explore should be overridden by project, got %q", resolved[0].OverriddenBy)
	}
	// Project Explore is active — not overridden
	if resolved[1].OverriddenBy != "" {
		t.Errorf("project Explore should not be overridden, got %q", resolved[1].OverriddenBy)
	}
	// Plan is active built-in — not overridden
	if resolved[2].OverriddenBy != "" {
		t.Errorf("Plan should not be overridden, got %q", resolved[2].OverriddenBy)
	}
}

func TestResolveAgentOverrides_Dedup(t *testing.T) {
	// Duplicate from worktree
	all := []AgentDefinition{
		{AgentType: "custom", Source: AgentSourceProject},
		{AgentType: "custom", Source: AgentSourceProject}, // duplicate
	}
	active := []AgentDefinition{
		{AgentType: "custom", Source: AgentSourceProject},
	}
	resolved := ResolveAgentOverrides(all, active)
	if len(resolved) != 1 {
		t.Errorf("should deduplicate, got %d", len(resolved))
	}
}

func TestGetAgentMemoryDir(t *testing.T) {
	cwd := "/projects/my-app"

	userDir := GetAgentMemoryDir("Explore", AgentMemoryUser, cwd)
	home, _ := os.UserHomeDir()
	if userDir != filepath.Join(home, ".claude", "agent-memory", "Explore") {
		t.Errorf("user dir = %q", userDir)
	}

	projDir := GetAgentMemoryDir("Plan", AgentMemoryProject, cwd)
	if projDir != filepath.Join(cwd, ".claude", "agent-memory", "Plan") {
		t.Errorf("project dir = %q", projDir)
	}

	localDir := GetAgentMemoryDir("my-plugin:agent", AgentMemoryLocal, cwd)
	if localDir != filepath.Join(cwd, ".claude", "agent-memory-local", "my-plugin-agent") {
		t.Errorf("local dir = %q (colon should be replaced)", localDir)
	}
}

func TestAgentColorManager(t *testing.T) {
	m := NewAgentColorManager()

	c1 := m.AssignColor("Explore")
	c2 := m.AssignColor("Plan")
	c3 := m.AssignColor("Explore") // should be same as c1

	if c1 == "" {
		t.Error("color should not be empty")
	}
	if c1 == c2 {
		t.Error("different agents should get different colors")
	}
	if c1 != c3 {
		t.Error("same agent should get same color")
	}
}

func TestAgentSourceGroups(t *testing.T) {
	// Source: agentDisplay.ts:24-32 — exact order and membership.
	want := []AgentSourceGroup{
		{Label: "User agents", Source: AgentSourceUser},
		{Label: "Project agents", Source: AgentSourceProject},
		{Label: "Local agents", Source: AgentSourceLocal},
		{Label: "Managed agents", Source: AgentSourcePolicy},
		{Label: "Plugin agents", Source: AgentSourcePlugin},
		{Label: "CLI arg agents", Source: AgentSourceFlag},
		{Label: "Built-in agents", Source: AgentSourceBuiltIn},
	}
	if len(AgentSourceGroups) != len(want) {
		t.Fatalf("expected %d groups, got %d", len(want), len(AgentSourceGroups))
	}
	for i, g := range want {
		if AgentSourceGroups[i] != g {
			t.Errorf("group %d = %+v, want %+v", i, AgentSourceGroups[i], g)
		}
	}
}

func TestOverrideSourceLabel(t *testing.T) {
	cases := []struct {
		source AgentSource
		want   string
	}{
		{AgentSourceUser, "user"},
		{AgentSourceProject, "project"},
		{AgentSourceLocal, "local"},
		{AgentSourcePolicy, "managed"},
		{AgentSourcePlugin, "plugin"},
		{AgentSourceFlag, "flag"},
		{AgentSourceBuiltIn, "built-in"},
		{AgentSource("unknown-source"), "unknown-source"},
	}
	for _, c := range cases {
		if got := OverrideSourceLabel(c.source); got != c.want {
			t.Errorf("OverrideSourceLabel(%q) = %q, want %q", c.source, got, c.want)
		}
	}
}

func TestCompareAgentsByName(t *testing.T) {
	a := AgentDefinition{AgentType: "Explore"}
	b := AgentDefinition{AgentType: "explore"}
	if got := CompareAgentsByName(a, b); got != 0 {
		t.Errorf("case-insensitive equal expected 0, got %d", got)
	}

	c := AgentDefinition{AgentType: "alpha"}
	d := AgentDefinition{AgentType: "Beta"}
	if got := CompareAgentsByName(c, d); got >= 0 {
		t.Errorf("alpha should sort before Beta, got %d", got)
	}
	if got := CompareAgentsByName(d, c); got <= 0 {
		t.Errorf("Beta should sort after alpha, got %d", got)
	}
}

func TestResolveAgentModelDisplay(t *testing.T) {
	if got := ResolveAgentModelDisplay(AgentDefinition{Model: "sonnet"}); got != "sonnet" {
		t.Errorf("explicit model should pass through, got %q", got)
	}
	if got := ResolveAgentModelDisplay(AgentDefinition{Model: "inherit"}); got != "inherit" {
		t.Errorf("inherit should pass through, got %q", got)
	}
	if got := ResolveAgentModelDisplay(AgentDefinition{}); got != "" {
		t.Errorf("unset model should return empty, got %q", got)
	}
}

func TestFormatAgentListing(t *testing.T) {
	cases := []struct {
		name  string
		agent ResolvedAgent
		want  string
	}{
		{
			name:  "name only",
			agent: ResolvedAgent{AgentDefinition: AgentDefinition{AgentType: "reviewer"}},
			want:  "reviewer",
		},
		{
			name:  "name and model",
			agent: ResolvedAgent{AgentDefinition: AgentDefinition{AgentType: "reviewer", Model: "sonnet"}},
			want:  "reviewer · sonnet",
		},
		{
			name:  "name and memory",
			agent: ResolvedAgent{AgentDefinition: AgentDefinition{AgentType: "reviewer", Memory: AgentMemoryUser}},
			want:  "reviewer · user memory",
		},
		{
			name:  "name, model, and memory",
			agent: ResolvedAgent{AgentDefinition: AgentDefinition{AgentType: "reviewer", Model: "sonnet", Memory: AgentMemoryProject}},
			want:  "reviewer · sonnet · project memory",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FormatAgentListing(c.agent); got != c.want {
				t.Errorf("FormatAgentListing() = %q, want %q", got, c.want)
			}
		})
	}
}
