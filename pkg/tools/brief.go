package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Source: tools/BriefTool/BriefTool.ts, tools/BriefTool/prompt.ts

// BriefToolName and LegacyBriefToolName are the current and legacy names for
// the tool that sends a message to the user.
// Source: BriefTool/prompt.ts:1-2
const (
	BriefToolName       = "SendUserMessage"
	LegacyBriefToolName = "Brief"
)

// BriefToolPrompt is the system-prompt guidance returned by Prompt().
// Source: BriefTool/prompt.ts:6-10
const BriefToolPrompt = `Send a message the user will read. Text outside this tool is visible in the detail view, but most won't open it — the answer lives here.

` + "`message`" + ` supports markdown. ` + "`attachments`" + ` takes file paths (absolute or cwd-relative) for images, diffs, logs.

` + "`status`" + ` labels intent: 'normal' when replying to what they just asked; 'proactive' when you're initiating — a scheduled task finished, a blocker surfaced during background work, you need input on something they haven't asked about. Set it honestly; downstream routing uses it.`

// BriefTool sends a message (with optional attachments) to the user. It is
// the model's primary visible output channel when brief-only mode is active.
// Source: tools/BriefTool/BriefTool.ts
type BriefTool struct{}

func (t *BriefTool) Name() string        { return BriefToolName }
func (t *BriefTool) Description() string { return "Send a message to the user" }

// IsReadOnly — Source: BriefTool.ts:117
func (t *BriefTool) IsReadOnly() bool { return true }

// IsConcurrencySafe — Source: BriefTool.ts:114-116
func (t *BriefTool) IsConcurrencySafe(json.RawMessage) bool { return true }

// Aliases — Source: BriefTool/prompt.ts:2 (LEGACY_BRIEF_TOOL_NAME)
func (t *BriefTool) Aliases() []string { return []string{LegacyBriefToolName} }

// SearchHint — Source: BriefTool.ts:69-70
func (t *BriefTool) SearchHint() string {
	return "send a message to the user — your primary visible output channel"
}

// MaxResultSizeChars — Source: BriefTool.ts:141 (maxResultSizeChars: 100_000)
func (t *BriefTool) MaxResultSizeChars() int { return 100_000 }

// IsEnabled — Source: BriefTool.ts:125-127 (isEnabled: () => isBriefEnabled())
func (t *BriefTool) IsEnabled() bool { return BriefEnabled() }

// Prompt — Source: BriefTool.ts:150-152
func (t *BriefTool) Prompt() string { return BriefToolPrompt }

// InputSchema — Source: BriefTool.ts:22-38
func (t *BriefTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"message": {"type": "string", "description": "The message for the user. Supports markdown formatting."},
			"attachments": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Optional file paths (absolute or relative to cwd) to attach. Use for photos, screenshots, diffs, logs, or any file the user should see alongside your message."
			},
			"status": {
				"type": "string",
				"enum": ["normal", "proactive"],
				"description": "Use 'proactive' when you're surfacing something the user hasn't asked for and needs to see now — task completion while they're away, a blocker you hit, an unsolicited status update. Use 'normal' when replying to something the user just said."
			}
		},
		"required": ["message", "status"],
		"additionalProperties": false
	}`)
}

// BriefDisplay is the structured payload attached to the ToolOutput for rich
// TUI rendering. Source: BriefTool.ts output schema (message/attachments/sentAt)
// plus the 'status' input field, carried through for a future brief-only label.
type BriefDisplay struct {
	Message     string
	Status      string
	SentAt      string
	Attachments []ResolvedAttachment
}

// Execute — Source: BriefTool.ts:154-166 (call), :168-177 (mapToolResultToToolResultBlockParam)
func (t *BriefTool) Execute(_ context.Context, tc *ToolContext, input json.RawMessage) (*ToolOutput, error) {
	var params struct {
		Message     string   `json:"message"`
		Attachments []string `json:"attachments"`
		Status      string   `json:"status"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ErrorOutput("invalid input: " + err.Error()), nil
	}

	sentAt := time.Now().UTC().Format(time.RFC3339)

	var resolved []ResolvedAttachment
	if len(params.Attachments) > 0 {
		cwd := ""
		if tc != nil {
			cwd = tc.CWD
		}
		if err := validateAttachmentPaths(cwd, params.Attachments); err != nil {
			return ErrorOutput(err.Error()), nil
		}
		r, err := resolveAttachments(cwd, params.Attachments)
		if err != nil {
			return ErrorOutput(err.Error()), nil
		}
		resolved = r
	}

	content := "Message delivered to user."
	if n := len(resolved); n > 0 {
		if n == 1 {
			content += " (1 attachment included)"
		} else {
			content += fmt.Sprintf(" (%d attachments included)", n)
		}
	}

	out := SuccessOutput(content)
	out.Display = BriefDisplay{
		Message:     params.Message,
		Status:      params.Status,
		SentAt:      sentAt,
		Attachments: resolved,
	}
	return out, nil
}
