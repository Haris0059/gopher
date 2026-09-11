package tools

import (
	"strings"

	"github.com/Haris0059/gopher/pkg/skills"
)

// staticAgentListSection is the fallback sentence used when the agent list
// isn't rendered inline — either because there are no agents to show or
// because CLAUDE_CODE_AGENT_LIST_IN_MESSAGES opts into the (unimplemented in
// Gopher) attachment path.
// Source: AgentTool/prompt.ts:197
const staticAgentListSection = "Available agent types are listed in <system-reminder> messages in the conversation."

// agentToolsDescription describes the tools available to an agent for
// display in the agent list.
// Source: AgentTool/prompt.ts:15-37 getToolsDescription
func agentToolsDescription(a skills.AgentDefinition) string {
	hasAllowlist := len(a.Tools) > 0
	hasDenylist := len(a.DisallowedTools) > 0

	switch {
	case hasAllowlist && hasDenylist:
		deny := make(map[string]bool, len(a.DisallowedTools))
		for _, t := range a.DisallowedTools {
			deny[t] = true
		}
		var effective []string
		for _, t := range a.Tools {
			if !deny[t] {
				effective = append(effective, t)
			}
		}
		if len(effective) == 0 {
			return "None"
		}
		return strings.Join(effective, ", ")
	case hasAllowlist:
		return strings.Join(a.Tools, ", ")
	case hasDenylist:
		return "All tools except " + strings.Join(a.DisallowedTools, ", ")
	default:
		return "All tools"
	}
}

// FormatAgentLine formats one agent line for the agent list section:
// `- type: whenToUse (Tools: ...)`.
// Source: AgentTool/prompt.ts:39-46 formatAgentLine
func FormatAgentLine(a skills.AgentDefinition) string {
	return "- " + a.AgentType + ": " + a.WhenToUse + " (Tools: " + agentToolsDescription(a) + ")"
}

// BuildAgentListSection builds the agentListSection injected into the Agent
// tool's prompt: the real agent list, or the static fallback sentence when
// there's nothing to show or when CLAUDE_CODE_AGENT_LIST_IN_MESSAGES opts
// into the attachment path (which Gopher doesn't implement, so it always
// falls back to the static sentence here).
// Source: AgentTool/prompt.ts:57-63,196-199
func BuildAgentListSection(agents []skills.AgentDefinition) string {
	if shouldInjectAgentListInMessages() || len(agents) == 0 {
		return staticAgentListSection
	}
	lines := make([]string, 0, len(agents))
	for _, a := range agents {
		lines = append(lines, FormatAgentLine(a))
	}
	return "Available agent types and the tools they have access to:\n" + strings.Join(lines, "\n")
}

// shouldInjectAgentListInMessages reports whether the agent list should be
// omitted from the tool description in favor of an (unimplemented) attachment
// path, controlled by CLAUDE_CODE_AGENT_LIST_IN_MESSAGES.
// Source: AgentTool/prompt.ts:57-63
func shouldInjectAgentListInMessages() bool {
	v := strings.ToLower(strings.TrimSpace(getEnvVar("CLAUDE_CODE_AGENT_LIST_IN_MESSAGES")))
	return v == "true" || v == "1"
}
