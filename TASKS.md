# TASKS.md

Backlog for closing the gap between Gopher and the reference Claude Code implementation
(`claude-code-main/`, snapshot `0.0.0-leaked`, 2026-03-31). See `progress.md` for the full
parity matrix these tasks are derived from.

Format: each epic is a checklist table — `ID | Task | Est | Depends on`. Estimates are
person-days for one contributor already familiar with the relevant Go package; treat them as
ballpark, not commitments. IDs are stable — once assigned, don't renumber, even if a task is
dropped (leave it struck through instead).

Epics are ordered roughly by unblocking value, not difficulty.

---

## TEST — Test & hygiene debt

Small, low-risk, and blocking: `TEST-01` was the only thing failing `go vet ./...` in the whole
repo; it's now fixed.

| ID | Task | Est | Depends on |
|---|---|---|---|
| ~~TEST-01~~ | ~~Fix `internal/cli/startup_bench_test.go:23` — it calls `RunTUIV2` with a stale 5-arg signature; current signature is `(ctx, *session.SessionState, provider.ModelProvider, *tools.ToolRegistry)` at `internal/cli/tui_v2.go:18`~~ Done | 0.25 | — |
| TEST-02 | Resolve `pkg/agents` (236 LOC, 0 tests) vs `pkg/skills/agents.go` — two independent agent-loading paths (`agents.Agent` vs `skills.AgentDefinition`); determine which is live, delete or merge the other | 2 | — |
| TEST-03 | Add tests for `pkg/stats` (`store.go`, 142 LOC, 0 tests) | 1 | — |
| TEST-04 | Add tests for `pkg/async` (63 LOC, 0 tests) | 0.5 | — |
| TEST-05 | Add tests for `pkg/installer` (45 LOC, 0 tests) | 0.5 | INST-01 |
| TEST-06 | Add tests for `pkg/commands/install_github_app` (53 LOC, 0 tests) | 0.5 | — |
| TEST-07 | Add tests for `pkg/ui/components/shell` (55 LOC, 0 tests) and `pkg/ui/components/wizard` (108 LOC, 0 tests) | 1 | — |
| TEST-08 | Thicken `pkg/services` tests (451 src / 109 test LOC — thinnest ratio outside known stubs) | 2 | — |

## MCP — Remote MCP transports & auth

`pkg/mcp/client.go` only speaks stdio; `TransportSSE`/`TransportHTTP`/`TransportWS` exist as
config enum values in `pkg/mcp/config.go` but nothing implements them. Reference:
`src/services/mcp/` (22 files, incl. `auth.ts` at 2,466 LOC).

| ID | Task | Est | Depends on |
|---|---|---|---|
| MCP-01 | Implement Streamable-HTTP transport (`tools/list`, `tools/call`, `resources/*` over HTTP POST/SSE per MCP spec) | 5 | — |
| MCP-02 | Implement SSE transport (legacy MCP SSE, separate from Streamable-HTTP) | 3 | MCP-01 |
| MCP-03 | Implement WebSocket transport | 3 | MCP-01 |
| MCP-04 | MCP OAuth: PKCE flow, token exchange, refresh, per-server credential storage | 8 | MCP-01 |
| MCP-05 | `prompts/list` and `prompts/get` support | 2 | MCP-01 |
| MCP-06 | Sampling support (server-initiated completion requests back through the host) | 3 | MCP-01 |
| MCP-07 | Elicitation over MCP (server-initiated structured user prompts) | 3 | MCP-01 |
| MCP-08 | `notifications/*` subscriptions (resource/tool list changes) | 2 | MCP-01 |
| MCP-09 | Wire `pkg/mcp/mcpauth.go` (144 LOC, currently not default-registered) into the auth flow from MCP-04 | 1 | MCP-04 |

## PROV — Cloud provider backends

`pkg/provider/bedrock.go` (39 LOC) and `pkg/provider/vertex.go` (40 LOC) unconditionally
return an error. `pkg/auth/aws` (164 LOC of credential-chain resolution) already exists and is
unused by Bedrock.

| ID | Task | Est | Depends on |
|---|---|---|---|
| PROV-01 | Bedrock: SigV4 request signing + `InvokeModelWithResponseStream`, wire up existing `pkg/auth/aws` credential chain | 4 | — |
| PROV-02 | Bedrock: model ID mapping (Anthropic model string → Bedrock model ID) and region config | 1 | PROV-01 |
| PROV-03 | Vertex: Google Cloud auth (`google-auth-library` equivalent — service account / ADC), request signing | 4 | — |
| PROV-04 | Vertex: streaming response parsing, project/region config | 2 | PROV-03 |
| PROV-05 | Foundry provider: currently an enum only in `model.go`/`effort.go`, no implementation at all — scope and build from scratch | 5 | — |

## PLUG — Plugins & marketplace

`pkg/plugins/operations.go:23,34,45` — `InstallPlugin`, `UninstallPlugin`, `ListInstalledPlugins`
are TODO no-ops. `pkg/plugins/builtin.go` and `types.go` are real.

