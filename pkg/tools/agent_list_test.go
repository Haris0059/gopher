package tools

import (
	"strings"
	"testing"

	"github.com/Haris0059/gopher/pkg/skills"
)

func TestAgentToolsDescription(t *testing.T) {
	cases := []struct {
		name string
		def  skills.AgentDefinition
		want string
	}{
		{
			name: "no restrictions",
			def:  skills.AgentDefinition{},
			want: "All tools",
		},
		{
			name: "allowlist only",
			def:  skills.AgentDefinition{Tools: []string{"Read", "Grep"}},
			want: "Read, Grep",
		},
		{
			name: "denylist only",
			def:  skills.AgentDefinition{DisallowedTools: []string{"Edit", "Write"}},
			want: "All tools except Edit, Write",
		},
		{
			name: "allowlist filtered by denylist",
			def:  skills.AgentDefinition{Tools: []string{"Read", "Grep", "Edit"}, DisallowedTools: []string{"Edit"}},
			want: "Read, Grep",
		},
		{
			name: "allowlist fully denied",
			def:  skills.AgentDefinition{Tools: []string{"Edit"}, DisallowedTools: []string{"Edit"}},
			want: "None",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := agentToolsDescription(c.def); got != c.want {
				t.Errorf("agentToolsDescription() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFormatAgentLine(t *testing.T) {
	def := skills.AgentDefinition{
		AgentType: "Explore",
		WhenToUse: "Fast codebase search.",
		Model:     "haiku",
	}
	want := `- Explore: Fast codebase search. (Tools: All tools)`
	if got := FormatAgentLine(def); got != want {
		t.Errorf("FormatAgentLine() = %q, want %q", got, want)
	}
}

func TestBuildAgentListSection(t *testing.T) {
	t.Run("empty list falls back to static sentence", func(t *testing.T) {
		got := BuildAgentListSection(nil)
		if got != staticAgentListSection {
			t.Errorf("BuildAgentListSection(nil) = %q, want %q", got, staticAgentListSection)
		}
	})

	t.Run("real list is rendered", func(t *testing.T) {
		agents := []skills.AgentDefinition{
			{AgentType: "Explore", WhenToUse: "Search the codebase."},
			{AgentType: "general-purpose", WhenToUse: "General tasks."},
		}
		got := BuildAgentListSection(agents)
		if !strings.HasPrefix(got, "Available agent types and the tools they have access to:\n") {
			t.Errorf("BuildAgentListSection() missing header, got %q", got)
		}
		if !strings.Contains(got, "- Explore: Search the codebase. (Tools: All tools)") {
			t.Errorf("BuildAgentListSection() missing Explore line, got %q", got)
		}
		if !strings.Contains(got, "- general-purpose: General tasks. (Tools: All tools)") {
			t.Errorf("BuildAgentListSection() missing general-purpose line, got %q", got)
		}
	})

	t.Run("CLAUDE_CODE_AGENT_LIST_IN_MESSAGES opts into the static sentence", func(t *testing.T) {
		orig := getEnvVar
		defer func() { getEnvVar = orig }()
		getEnvVar = func(key string) string {
			if key == "CLAUDE_CODE_AGENT_LIST_IN_MESSAGES" {
				return "true"
			}
			return ""
		}
		agents := []skills.AgentDefinition{{AgentType: "Explore", WhenToUse: "Search."}}
		got := BuildAgentListSection(agents)
		if got != staticAgentListSection {
			t.Errorf("BuildAgentListSection() with env override = %q, want %q", got, staticAgentListSection)
		}
	})
}
