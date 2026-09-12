package install_github_app

import (
	"net/url"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func parseYAML(t *testing.T, content string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal([]byte(content), &m); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	return m
}

func TestWorkflowContentIsValidYAML(t *testing.T) {
	parseYAML(t, WorkflowContent)
}

func TestCodeReviewWorkflowIsValidYAML(t *testing.T) {
	parseYAML(t, CodeReviewPluginWorkflowContent)
}

func TestWorkflowContentStructure(t *testing.T) {
	m := parseYAML(t, WorkflowContent)

	t.Run("name", func(t *testing.T) {
		if m["name"] != "Claude Code" {
			t.Errorf("name = %v, want Claude Code", m["name"])
		}
	})

	on, ok := m["on"].(map[string]any)
	if !ok {
		t.Fatalf("on: expected map, got %T", m["on"])
	}

	t.Run("triggers", func(t *testing.T) {
		wantTriggers := map[string][]string{
			"issue_comment":               {"created"},
			"pull_request_review_comment": {"created"},
			"issues":                      {"opened", "assigned"},
			"pull_request_review":         {"submitted"},
		}
		if len(on) != len(wantTriggers) {
			t.Errorf("on: got %d triggers, want %d (%v)", len(on), len(wantTriggers), on)
		}
		for trigger, wantTypes := range wantTriggers {
			entry, ok := on[trigger].(map[string]any)
			if !ok {
				t.Errorf("on.%s: missing or not a map", trigger)
				continue
			}
			types, ok := entry["types"].([]any)
			if !ok {
				t.Errorf("on.%s.types: missing or not a list", trigger)
				continue
			}
			if len(types) != len(wantTypes) {
				t.Errorf("on.%s.types = %v, want %v", trigger, types, wantTypes)
				continue
			}
			for i, want := range wantTypes {
				if types[i] != want {
					t.Errorf("on.%s.types[%d] = %v, want %v", trigger, i, types[i], want)
				}
			}
		}
	})

	jobs, ok := m["jobs"].(map[string]any)
	if !ok {
		t.Fatalf("jobs: expected map, got %T", m["jobs"])
	}
	claude, ok := jobs["claude"].(map[string]any)
	if !ok {
		t.Fatalf("jobs.claude: expected map, got %T", jobs["claude"])
	}

	t.Run("runs-on", func(t *testing.T) {
		if claude["runs-on"] != "ubuntu-latest" {
			t.Errorf("jobs.claude.runs-on = %v, want ubuntu-latest", claude["runs-on"])
		}
	})

	t.Run("permissions", func(t *testing.T) {
		perms, ok := claude["permissions"].(map[string]any)
		if !ok {
			t.Fatalf("jobs.claude.permissions: expected map, got %T", claude["permissions"])
		}
		want := map[string]string{
			"contents":      "read",
			"pull-requests": "read",
			"issues":        "read",
			"id-token":      "write",
			"actions":       "read",
		}
		if len(perms) != len(want) {
			t.Errorf("permissions has %d keys, want %d (%v)", len(perms), len(want), perms)
		}
		for k, v := range want {
			if perms[k] != v {
				t.Errorf("permissions.%s = %v, want %v", k, perms[k], v)
			}
		}
	})
}

func TestWorkflowContentMentionGating(t *testing.T) {
	m := parseYAML(t, WorkflowContent)
	jobs := m["jobs"].(map[string]any)
	claude := jobs["claude"].(map[string]any)
	ifExpr, ok := claude["if"].(string)
	if !ok {
		t.Fatalf("jobs.claude.if: expected string, got %T", claude["if"])
	}

	tests := []struct {
		name string
		want string
	}{
		{"issue_comment body", "github.event_name == 'issue_comment' && contains(github.event.comment.body, '@claude')"},
		{"pull_request_review_comment body", "github.event_name == 'pull_request_review_comment' && contains(github.event.comment.body, '@claude')"},
		{"pull_request_review body", "github.event_name == 'pull_request_review' && contains(github.event.review.body, '@claude')"},
		{"issues body", "contains(github.event.issue.body, '@claude')"},
		{"issues title", "contains(github.event.issue.title, '@claude')"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(ifExpr, tt.want) {
				t.Errorf("if expression missing %q\ngot: %s", tt.want, ifExpr)
			}
		})
	}
}

// TestWorkflowContentSecretPlaceholder pins the exact literal
// `anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}`. The (unported)
// reference setupGitHubActions.ts regex-replaces this exact string when the
// user picks an OAuth token or a custom secret name — reformatting it here
// would silently break that substitution once it lands in Go.
func TestWorkflowContentSecretPlaceholder(t *testing.T) {
	want := "anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}"
	if !strings.Contains(WorkflowContent, want) {
		t.Errorf("WorkflowContent missing secret placeholder %q", want)
	}
	if !strings.Contains(CodeReviewPluginWorkflowContent, want) {
		t.Errorf("CodeReviewPluginWorkflowContent missing secret placeholder %q", want)
	}
}