| ID | Task | Est | Depends on |
|---|---|---|---|
| PLUG-01 | Implement marketplace fetch (reference: `marketplaceManager.ts`, 2,644 LOC — scope down to what Gopher needs) | 5 | — |
| PLUG-02 | Implement `InstallPlugin` — download, verify, extract into plugin dir | 3 | PLUG-01 |
| PLUG-03 | Implement `UninstallPlugin` | 1 | PLUG-02 |
| PLUG-04 | Implement `ListInstalledPlugins` — scan plugin directories and load manifests | 2 | — |
| PLUG-05 | Plugin-contributed commands/agents/hooks/output-styles/MCP servers wiring (verify what's already handled vs. stubbed once PLUG-01..04 land) | 3 | PLUG-01..04 |

## INST — Self-install & update

`pkg/installer` (45 LOC) only has `InstallDir()`, `BinaryName()`, `IsInstalled()`.
`cmd/gopher/handlers/install.go:48` has a TODO in place of calling real install logic.

| ID | Task | Est | Depends on |
|---|---|---|---|
| INST-01 | Implement real install/update logic in `pkg/installer` (download release, verify checksum, replace binary) | 3 | — |
| INST-02 | Wire `cmd/gopher/handlers/install.go` to call `pkg/installer` instead of its current placeholder | 1 | INST-01 |
| INST-03 | `cmd/gopher/handlers/doctor.go:22` — launch the real Doctor TUI screen instead of the current stand-in | 2 | — |
| INST-04 | `cmd/gopher/handlers/setup_token.go:82` — launch the real `ConsoleOAuthFlow` TUI component | 2 | — |
| INST-05 | `cmd/gopher/main.go:827` — wire real user config from settings instead of the current placeholder | 1 | — |

## TOOL — Tool completeness

| ID | Task | Est | Depends on |
|---|---|---|---|
| TOOL-01 | `pkg/tools/brief.go` (39 LOC) — replace echo-only `send`/`receive` with real cross-session storage | 3 | — |
| TOOL-02 | `pkg/tools/repltool.go` (73 LOC) — replace one-shot `<lang> -c <cmd>` with a persistent REPL session (process kept alive, stdin/stdout piped across calls) | 4 | — |
| TOOL-03 | Gate `TestingPermission` tool registration on test mode (it's currently registered unconditionally in `RegisterDefaults`, unlike the TS reference) | 0.5 | — |
| TOOL-04 | `pkg/tools/agent.go:61` — build `agentListSection` from loaded agent definitions instead of the static placeholder string | 1 | TEST-02 |
| TOOL-05 | `pkg/tools/powershell_prompt.go:122,129` — check `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS` env var | 0.5 | — |
| TOOL-06 | `pkg/cli/structured_io.go:402` — implement `update_environment_variables` (currently a no-op stub) | 1 | — |
| TOOL-07 | `pkg/ui/components/input.go:432` — implement undo (`'u'` key is currently a no-op) | 2 | — |

## PERM — Permissions

| ID | Task | Est | Depends on |
|---|---|---|---|
| PERM-01 | `pkg/permissions/rules.go:58` — implement the `auto` mode classifier (currently always falls back to `ask`); reference: `bashClassifier.ts` / `yoloClassifier.ts` | 5 | — |

## IDE — Editor integration

`pkg/ide` (167 LOC) does process detection (VS Code/Cursor/JetBrains via parent process) and
lockfile paths only — no RPC. `pkg/lsp` (431 LOC) has a bare JSON-RPC transport with no
diagnostics or document-sync helpers.

| ID | Task | Est | Depends on |
|---|---|---|---|
| IDE-01 | Design the IDE RPC/attach protocol (what `pkg/bridge` already provides for remote vs. what local IDE attach needs) | 2 | — |
| IDE-02 | Implement local IDE attach beyond detection (diff-in-IDE, selection sync) | 5 | IDE-01 |
| LSP-01 | `pkg/lsp` — add document-sync helpers (`textDocument/didOpen`/`didChange`/`didClose`) | 2 | — |
| LSP-02 | `pkg/lsp` — add diagnostics handling (`textDocument/publishDiagnostics`) and expose it to the `LSP` tool | 3 | LSP-01 |

## VOICE — Voice input

`pkg/voice/state.go` (106 LOC) is a state machine only — no audio capture, no transcription.

| ID | Task | Est | Depends on |
|---|---|---|---|
| VOICE-01 | Audio capture (mic input, cross-platform) | 5 | — |
| VOICE-02 | STT integration behind the existing state machine | 4 | VOICE-01 |

## CLEAN — Rebrand cleanup

Leftovers from the `gopher-code` → `gopher` rebrand (commit `d46cbab` and neighbors).

| ID | Task | Est | Depends on |
|---|---|---|---|
| CLEAN-01 | Fix stale `cmd/gopher-code/main.go` references in 4 scenario fixtures: `scripts/capture-tui/scenarios/area-04-tools/07-tool-edit-file.json`, `.../32-tool-file-diff-preview.json`, `scripts/capture-tui/scenarios/area-05-permissions/05-perm-file-edit.json`, `.../24-perm-diff-in-edit.json` | 0.5 | — |
| CLEAN-02 | Fix self-referential comments left over from the sed-replace rebrand: `pkg/ui/components/statusline.go:103`, `pkg/ui/components/utils.go:8`, `utils.go:21` (all now read "Gopher matches Gopher" where they meant "matches Claude Code") | 0.25 | — |
| CLEAN-03 | `pkg/bridge/init_repl.go:263` — v1/v2 branch selection, version gates, session title (leftover TODO, predates rebrand but adjacent) | 2 | — |

---

## Suggested starting point

~~**TEST-01**~~ — done. Next smallest: **CLEAN-02** or **TOOL-03**.
