package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// REPLTool runs code against a persistent interpreter session that survives
// across tool calls, keyed per (session, language) — see repl_session.go.
//
// This is not a port of the TS reference's REPLTool: that tool is an
// ant-only JavaScript VM sandbox that hides Read/Write/Edit/Glob/Grep/
// Bash/NotebookEdit/Agent from the model and re-exposes them as in-VM
// function calls (src/tools/REPLTool/constants.ts, primitiveTools.ts). Its
// implementation file is missing from the leaked checkout, and the feature
// it provides — forcing batched tool calls through a JS sandbox for ant
// telemetry — doesn't fit Gopher. Gopher's REPL is a different, genuinely
// useful capability instead: a stateful interpreter Bash cannot offer.
// Tracked separately as TASKS.md TOOL-10.
type REPLTool struct{}

type replInput struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	// Command is a deprecated alias for Code, kept so any transcripts
	// written against the old one-shot schema still parse.
	Command string `json:"command,omitempty"`
	Restart bool   `json:"restart,omitempty"`
	Timeout int    `json:"timeout,omitempty"` // milliseconds
}

func (t *REPLTool) Name() string { return "REPL" }

func (t *REPLTool) Description() string {
	return "Runs code in a persistent interpreter session (python, node, ruby, or bash). " +
		"State — variables, imports, open handles — is kept alive across calls until the " +
		"session is restarted or closed."
}

func (t *REPLTool) IsReadOnly() bool { return false }

// IsConcurrencySafe is always false: every call mutates shared interpreter
// state, so two calls racing against the same session would interleave.
func (t *REPLTool) IsConcurrencySafe(json.RawMessage) bool { return false }

func (t *REPLTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"language": {"type": "string", "enum": ["python", "node", "ruby", "bash"], "description": "Which persistent interpreter session to run code in."},
			"code": {"type": "string", "description": "Code to execute in the session. Variables and state from prior calls in this session are still available."},
			"restart": {"type": "boolean", "description": "Kill the existing session for this language (if any) and start a fresh one before running code."},
			"timeout": {"type": "integer", "description": "Optional timeout in milliseconds (up to 600000ms / 10 minutes). Default 120000ms. On timeout the session is killed and its state is lost."}
		},
		"required": ["language", "code"],
		"additionalProperties": false
	}`)
}

// CheckPermissions denies REPL in plan mode — it executes arbitrary code,
// mirroring BashTool's plan-mode rejection (ValidateBashCommand).
// Source: BashTool.tsx's plan-mode leg has no distinct REPL equivalent in
// the (missing) reference REPLTool.js, so this reuses Bash's rule.
func (t *REPLTool) CheckPermissions(_ context.Context, tc *ToolContext, input json.RawMessage) PermissionCheckResult {
	if tc != nil && tc.PlanMode {
		return PermissionCheckResult{
			Behavior: "deny",
			Message:  "REPL cannot run in plan mode: it executes arbitrary code, which plan mode disallows.",
		}
	}
	return PermissionCheckResult{Behavior: "passthrough"}
}

// IsEnabled gates the tool off only when explicitly disabled via
// CLAUDE_CODE_REPL=0/false/no (mirrors the TS isEnvDefinedFalsy check in
// tools/REPLTool/constants.ts, though the gate itself now guards a
// different feature). On by default — unlike the ant-only reference tool,
// Gopher's REPL has no USER_TYPE gate.
func (t *REPLTool) IsEnabled() bool {
	return !isEnvDefinedFalsy(os.Getenv("CLAUDE_CODE_REPL")) && !isEnvDefinedFalsy(os.Getenv("GOPHER_REPL"))
}

// Prompt tells the model the session persists across calls.
func (t *REPLTool) Prompt() string {
	return "REPL runs code in a persistent " + strings.Join(SupportedREPLLanguages, "/") + " session per language: " +
		"variables, imports, and open handles from earlier REPL calls in this conversation are " +
		"still there on the next call. Prefer it over Bash's one-shot `python3 -c ...` style when " +
		"you need to build up state across several steps (e.g. load data once, then query it " +
		"repeatedly). Set restart:true to discard a session's state and start over. A call that " +
		"times out kills that session — its state is gone, and the next call starts fresh."
}

func (t *REPLTool) Execute(ctx context.Context, tc *ToolContext, input json.RawMessage) (*ToolOutput, error) {
	var in replInput
	if err := json.Unmarshal(input, &in); err != nil {
		return ErrorOutput("invalid input: " + err.Error()), nil
	}

	code := in.Code
	if code == "" {
		code = in.Command
	}
	if code == "" {
		return ErrorOutput("code is required"), nil
	}

	lang, ok := canonicalLanguage(in.Language)
	if !ok {
		return ErrorOutput(fmt.Sprintf(
			"unsupported REPL language %q; supported languages: %s",
			in.Language, strings.Join(SupportedREPLLanguages, ", "),
		)), nil
	}

	cwd := "."
	sessionID := ""
	if tc != nil {
		if tc.CWD != "" {
			cwd = tc.CWD
		}
		sessionID = tc.SessionID
	}

	timeoutMs := DefaultBashTimeoutMs
	if in.Timeout > 0 {
		timeoutMs = in.Timeout
		if timeoutMs > MaxBashTimeoutMs {
			timeoutMs = MaxBashTimeoutMs
		}
	}

	var (
		sess *replSession
		err  error
	)
	if in.Restart {
		sess, err = restartREPLSession(sessionID, lang, cwd)
	} else {
		sess, err = getOrStartREPLSession(sessionID, lang, cwd)
	}
	if err != nil {
		return ErrorOutput(fmt.Sprintf("failed to start %s REPL session: %s", lang, err)), nil
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	result, err := sess.Run(runCtx, code)
	if err != nil {
		// The session died mid-run (broken pipe, interpreter crashed). Drop
		// it so the next call starts clean instead of reusing a corpse.
		dropREPLSession(sessionID, lang)
		return ErrorOutput(fmt.Sprintf("%s REPL session ended unexpectedly: %s", lang, err)), nil
	}
	if result.TimedOut {
		dropREPLSession(sessionID, lang)
		return ErrorOutput(fmt.Sprintf(
			"%s REPL call timed out after %d seconds; the session was killed and its state is lost. The next call will start a fresh session.",
			lang, timeoutMs/1000,
		)), nil
	}

	output := truncateBashOutput(stripEmptyLines(result.Output))
	if result.ExitCode != 0 {
		if output != "" {
			output += "\n"
		}
		output += fmt.Sprintf("Exit code %d", result.ExitCode)
		return ErrorOutput(output), nil
	}
	return SuccessOutput(output), nil
}

// isEnvDefinedFalsy reports whether v is a defined, explicitly-falsy value
// (0/false/no), mirroring the TS isEnvDefinedFalsy helper referenced by
// tools/REPLTool/constants.ts:24 — unset/empty is not falsy, it's absent.
func isEnvDefinedFalsy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "0", "false", "no":
		return true
	default:
		return false
	}
}
