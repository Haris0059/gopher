package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Haris0059/gopher/pkg/tools"
)

// requireInterpreter skips the test if the named binary isn't on PATH —
// the CI/dev box may not have every interpreter installed.
func requireInterpreter(t *testing.T, bin string) {
	t.Helper()
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("%s not found on PATH, skipping", bin)
	}
}

func runREPL(t *testing.T, sessionID, lang, code string) *tools.ToolOutput {
	t.Helper()
	tool := &tools.REPLTool{}
	tc := &tools.ToolContext{CWD: t.TempDir(), SessionID: sessionID}
	input, err := json.Marshal(map[string]any{"language": lang, "code": code})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	out, err := tool.Execute(context.Background(), tc, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return out
}

func TestREPLTool_Basics(t *testing.T) {
	tool := &tools.REPLTool{}

	t.Run("name", func(t *testing.T) {
		if tool.Name() != "REPL" {
			t.Errorf("expected 'REPL', got %q", tool.Name())
		}
	})

	t.Run("is_not_read_only", func(t *testing.T) {
		if tool.IsReadOnly() {
			t.Error("REPLTool should not be read-only")
		}
	})

	t.Run("not_concurrency_safe", func(t *testing.T) {
		if tool.IsConcurrencySafe(json.RawMessage(`{}`)) {
			t.Error("REPLTool should never claim concurrency safety")
		}
	})

	t.Run("denies_in_plan_mode", func(t *testing.T) {
		tc := &tools.ToolContext{CWD: t.TempDir(), PlanMode: true}
		res := tool.CheckPermissions(context.Background(), tc, json.RawMessage(`{"language":"python","code":"1"}`))
		if res.Behavior != "deny" {
			t.Errorf("expected deny in plan mode, got %q", res.Behavior)
		}
	})

	t.Run("passthrough_outside_plan_mode", func(t *testing.T) {
		tc := &tools.ToolContext{CWD: t.TempDir()}
		res := tool.CheckPermissions(context.Background(), tc, json.RawMessage(`{"language":"python","code":"1"}`))
		if res.Behavior != "passthrough" {
			t.Errorf("expected passthrough, got %q", res.Behavior)
		}
	})

	t.Run("unknown_language_no_process_spawned", func(t *testing.T) {
		out := runREPL(t, "test-unknown-lang", "cobol", "DISPLAY 'hi'")
		if !out.IsError {
			t.Fatalf("expected error for unsupported language, got: %s", out.Content)
		}
		if !strings.Contains(out.Content, "unsupported REPL language") {
			t.Errorf("expected unsupported-language message, got %q", out.Content)
		}
	})

	t.Run("prompt_mentions_persistence", func(t *testing.T) {
		if !strings.Contains(tool.Prompt(), "persistent") {
			t.Error("expected Prompt() to mention persistence")
		}
	})
}

func TestREPLTool_StatePersists(t *testing.T) {
	cases := []struct {
		lang string
		bin  string
		set  string
		get  string
		want string
	}{
		{"python", "python3", "x = 41", "print(x + 1)", "42"},
		{"node", "node", "let x = 41;", "console.log(x + 1)", "42"},
		{"ruby", "ruby", "x = 41", "puts x + 1", "42"},
		{"bash", "bash", "x=41", "echo $((x + 1))", "42"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.lang, func(t *testing.T) {
			requireInterpreter(t, tc.bin)
			sessionID := "test-persist-" + tc.lang
			defer tools.CloseREPLSessions(sessionID)

			out := runREPL(t, sessionID, tc.lang, tc.set)
			if out.IsError {
				t.Fatalf("setting state failed: %s", out.Content)
			}

			out = runREPL(t, sessionID, tc.lang, tc.get)
			if out.IsError {
				t.Fatalf("reading state failed: %s", out.Content)
			}
			if strings.TrimSpace(out.Content) != tc.want {
				t.Errorf("expected %q, got %q", tc.want, out.Content)
			}
		})
	}
}

func TestREPLTool_ErrorsSurviveSession(t *testing.T) {
	requireInterpreter(t, "python3")
	sessionID := "test-errors-python"
	defer tools.CloseREPLSessions(sessionID)

	out := runREPL(t, sessionID, "python", "x = 5")
	if out.IsError {
		t.Fatalf("unexpected error: %s", out.Content)
	}

	out = runREPL(t, sessionID, "python", "1 / 0")
	if !out.IsError {
		t.Fatalf("expected division-by-zero to be an error, got: %s", out.Content)
	}
	if !strings.Contains(out.Content, "ZeroDivisionError") {
		t.Errorf("expected traceback in output, got %q", out.Content)
	}

	// The session must still be alive and hold prior state after an error.
	out = runREPL(t, sessionID, "python", "print(x)")
	if out.IsError {
		t.Fatalf("session did not survive the prior error: %s", out.Content)
	}
	if strings.TrimSpace(out.Content) != "5" {
		t.Errorf("expected '5', got %q", out.Content)
	}
}

func TestREPLTool_JSErrorsSurviveSession(t *testing.T) {
	requireInterpreter(t, "node")
	sessionID := "test-errors-node"
	defer tools.CloseREPLSessions(sessionID)

	out := runREPL(t, sessionID, "node", "let x = 5;")
	if out.IsError {
		t.Fatalf("unexpected error: %s", out.Content)
	}

	out = runREPL(t, sessionID, "node", "throw new Error('boom')")
	if !out.IsError {
		t.Fatalf("expected thrown error to surface as error, got: %s", out.Content)
	}
	if !strings.Contains(out.Content, "boom") {
		t.Errorf("expected error message in output, got %q", out.Content)
	}

	out = runREPL(t, sessionID, "node", "console.log(x)")
	if out.IsError {
		t.Fatalf("session did not survive the prior error: %s", out.Content)
	}
	if strings.TrimSpace(out.Content) != "5" {
		t.Errorf("expected '5', got %q", out.Content)
	}
}

func TestREPLTool_SessionsAreIsolatedPerSessionID(t *testing.T) {
	requireInterpreter(t, "python3")
	defer tools.CloseREPLSessions("test-iso-a")
	defer tools.CloseREPLSessions("test-iso-b")

	out := runREPL(t, "test-iso-a", "python", "secret = 1")
	if out.IsError {
		t.Fatalf("unexpected error: %s", out.Content)
	}

	out = runREPL(t, "test-iso-b", "python", "print(secret)")
	if !out.IsError {
		t.Fatalf("expected NameError in a fresh session, got: %s", out.Content)
	}
	if !strings.Contains(out.Content, "NameError") {
		t.Errorf("expected NameError, got %q", out.Content)
	}
}

func TestREPLTool_Restart(t *testing.T) {
	requireInterpreter(t, "python3")
	sessionID := "test-restart-python"
	defer tools.CloseREPLSessions(sessionID)

	out := runREPL(t, sessionID, "python", "y = 99")
	if out.IsError {
		t.Fatalf("unexpected error: %s", out.Content)
	}

	tool := &tools.REPLTool{}
	tc := &tools.ToolContext{CWD: t.TempDir(), SessionID: sessionID}
	input, _ := json.Marshal(map[string]any{"language": "python", "code": "print(y)", "restart": true})
	out2, err := tool.Execute(context.Background(), tc, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out2.IsError {
		t.Fatalf("expected restart to clear state, got: %s", out2.Content)
	}
	if !strings.Contains(out2.Content, "NameError") {
		t.Errorf("expected NameError after restart, got %q", out2.Content)
	}
}

func TestREPLTool_Timeout(t *testing.T) {
	requireInterpreter(t, "python3")
	sessionID := "test-timeout-python"
	defer tools.CloseREPLSessions(sessionID)

	tool := &tools.REPLTool{}
	tc := &tools.ToolContext{CWD: t.TempDir(), SessionID: sessionID}
	input, _ := json.Marshal(map[string]any{
		"language": "python",
		"code":     "import time\nwhile True:\n    time.sleep(0.05)\n",
		"timeout":  500,
	})

	start := time.Now()
	out, err := tool.Execute(context.Background(), tc, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("timeout took too long to fire: %s", elapsed)
	}
	if !out.IsError {
		t.Fatalf("expected timeout to be reported as an error, got: %s", out.Content)
	}
	if !strings.Contains(out.Content, "timed out") {
		t.Errorf("expected timeout message, got %q", out.Content)
	}

	// The next call must start a fresh session rather than hang against the
	// killed process.
	out = runREPL(t, sessionID, "python", "print('fresh')")
	if out.IsError {
		t.Fatalf("expected a fresh session after timeout, got: %s", out.Content)
	}
	if strings.TrimSpace(out.Content) != "fresh" {
		t.Errorf("expected 'fresh', got %q", out.Content)
	}
}

func TestREPLTool_CloseAllSessions(t *testing.T) {
	requireInterpreter(t, "python3")
	sessionID := "test-close-all-python"

	out := runREPL(t, sessionID, "python", "1 + 1")
	if out.IsError {
		t.Fatalf("unexpected error: %s", out.Content)
	}

	tools.CloseAllREPLSessions()

	// After closing everything, the same session ID should start clean
	// (no leftover state, no leftover process to hang on).
	out = runREPL(t, sessionID, "python", "print('x' in dir())")
	if out.IsError {
		t.Fatalf("unexpected error after CloseAllREPLSessions: %s", out.Content)
	}
	if strings.TrimSpace(out.Content) != "False" {
		t.Errorf("expected a clean namespace, got %q", out.Content)
	}
	tools.CloseREPLSessions(sessionID)
}

func TestREPLTool_DeprecatedCommandAlias(t *testing.T) {
	requireInterpreter(t, "python3")
	sessionID := "test-alias-python"
	defer tools.CloseREPLSessions(sessionID)

	tool := &tools.REPLTool{}
	tc := &tools.ToolContext{CWD: t.TempDir(), SessionID: sessionID}
	input, _ := json.Marshal(map[string]any{"language": "python", "command": "print(1 + 1)"})
	out, err := tool.Execute(context.Background(), tc, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.IsError {
		t.Fatalf("unexpected tool error: %s", out.Content)
	}
	if strings.TrimSpace(out.Content) != "2" {
		t.Errorf("expected '2', got %q", out.Content)
	}
}

func TestREPLTool_MissingCode(t *testing.T) {
	tool := &tools.REPLTool{}
	tc := &tools.ToolContext{CWD: t.TempDir()}
	input := json.RawMessage(`{"language":"python"}`)
	out, err := tool.Execute(context.Background(), tc, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.IsError {
		t.Fatal("expected error for missing code")
	}
}

func TestREPLTool_IsEnabled(t *testing.T) {
	tool := &tools.REPLTool{}
	t.Setenv("CLAUDE_CODE_REPL", "")
	t.Setenv("GOPHER_REPL", "")
	if !tool.IsEnabled() {
		t.Error("expected REPL enabled by default")
	}

	t.Setenv("CLAUDE_CODE_REPL", "0")
	if tool.IsEnabled() {
		t.Error("expected REPL disabled when CLAUDE_CODE_REPL=0")
	}
	t.Setenv("CLAUDE_CODE_REPL", "")

	t.Setenv("GOPHER_REPL", "false")
	if tool.IsEnabled() {
		t.Error("expected REPL disabled when GOPHER_REPL=false")
	}
}

func TestREPLTool_ConcurrentSessionsDoNotDeadlock(t *testing.T) {
	requireInterpreter(t, "python3")
	sessionID := "test-concurrent-python"
	defer tools.CloseREPLSessions(sessionID)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 3; i++ {
			out := runREPL(t, sessionID, "python", fmt.Sprintf("v%d = %d", i, i))
			if out.IsError {
				t.Errorf("unexpected error: %s", out.Content)
			}
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("sequential REPL calls against the same session deadlocked")
	}
}
