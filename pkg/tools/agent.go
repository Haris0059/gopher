package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/Haris0059/gopher/pkg/compact"
	"github.com/Haris0059/gopher/pkg/message"
	"github.com/Haris0059/gopher/pkg/permissions"
	"github.com/Haris0059/gopher/pkg/provider"
	"github.com/Haris0059/gopher/pkg/session"
	"github.com/Haris0059/gopher/pkg/skills"
)

// QueryFunc is the function signature for running a query loop.
// This breaks the import cycle between tools and query: the caller
// injects query.Query at registration time.
type QueryFunc func(
	ctx context.Context,
	sess *session.SessionState,
	prov provider.ModelProvider,
	registry *ToolRegistry,
	orchestrator *ToolOrchestrator,
	onEvent func(text string),
) error

// AgentTool spawns a child query loop to handle a sub-task.
type AgentTool struct {
	provider provider.ModelProvider
	registry *ToolRegistry
	queryFn  QueryFunc

	cwd        string
	loadAgents func(cwd string) []skills.AgentDefinition

	agentsOnce   sync.Once
	agentsCached []skills.AgentDefinition
}

// NewAgentTool creates an AgentTool with the dependencies it needs.
// The queryFn parameter breaks the import cycle between tools and query.
func NewAgentTool(prov provider.ModelProvider, reg *ToolRegistry, queryFn QueryFunc) *AgentTool {
	return &AgentTool{
		provider:   prov,
		registry:   reg,
		queryFn:    queryFn,
		loadAgents: defaultLoadAgents,
	}
}

// defaultLoadAgents loads active agent definitions from the standard
// locations (built-ins plus ~/.claude/agents and <cwd>/.claude/agents).
// Source: loadAgentsDir.ts:270-350, agentToolUtils.ts — activeAgents
func defaultLoadAgents(cwd string) []skills.AgentDefinition {
	return skills.LoadAllAgents(cwd).ActiveAgents
}

// SetAgentLoader overrides how agent definitions are loaded. Exposed for
// tests; production code uses defaultLoadAgents.
func (t *AgentTool) SetAgentLoader(fn func(cwd string) []skills.AgentDefinition) {
	t.loadAgents = fn
	t.agentsOnce = sync.Once{}
}

// SetCWD overrides the working directory used to discover project-level
// agent definitions. Defaults to os.Getwd() when unset.
func (t *AgentTool) SetCWD(dir string) {
	t.cwd = dir
	t.agentsOnce = sync.Once{}
}

// activeAgents returns the cached set of active agent definitions,
// discovering them on first use.
func (t *AgentTool) activeAgents() []skills.AgentDefinition {
	t.agentsOnce.Do(func() {
		cwd := t.cwd
		if cwd == "" {
			cwd, _ = os.Getwd()
		}
		loader := t.loadAgents
		if loader == nil {
			loader = defaultLoadAgents
		}
		t.agentsCached = loader(cwd)
	})
	return t.agentsCached
}

func (t *AgentTool) Name() string { return AgentToolName }
func (t *AgentTool) Description() string {
	return "Launch a new agent"
}
func (t *AgentTool) IsReadOnly() bool { return false }

// Aliases returns the legacy tool name for backward compatibility.
// Source: AgentTool/AgentTool.tsx:228
func (t *AgentTool) Aliases() []string { return []string{LegacyAgentToolName} }

// SearchHint returns the search hint for tool discovery.
// Source: AgentTool/AgentTool.tsx:227
func (t *AgentTool) SearchHint() string { return "delegate work to a subagent" }

// MaxResultSizeChars returns the max result size for agent output.
// Source: AgentTool/AgentTool.tsx:229
func (t *AgentTool) MaxResultSizeChars() int { return 100_000 }

// Prompt returns the system-prompt section for the Agent tool.
// Source: AgentTool/prompt.ts:66-286
func (t *AgentTool) Prompt() string {
	return AgentToolPrompt(BuildAgentListSection(t.activeAgents()))
}

