package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Haris0059/gopher/pkg/tools"
)

// resetBriefState clears the package-level Brief feature gate around a test
// so global state does not leak across subtests.
func resetBriefState(t *testing.T) {
	t.Helper()
	tools.SetKairosActive(false)
	tools.SetUserMsgOptIn(false)
	t.Cleanup(func() {
		tools.SetKairosActive(false)
		tools.SetUserMsgOptIn(false)
	})
}

func TestBriefTool(t *testing.T) {
	// Source: tools/BriefTool/BriefTool.ts, tools/BriefTool/prompt.ts

	tool := &tools.BriefTool{}

	// Source: prompt.ts:1 — BRIEF_TOOL_NAME = 'SendUserMessage'
	t.Run("name", func(t *testing.T) {
		if tool.Name() != "SendUserMessage" {
			t.Errorf("Name() = %q, want %q", tool.Name(), "SendUserMessage")
		}
	})

	// Source: prompt.ts:2 — LEGACY_BRIEF_TOOL_NAME = 'Brief'
	t.Run("legacy_alias", func(t *testing.T) {
		aliases := tools.GetAliases(tool)
		if len(aliases) != 1 || aliases[0] != "Brief" {
			t.Errorf("Aliases() = %v, want [Brief]", aliases)
		}
	})

	// Source: prompt.ts:4 — DESCRIPTION = 'Send a message to the user'
	t.Run("description_matches_ts", func(t *testing.T) {
		want := "Send a message to the user"
		if tool.Description() != want {
			t.Errorf("Description() = %q, want %q", tool.Description(), want)
		}
	})

	// Source: BriefTool.ts:117
	t.Run("is_read_only", func(t *testing.T) {
		if !tool.IsReadOnly() {
			t.Error("BriefTool should be read-only")
		}
	})

	// Source: BriefTool.ts:114-116
	t.Run("is_concurrency_safe", func(t *testing.T) {
		if !tools.CheckConcurrencySafe(tool, nil) {
			t.Error("BriefTool should be concurrency-safe")
		}
	})

	// Source: BriefTool.ts:141
	t.Run("max_result_size_chars", func(t *testing.T) {
		if got := tools.GetMaxResultSizeChars(tool); got != 100_000 {
			t.Errorf("GetMaxResultSizeChars() = %d, want 100000", got)
		}
	})

	// Source: BriefTool.ts:69-70
	t.Run("search_hint", func(t *testing.T) {
		want := "send a message to the user — your primary visible output channel"
		if got := tools.GetSearchHint(tool); got != want {
			t.Errorf("SearchHint() = %q, want %q", got, want)
		}
	})

	t.Run("prompt_matches_ts", func(t *testing.T) {
		if tools.GetToolPrompt(tool) != tools.BriefToolPrompt {
			t.Error("Prompt() does not match BriefToolPrompt constant")
		}
	})

	// Source: BriefTool.ts:22-38
	t.Run("schema", func(t *testing.T) {
		schema := tool.InputSchema()
		var parsed map[string]interface{}
		if err := json.Unmarshal(schema, &parsed); err != nil {
			t.Fatalf("schema is not valid JSON: %v", err)
		}
		if parsed["additionalProperties"] != false {
			t.Errorf("additionalProperties = %v, want false", parsed["additionalProperties"])
		}
		required, _ := parsed["required"].([]interface{})
		reqSet := map[string]bool{}
		for _, r := range required {
			reqSet[r.(string)] = true
		}
		if !reqSet["message"] || !reqSet["status"] {
			t.Errorf("required = %v, want message and status required", required)
		}
		if reqSet["attachments"] {
			t.Error("attachments must stay optional — resumed sessions replay pre-attachment outputs")
		}
		props := parsed["properties"].(map[string]interface{})
		status := props["status"].(map[string]interface{})
		enum, _ := status["enum"].([]interface{})
		if len(enum) != 2 || enum[0] != "normal" || enum[1] != "proactive" {
			t.Errorf("status enum = %v, want [normal proactive]", enum)
		}
	})

	t.Run("gating", func(t *testing.T) {
		t.Run("disabled_by_default", func(t *testing.T) {
			resetBriefState(t)
			if tool.IsEnabled() {
				t.Error("BriefTool should be disabled by default")
			}
		})

		t.Run("enabled_when_kairos_active", func(t *testing.T) {
			resetBriefState(t)
			tools.SetKairosActive(true)
			if !tool.IsEnabled() {
				t.Error("BriefTool should be enabled when Kairos is active")
			}
		})

		t.Run("disabled_when_opt_in_but_not_entitled", func(t *testing.T) {
			resetBriefState(t)
			tools.SetUserMsgOptIn(true)
			if tool.IsEnabled() {
				t.Error("BriefTool should stay disabled: opt-in alone is not entitlement")
			}
		})

		t.Run("enabled_when_opt_in_and_env_brief", func(t *testing.T) {
			resetBriefState(t)
			t.Setenv("CLAUDE_CODE_BRIEF", "1")
			tools.SetUserMsgOptIn(true)
			if !tool.IsEnabled() {
				t.Error("BriefTool should be enabled: opt-in plus CLAUDE_CODE_BRIEF entitlement")
			}
		})

		t.Run("registry_all_reflects_gate", func(t *testing.T) {
			resetBriefState(t)
			reg := tools.NewRegistry()
			reg.Register(&tools.BriefTool{})

			found := func() bool {
				for _, tl := range reg.All() {
					if tl.Name() == "SendUserMessage" {
						return true
					}
				}
				return false
			}

			if found() {
				t.Error("disabled BriefTool should be excluded from All()")
			}
			tools.SetKairosActive(true)
			if !found() {
				t.Error("enabled BriefTool should be included in All()")
			}
		})
	})

	t.Run("execute", func(t *testing.T) {
		t.Run("invalid_json", func(t *testing.T) {
			out, err := tool.Execute(context.Background(), &tools.ToolContext{}, json.RawMessage(`not json`))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !out.IsError {
				t.Error("expected IsError for invalid JSON")
			}
		})

		t.Run("no_attachments", func(t *testing.T) {
			input, _ := json.Marshal(map[string]any{"message": "hello", "status": "normal"})
			out, err := tool.Execute(context.Background(), &tools.ToolContext{}, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.IsError {
				t.Fatalf("unexpected error output: %s", out.Content)
			}
			want := "Message delivered to user."
			if out.Content != want {
				t.Errorf("Content = %q, want %q", out.Content, want)
			}
			disp, ok := out.Display.(tools.BriefDisplay)
			if !ok {
				t.Fatalf("Display is %T, want tools.BriefDisplay", out.Display)
			}
			if disp.Message != "hello" || disp.Status != "normal" {
				t.Errorf("Display = %+v, want Message=hello Status=normal", disp)
			}
			if disp.SentAt == "" {
				t.Error("SentAt should be set")
			}
		})

		t.Run("one_attachment_singular", func(t *testing.T) {
			dir := t.TempDir()
			imgPath := filepath.Join(dir, "shot.png")
			if err := os.WriteFile(imgPath, []byte("fakepng"), 0644); err != nil {
				t.Fatal(err)
			}
			input, _ := json.Marshal(map[string]any{
				"message":     "see attached",
				"status":      "normal",
				"attachments": []string{"shot.png"},
			})
			out, err := tool.Execute(context.Background(), &tools.ToolContext{CWD: dir}, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.IsError {
				t.Fatalf("unexpected error output: %s", out.Content)
			}
			want := "Message delivered to user. (1 attachment included)"
			if out.Content != want {
				t.Errorf("Content = %q, want %q", out.Content, want)
			}
			disp := out.Display.(tools.BriefDisplay)
			if len(disp.Attachments) != 1 {
				t.Fatalf("Attachments = %v, want 1 entry", disp.Attachments)
			}
			att := disp.Attachments[0]
			if att.Path != imgPath {
				t.Errorf("Path = %q, want %q (relative resolved against cwd)", att.Path, imgPath)
			}
			if !att.IsImage {
				t.Error("shot.png should be detected as an image")
			}
			if att.Size != int64(len("fakepng")) {
				t.Errorf("Size = %d, want %d", att.Size, len("fakepng"))
			}
		})

		t.Run("two_attachments_plural", func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"a.txt", "b.txt"} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			input, _ := json.Marshal(map[string]any{
				"message":     "two files",
				"status":      "proactive",
				"attachments": []string{"a.txt", "b.txt"},
			})
			out, err := tool.Execute(context.Background(), &tools.ToolContext{CWD: dir}, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := "Message delivered to user. (2 attachments included)"
			if out.Content != want {
				t.Errorf("Content = %q, want %q", out.Content, want)
			}
			disp := out.Display.(tools.BriefDisplay)
			for _, att := range disp.Attachments {
				if att.IsImage {
					t.Errorf("%s should not be detected as an image", att.Path)
				}
			}
		})

		t.Run("attachment_missing", func(t *testing.T) {
			dir := t.TempDir()
			input, _ := json.Marshal(map[string]any{
				"message":     "oops",
				"status":      "normal",
				"attachments": []string{"nope.png"},
			})
			out, err := tool.Execute(context.Background(), &tools.ToolContext{CWD: dir}, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !out.IsError {
				t.Fatal("expected IsError for a missing attachment")
			}
			want := `Attachment "nope.png" does not exist. Current working directory: ` + dir + `.`
			if out.Content != want {
				t.Errorf("Content = %q, want %q", out.Content, want)
			}
		})

		t.Run("attachment_is_directory", func(t *testing.T) {
			dir := t.TempDir()
			sub := filepath.Join(dir, "subdir")
			if err := os.Mkdir(sub, 0755); err != nil {
				t.Fatal(err)
			}
			input, _ := json.Marshal(map[string]any{
				"message":     "oops",
				"status":      "normal",
				"attachments": []string{"subdir"},
			})
			out, err := tool.Execute(context.Background(), &tools.ToolContext{CWD: dir}, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !out.IsError {
				t.Fatal("expected IsError for a directory attachment")
			}
			want := `Attachment "subdir" is not a regular file.`
			if out.Content != want {
				t.Errorf("Content = %q, want %q", out.Content, want)
			}
		})

		t.Run("attachment_tilde_expansion", func(t *testing.T) {
			home, err := os.UserHomeDir()
			if err != nil {
				t.Skip("no home directory available")
			}
			tmp, err := os.MkdirTemp(home, "brief-test-*")
			if err != nil {
				t.Skip("cannot create temp dir under home")
			}
			t.Cleanup(func() { os.RemoveAll(tmp) })
			f := filepath.Join(tmp, "note.txt")
			if err := os.WriteFile(f, []byte("hi"), 0644); err != nil {
				t.Fatal(err)
			}
			rel := "~/" + filepath.Base(tmp) + "/note.txt"
			input, _ := json.Marshal(map[string]any{
				"message":     "hi",
				"status":      "normal",
				"attachments": []string{rel},
			})
			out, err := tool.Execute(context.Background(), &tools.ToolContext{}, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.IsError {
				t.Fatalf("unexpected error output: %s", out.Content)
			}
			disp := out.Display.(tools.BriefDisplay)
			if len(disp.Attachments) != 1 || disp.Attachments[0].Path != f {
				t.Errorf("Attachments = %+v, want a single entry with Path=%q", disp.Attachments, f)
			}
		})
	})
}

func TestToolRegistryAliasResolution(t *testing.T) {
	resetBriefState(t)
	tools.SetKairosActive(true) // BriefTool must be enabled to appear in ToolDefinitions()

	reg := tools.NewRegistry()
	reg.Register(&tools.BriefTool{})
	reg.Register(&tools.BashTool{})

	t.Run("get_resolves_alias", func(t *testing.T) {
		byCanonical := reg.Get("SendUserMessage")
		byAlias := reg.Get("Brief")
		if byCanonical == nil || byAlias == nil || byCanonical != byAlias {
			t.Errorf("Get(\"Brief\") = %v, Get(\"SendUserMessage\") = %v, want same non-nil tool", byAlias, byCanonical)
		}
	})

	t.Run("real_tool_name_beats_alias", func(t *testing.T) {
		if got := reg.Get("Bash"); got == nil || got.Name() != "Bash" {
			t.Errorf("Get(\"Bash\") = %v, want the Bash tool", got)
		}
	})

	t.Run("one_entry_per_tool_in_definitions", func(t *testing.T) {
		defs := reg.ToolDefinitions()
		seen := map[string]int{}
		for _, d := range defs {
			seen[d.Name]++
		}
		if seen["SendUserMessage"] != 1 {
			t.Errorf("ToolDefinitions() has %d SendUserMessage entries, want 1", seen["SendUserMessage"])
		}
		if _, ok := seen["Brief"]; ok {
			t.Error("ToolDefinitions() should not emit a separate entry for the alias")
		}
	})
}