func TestWorkflowContentActionVersions(t *testing.T) {
	for _, content := range []string{WorkflowContent, CodeReviewPluginWorkflowContent} {
		if !strings.Contains(content, "actions/checkout@v4") {
			t.Errorf("missing actions/checkout@v4 in:\n%s", content)
		}
		if !strings.Contains(content, "anthropics/claude-code-action@v1") {
			t.Errorf("missing anthropics/claude-code-action@v1 in:\n%s", content)
		}
	}
}

func TestCodeReviewWorkflowStructure(t *testing.T) {
	m := parseYAML(t, CodeReviewPluginWorkflowContent)

	t.Run("name", func(t *testing.T) {
		if m["name"] != "Claude Code Review" {
			t.Errorf("name = %v, want Claude Code Review", m["name"])
		}
	})

	on, ok := m["on"].(map[string]any)
	if !ok {
		t.Fatalf("on: expected map, got %T", m["on"])
	}
	t.Run("pull_request types", func(t *testing.T) {
		pr, ok := on["pull_request"].(map[string]any)
		if !ok {
			t.Fatalf("on.pull_request: expected map, got %T", on["pull_request"])
		}
		types, ok := pr["types"].([]any)
		if !ok {
			t.Fatalf("on.pull_request.types: expected list, got %T", pr["types"])
		}
		want := []string{"opened", "synchronize", "ready_for_review", "reopened"}
		if len(types) != len(want) {
			t.Fatalf("on.pull_request.types = %v, want %v", types, want)
		}
		for i, w := range want {
			if types[i] != w {
				t.Errorf("on.pull_request.types[%d] = %v, want %v", i, types[i], w)
			}
		}
	})

	jobs := m["jobs"].(map[string]any)
	job, ok := jobs["claude-review"].(map[string]any)
	if !ok {
		t.Fatalf("jobs.claude-review: expected map, got %T", jobs["claude-review"])
	}

	t.Run("permissions", func(t *testing.T) {
		perms, ok := job["permissions"].(map[string]any)
		if !ok {
			t.Fatalf("permissions: expected map, got %T", job["permissions"])
		}
		want := map[string]string{
			"contents":      "read",
			"pull-requests": "read",
			"issues":        "read",
			"id-token":      "write",
		}
		if len(perms) != len(want) {
			t.Errorf("permissions has %d keys, want %d (%v)", len(perms), len(want), perms)
		}
		for k, v := range want {
			if perms[k] != v {
				t.Errorf("permissions.%s = %v, want %v", k, perms[k], v)
			}
		}
	})

	t.Run("plugin params", func(t *testing.T) {
		for _, want := range []string{
			"plugin_marketplaces", "plugins", "prompt",
			"https://github.com/anthropics/claude-code.git",
			"code-review@claude-code-plugins",
		} {
			if !strings.Contains(CodeReviewPluginWorkflowContent, want) {
				t.Errorf("workflow content missing %q", want)
			}
		}
	})
}

func TestPRTitleAndDocsURL(t *testing.T) {
	if PRTitle != "Add Claude Code GitHub Workflow" {
		t.Errorf("PRTitle = %q, want %q", PRTitle, "Add Claude Code GitHub Workflow")
	}

	u, err := url.Parse(GitHubActionSetupDocsURL)
	if err != nil {
		t.Fatalf("url.Parse(GitHubActionSetupDocsURL): %v", err)
	}
	if u.Scheme != "https" {
		t.Errorf("GitHubActionSetupDocsURL scheme = %q, want https", u.Scheme)
	}
	if u.Host != "github.com" {
		t.Errorf("GitHubActionSetupDocsURL host = %q, want github.com", u.Host)
	}
}

func TestPRBodyStructure(t *testing.T) {
	if PRBody == "" {
		t.Fatal("PRBody is empty")
	}
	for _, want := range []string{
		"## 🤖 Installing Claude Code GitHub App",
		"### Security",
		"allowed_tools: Bash(npm install),Bash(npm run build),Bash(npm run lint),Bash(npm run test)",
		"https://github.com/anthropics/claude-code-action",
	} {
		if !strings.Contains(PRBody, want) {
			t.Errorf("PRBody missing %q", want)
		}
	}

	fenceCount := strings.Count(PRBody, "```")
	if fenceCount%2 != 0 {
		t.Errorf("PRBody has an unbalanced number of ``` fences: %d", fenceCount)
	}
}
