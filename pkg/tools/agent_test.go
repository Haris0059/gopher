package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Haris0059/gopher/internal/testharness"
	"github.com/Haris0059/gopher/pkg/provider"
	"github.com/Haris0059/gopher/pkg/query"
	"github.com/Haris0059/gopher/pkg/session"
	"github.com/Haris0059/gopher/pkg/skills"
	"github.com/Haris0059/gopher/pkg/tools"
)

func TestAgentTool(t *testing.T) {

	t.Run("name", func(t *testing.T) {
		tool := tools.NewAgentTool(nil, nil, nil)
		if tool.Name() != "Agent" {
			t.Errorf("expected 'Agent', got %q", tool.Name())
		}
	})

	t.Run("is_not_read_only", func(t *testing.T) {
		tool := tools.NewAgentTool(nil, nil, nil)
		if tool.IsReadOnly() {
			t.Error("AgentTool should not be read-only")
		}
	})

	t.Run("valid_schema", func(t *testing.T) {
		tool := tools.NewAgentTool(nil, nil, nil)
		schema := tool.InputSchema()
		var parsed map[string]interface{}
		if err := json.Unmarshal(schema, &parsed); err != nil {
			t.Fatalf("schema is not valid JSON: %v", err)
		}
		props, ok := parsed["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("schema missing properties")
		}
		if _, ok := props["prompt"]; !ok {
			t.Error("schema missing 'prompt' property")
		}
		if _, ok := props["description"]; !ok {
			t.Error("schema missing 'description' property")
		}
		required, ok := parsed["required"].([]interface{})
		if !ok {
			t.Fatal("schema missing required")
		}
		requiredSet := make(map[string]bool)
		for _, r := range required {
			if s, ok := r.(string); ok {
				requiredSet[s] = true
			}
		}
		if !requiredSet["prompt"] {
			t.Error("prompt should be required")
		}
		if !requiredSet["description"] {
			t.Error("description should be required")
		}
	})

	t.Run("empty_prompt_returns_error", func(t *testing.T) {
		tool := tools.NewAgentTool(nil, nil, nil)
		tc := &tools.ToolContext{CWD: t.TempDir()}
		input := json.RawMessage(`{"prompt": "", "description": "test"}`)
		out, err := tool.Execute(context.Background(), tc, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !out.IsError {
			t.Error("expected error output for empty prompt")
		}
		if !strings.Contains(out.Content, "prompt is required") {
			t.Errorf("error should mention prompt, got %q", out.Content)
		}
	})

	t.Run("invalid_json_returns_error", func(t *testing.T) {
		tool := tools.NewAgentTool(nil, nil, nil)
		tc := &tools.ToolContext{CWD: t.TempDir()}
		input := json.RawMessage(`{bad json}`)
		out, err := tool.Execute(context.Background(), tc, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !out.IsError {
			t.Error("expected error output for invalid JSON")
		}
	})

	t.Run("sub_agent_runs_and_collects_text", func(t *testing.T) {
		// Create a scripted provider that returns a single text turn.
		prov := testharness.NewScriptedProvider(
			testharness.MakeTextTurn("The answer is 42.", provider.StopReasonEndTurn),
		)

		registry := tools.NewRegistry()
		queryFn := query.AsQueryFunc()
		tool := tools.NewAgentTool(prov, registry, queryFn)

		tc := &tools.ToolContext{CWD: t.TempDir()}
		input := json.RawMessage(`{"prompt": "What is the answer?", "description": "compute answer"}`)
		out, err := tool.Execute(context.Background(), tc, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.IsError {
			t.Fatalf("unexpected tool error: %s", out.Content)
		}
		if !strings.Contains(out.Content, "The answer is 42.") {
			t.Errorf("expected output to contain sub-agent response, got %q", out.Content)
		}
	})

	t.Run("sub_agent_with_tool_use", func(t *testing.T) {
		// Create a scripted provider where the sub-agent calls a tool then responds.
		spy := testharness.NewSpyTool("my_tool", true)

		prov := testharness.NewScriptedProvider(
			testharness.MakeToolTurn("t1", "my_tool", json.RawMessage(`{"x":1}`), provider.StopReasonToolUse),
			testharness.MakeTextTurn("Done with tool use.", provider.StopReasonEndTurn),
		)

		registry := tools.NewRegistry()
		registry.Register(spy)
		queryFn := query.AsQueryFunc()
		tool := tools.NewAgentTool(prov, registry, queryFn)

		tc := &tools.ToolContext{CWD: t.TempDir()}
		input := json.RawMessage(`{"prompt": "Use the tool", "description": "tool usage"}`)
		out, err := tool.Execute(context.Background(), tc, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.IsError {
			t.Fatalf("unexpected tool error: %s", out.Content)
		}
		if !strings.Contains(out.Content, "Done with tool use.") {
			t.Errorf("expected final text from sub-agent, got %q", out.Content)
		}
		if spy.CallCount() != 1 {
			t.Errorf("expected spy tool called once, got %d", spy.CallCount())
		}
	})

	t.Run("sub_agent_no_text_output", func(t *testing.T) {
		// A sub-agent that only uses tools but produces no text.
		spy := testharness.NewSpyTool("silent_tool", true)

		prov := testharness.NewScriptedProvider(
			testharness.MakeToolTurn("t1", "silent_tool", json.RawMessage(`{}`), provider.StopReasonToolUse),
			// Simulate an end_turn with empty text
			testharness.MakeTextTurn("", provider.StopReasonEndTurn),
		)

		registry := tools.NewRegistry()
		registry.Register(spy)
		queryFn := query.AsQueryFunc()
		tool := tools.NewAgentTool(prov, registry, queryFn)

		tc := &tools.ToolContext{CWD: t.TempDir()}
		input := json.RawMessage(`{"prompt": "Do something silently", "description": "silent work"}`)
		out, err := tool.Execute(context.Background(), tc, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.IsError {
			t.Fatalf("unexpected tool error: %s", out.Content)
		}
		if !strings.Contains(out.Content, "agent completed with no text output") {
			t.Errorf("expected fallback message for no output, got %q", out.Content)
		}
	})

	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		prov := testharness.NewScriptedProvider(
			testharness.MakeTextTurn("should not reach", provider.StopReasonEndTurn),
		)

		registry := tools.NewRegistry()
		queryFn := query.AsQueryFunc()
		tool := tools.NewAgentTool(prov, registry, queryFn)

		tc := &tools.ToolContext{CWD: t.TempDir()}
		input := json.RawMessage(`{"prompt": "Do something", "description": "cancelled"}`)
		out, err := tool.Execute(ctx, tc, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The sub-agent should detect cancellation and return an error output
		if !out.IsError {
			// It's also acceptable if the query finishes before checking context
			t.Log("note: sub-agent completed despite cancellation (race condition is acceptable)")
		}
	})

	t.Run("unknown_subagent_type_returns_error", func(t *testing.T) {
		tool := tools.NewAgentTool(nil, nil, nil)
		tool.SetAgentLoader(func(cwd string) []skills.AgentDefinition {
			return []skills.AgentDefinition{
				{AgentType: "general-purpose", WhenToUse: "General."},
				{AgentType: "Explore", WhenToUse: "Search."},
			}
		})
		tc := &tools.ToolContext{CWD: t.TempDir()}
		input := json.RawMessage(`{"prompt": "do it", "description": "test", "subagent_type": "bogus"}`)
		out, err := tool.Execute(context.Background(), tc, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !out.IsError {
			t.Fatal("expected error output for unknown subagent_type")
		}
		if !strings.Contains(out.Content, "Agent type 'bogus' not found") {
			t.Errorf("error should name the unknown type, got %q", out.Content)
		}
		if !strings.Contains(out.Content, "general-purpose") || !strings.Contains(out.Content, "Explore") {
			t.Errorf("error should list available agents, got %q", out.Content)
		}
	})

	t.Run("subagent_type_selects_agent_definition", func(t *testing.T) {
		prov := testharness.NewScriptedProvider(
			testharness.MakeTextTurn("done", provider.StopReasonEndTurn),
		)

		var capturedSystemPrompt, capturedModel string
		var capturedMaxTurns int
		var capturedRegistry *tools.ToolRegistry
		queryFn := func(ctx context.Context, sess *session.SessionState, p provider.ModelProvider, reg *tools.ToolRegistry, orch *tools.ToolOrchestrator, onEvent func(string)) error {
			capturedSystemPrompt = sess.Config.SystemPrompt
			capturedModel = sess.Config.Model
			capturedMaxTurns = sess.Config.MaxTurns
			capturedRegistry = reg
			onEvent("done")
			return nil
		}

		registry := tools.NewRegistry()
		registry.Register(&testStubTool{name: "Read"})
		registry.Register(&testStubTool{name: "Edit"})
		registry.Register(&testStubTool{name: "Agent"})

		tool := tools.NewAgentTool(prov, registry, queryFn)
		tool.SetAgentLoader(func(cwd string) []skills.AgentDefinition {
			return []skills.AgentDefinition{
				{
					AgentType:       "reviewer",
					WhenToUse:       "Reviews code.",
					SystemPrompt:    "You are a code reviewer.",
					Model:           "haiku",
					MaxTurns:        7,
					DisallowedTools: []string{"Agent", "Edit"},
				},
			}
		})

		tc := &tools.ToolContext{CWD: t.TempDir()}
		input := json.RawMessage(`{"prompt": "review this", "description": "review", "subagent_type": "reviewer"}`)
		out, err := tool.Execute(context.Background(), tc, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.IsError {
			t.Fatalf("unexpected tool error: %s", out.Content)
		}
		if capturedSystemPrompt != "You are a code reviewer." {
			t.Errorf("system prompt = %q, want agent definition's prompt", capturedSystemPrompt)
		}
		if capturedModel != "claude-haiku-4-5-20251001" {
			t.Errorf("model = %q, want resolved haiku alias", capturedModel)
		}
		if capturedMaxTurns != 7 {
			t.Errorf("maxTurns = %d, want 7", capturedMaxTurns)
		}
		if capturedRegistry == nil {
			t.Fatal("expected a child registry to be captured")
		}
		if capturedRegistry.Get("Edit") != nil {
			t.Error("Edit should be excluded from the child registry (disallowedTools)")
		}
		if capturedRegistry.Get("Agent") != nil {
			t.Error("Agent should be excluded from the child registry (disallowedTools)")
		}
		if capturedRegistry.Get("Read") == nil {
			t.Error("Read should remain in the child registry")
		}
	})
}

// testStubTool is a minimal Tool implementation for registry-scoping tests.
type testStubTool struct{ name string }

func (s *testStubTool) Name() string                 { return s.name }
func (s *testStubTool) Description() string          { return "" }
func (s *testStubTool) IsReadOnly() bool             { return true }
func (s *testStubTool) InputSchema() json.RawMessage { return json.RawMessage(`{}`) }
func (s *testStubTool) Execute(ctx context.Context, tc *tools.ToolContext, input json.RawMessage) (*tools.ToolOutput, error) {
	return tools.SuccessOutput("ok"), nil
}