// Source: AgentTool/AgentTool.tsx:82-102 inputSchema
func (t *AgentTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"description": {
				"type": "string",
				"description": "A short (3-5 word) description of the task"
			},
			"prompt": {
				"type": "string",
				"description": "The task for the agent to perform"
			},
			"subagent_type": {
				"type": "string",
				"description": "The type of specialized agent to use for this task"
			},
			"model": {
				"type": "string",
				"description": "Optional model override for this agent. Takes precedence over the agent definition's model frontmatter. If omitted, uses the agent definition's model, or inherits from the parent.",
				"enum": ["sonnet", "opus", "haiku"]
			},
			"run_in_background": {
				"type": "boolean",
				"description": "Set to true to run this agent in the background. You will be notified when it completes."
			},
			"name": {
				"type": "string",
				"description": "Name for the spawned agent. Makes it addressable via SendMessage({to: name}) while running."
			},
			"team_name": {
				"type": "string",
				"description": "Team name for spawning. Uses current team context if omitted."
			},
			"mode": {
				"type": "string",
				"description": "Permission mode for spawned teammate (e.g., \"plan\" to require plan approval).",
				"enum": ["acceptEdits", "auto", "bypassPermissions", "default", "dontAsk", "plan"]
			},
			"isolation": {
				"type": "string",
				"description": "Isolation mode. \"worktree\" creates a temporary git worktree so the agent works on an isolated copy of the repo.",
				"enum": ["worktree"]
			},
			"cwd": {
				"type": "string",
				"description": "Absolute path to run the agent in. Overrides the working directory for all filesystem and shell operations within this agent. Mutually exclusive with isolation: \"worktree\"."
			}
		},
		"required": ["description", "prompt"],
		"additionalProperties": false
	}`)
}

// AgentMaxTurns is the default max turns for subagents.
// Source: AgentTool/runAgent.ts — agents get fewer turns than main loop
const AgentMaxTurns = 30

// agentModelAliases maps short model names to full IDs.
// Source: AgentTool prompt.ts — model enum options
var agentModelAliases = map[string]string{
	"haiku":  "claude-haiku-4-5-20251001",
	"sonnet": "claude-sonnet-4-6",
	"opus":   "claude-opus-4-6",
}

func (t *AgentTool) Execute(ctx context.Context, tc *ToolContext, input json.RawMessage) (*ToolOutput, error) {
	var params AgentToolInput
	if err := json.Unmarshal(input, &params); err != nil {
		return ErrorOutput("invalid input: " + err.Error()), nil
	}
	if err := ValidateAgentToolInput(&params); err != nil {
		return ErrorOutput(err.Error()), nil
	}

	// Resolve subagent_type against the active agent definitions, defaulting
	// to general-purpose when omitted (Gopher has no fork-subagent gate).
	// Source: AgentTool.tsx:319-353
	agentType := params.SubagentType
	if agentType == "" {
		agentType = skills.AgentTypeGeneralPurpose
	}
	agents := t.activeAgents()
	def := skills.FindAgent(agents, agentType)
	if def == nil {
		names := make([]string, 0, len(agents))
		for _, a := range agents {
			names = append(names, a.AgentType)
		}
		return ErrorOutput("Agent type '" + agentType + "' not found. Available agents: " + strings.Join(names, ", ")), nil
	}

	// Reject agent types denied via an Agent(<type>) permission rule.
	// Source: AgentTool.tsx:347-351 — agentExistsButDenied / getDenyRuleForAgent
	if tc.Permissions != nil {
		if _, denied := tc.Permissions.Check(ctx, AgentToolName, agentType).(permissions.DenyDecision); denied {
			return ErrorOutput("Agent type '" + agentType + "' has been denied by permission rule 'Agent(" + agentType + ")'."), nil
		}
	}

	// Reject agent types that require MCP servers not currently connected
	// with tools. Server names are derived from registered mcp__ prefixed
	// tool names, matching how the reference scans serversWithTools.
	// Source: AgentTool.tsx:391-410
	if len(def.RequiredMcpServers) > 0 {
		serversWithTools := mcpServersWithTools(t.registry.All())
		if !skills.HasRequiredMcpServers(*def, serversWithTools) {
			var missing []string
			for _, pattern := range def.RequiredMcpServers {
				found := false
				lowerPattern := strings.ToLower(pattern)
				for _, server := range serversWithTools {
					if strings.Contains(strings.ToLower(server), lowerPattern) {
						found = true
						break
					}
				}
				if !found {
					missing = append(missing, pattern)
				}
			}
			serverList := "none"
			if len(serversWithTools) > 0 {
				serverList = strings.Join(serversWithTools, ", ")
			}
			return ErrorOutput("Agent '" + agentType + "' requires MCP servers matching: " + strings.Join(missing, ", ") +
				". MCP servers with tools: " + serverList +
				". Use /mcp to configure and authenticate the required MCP servers."), nil
		}
	}

	// Resolve model: env override > tool-specified > agent definition > inherit.
	// Source: utils/model/agent.ts — getAgentModel()
	toolSpecifiedModel := resolveModelAlias(params.Model)
	agentDefModel := resolveModelAlias(def.Model)
	model := resolveModelAlias(GetAgentModel(agentDefModel, "", toolSpecifiedModel))
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	// System prompt: the selected agent's, falling back to the generic
	// sub-agent prompt when the definition carries none; append the
	// critical system reminder (verification agents) when present.
	// Source: runAgent.ts:135-160,782
	systemPrompt := def.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful sub-agent. Complete the task and report your findings concisely."
	}
	if def.CriticalSystemReminder != "" {
		systemPrompt += "\n\n" + def.CriticalSystemReminder
	}

	maxTurns := AgentMaxTurns
	if def.MaxTurns > 0 {
		maxTurns = def.MaxTurns
	}

	permissionMode := permissions.PermissionMode(permissions.AutoApprove)
	if def.PermissionMode != "" && ValidPermissionModes[def.PermissionMode] {
		permissionMode = permissions.PermissionMode(def.PermissionMode)
	}

	// Create child session with agent-appropriate config.
	// Source: AgentTool/runAgent.ts:135-160
	childCfg := session.SessionConfig{
		Model:          model,
		SystemPrompt:   systemPrompt,
		MaxTurns:       maxTurns,
		TokenBudget:    compact.DefaultBudget(),
		PermissionMode: permissionMode,
	}
	childSess := session.New(childCfg, tc.CWD)
	childSess.ParentSessionID = tc.SessionID
	childSess.PushMessage(message.UserMessage(params.Prompt))

	// Scope the child registry to the agent definition's allowed tools.
	// ResolveAgentTools already filters ResolvedTools by disallowedTools in
	// both the wildcard and explicit-allowlist cases, so always rebuild from
	// it rather than reusing the parent registry unfiltered.
	// Source: agentToolUtils.ts:122-225
	resolved := ResolveAgentTools(def.Tools, def.DisallowedTools, string(def.Source), string(permissionMode), t.registry.All(), false, false)
	childRegistry := NewRegistry()
	for _, tool := range resolved.ResolvedTools {
		childRegistry.Register(tool)
	}

	// Create a child orchestrator sharing the registry but not state.
	childOrch := NewOrchestrator(childRegistry)

	// Collect text output from the sub-agent.
	var resultText strings.Builder
	onText := func(text string) {
		resultText.WriteString(text)
	}

	// Run the child query loop.
	err := t.queryFn(ctx, childSess, t.provider, childRegistry, childOrch, onText)
	if err != nil {
		return ErrorOutput("agent error: " + err.Error()), nil
	}

	result := resultText.String()
	if result == "" {
		result = "(agent completed with no text output)"
	}
	return SuccessOutput(result), nil
}

// mcpServersWithTools returns the distinct MCP server names that have at
// least one registered tool, parsed from "mcp__<server>__<tool>" names.
// Duplicated from mcp.ParseMCPToolName rather than imported: pkg/mcp already
// imports pkg/tools, so importing pkg/mcp here would create a cycle.
// Source: AgentTool.tsx:391-401 — serversWithTools
func mcpServersWithTools(tools []Tool) []string {
	seen := make(map[string]bool)
	var servers []string
	for _, tool := range tools {
		name := tool.Name()
		if !strings.HasPrefix(name, "mcp__") {
			continue
		}
		rest := name[len("mcp__"):]
		idx := strings.Index(rest, "__")
		if idx < 0 {
			continue
		}
		server := rest[:idx]
		if !seen[server] {
			seen[server] = true
			servers = append(servers, server)
		}
	}
	return servers
}

// resolveModelAlias expands a short model name ("sonnet", "opus", "haiku")
// to its full model ID; other values (including "inherit" and "") pass
// through unchanged.
func resolveModelAlias(model string) string {
	if resolved, ok := agentModelAliases[model]; ok {
		return resolved
	}
	return model
}
