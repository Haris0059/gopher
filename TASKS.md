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
| ~~TEST-02~~ | ~~Resolve `pkg/agents` (236 LOC, 0 tests) vs `pkg/skills/agents.go` — two independent agent-loading paths (`agents.Agent` vs `skills.AgentDefinition`); determine which is live, delete or merge the other~~ Done — `pkg/skills` kept (it had zero consumers before this), `pkg/agents` deleted, `gopher agents` repointed | 2 | — |
| ~~TEST-03~~ | ~~Add tests for `pkg/stats` (`store.go`, 142 LOC, 0 tests)~~ Done | 1 | — |
| ~~TEST-04~~ | ~~Add tests for `pkg/async` (63 LOC, 0 tests)~~ Done | 0.5 | — |
| ~~TEST-05~~ | ~~Add tests for `pkg/installer` (45 LOC, 0 tests)~~ Done — full suite alongside `INST-01` | 0.5 | ~~INST-01~~ |
| ~~TEST-06~~ | ~~Add tests for `pkg/commands/install_github_app` (53 LOC, 0 tests)~~ Done — also ported the two constants missing from the reference (`PRBody`, `CodeReviewPluginWorkflowContent`) and restored `WorkflowContent`'s dropped inline comments; see `TEST-11` for the zero-consumer finding this turned up | 0.5 | — |
| ~~TEST-07~~ | ~~Add tests for `pkg/ui/components/shell` (55 LOC, 0 tests) and `pkg/ui/components/wizard` (108 LOC, 0 tests)~~ Done — both packages' `// Source:` comments pointed at reference files they didn't actually mirror, same invented-API pattern as `TOOL-01`/`TOOL-02`. Ported the cheap, unambiguous reference semantics before testing: `shell` gained `tryFormatJson`/`tryJsonFormatContent`/`linkifyUrlsInText`/`stripUnderlineAnsi`, `OutputLine`/`ShellProgressMessage`/`ShellTimeDisplay` were rewritten to the reference's last-5-lines/status-row/timeout shape, and `TruncateOutput` had three bugs fixed (odd-`maxLines` undercounted omissions by one, `maxLines=1` discarded all content, negative `maxLines` panicked). `wizard` gained the reference's navigation-history stack (`GoTo`/`Prev` push/pop instead of plain arithmetic), `Next()`/`Prev()` completing/cancelling at the ends, bulk `Update()`, and `Title`/`ShowStepCounter`. See `TEST-12`/`TEST-13` for the zero-consumer finding this turned up | 1 | — |
| TEST-08 | Thicken `pkg/services` tests (451 src / 109 test LOC — thinnest ratio outside known stubs) | 2 | — |
| TEST-09 | `pkg/stats` parity + wiring: `GetAll()` diverges from `createStatsStore()` (`src/context/stats.tsx`) — Go emits an extra `<name>_sum`, emits sets as `<name>_unique` instead of plain `<name>`, and skips percentile keys on an empty reservoir. Also: the package has zero consumers (the TS `StatsProvider` flushes `getAll()` into `lastSessionMetrics` on process exit; Gopher has no equivalent). Decide keep-or-align, then wire it or delete it. | 1 | — |
| TEST-10 | `pkg/async` keep-or-delete: the package has zero importers anywhere in the repo (not in `deps.go` either). Decide whether to wire `Debouncer`/`Throttler` into the call sites that want them (`pkg/ui`, `internal/cli`) or delete the package. | 0.5 | — |
| TEST-11 | `pkg/commands/install_github_app` keep-or-wire: the package (constants only) has zero importers anywhere in the repo — `/install-github-app` (`pkg/ui/commands/handlers.go:3117`, registered at `:3673`) is a stub `Handler` that returns a static `InstallGitHubAppMsg` without importing it. Either wire the stub to surface these constants or accept it stays dormant pending the real wizard (`setupGitHubActions.ts` port, out of scope here). | 0.5 | ~~TEST-06~~ |
| TEST-12 | `pkg/ui/components/shell` keep-or-wire: zero importers anywhere in the repo. The live tool-output truncation paths are `pkg/tools/bash.go:242` `truncateBashOutput` (character-count, wired into `BashTool.Execute` at `:200`) and `message_bubble.go:273` (inline 10-line display truncation) — neither uses this package. There is no `BashDisplay` renderer type at all; bash results flow through the generic `pkg/tools/brief.go` `BriefDisplay` path. Either wire `shell`'s now-reference-aligned `OutputLine`/`ShellProgressMessage` into that display path or accept it stays dormant. | 1 | ~~TEST-07~~ |
| TEST-13 | `pkg/ui/components/wizard` keep-or-wire: zero importers anywhere in the repo. Two existing flows hand-roll their own step machines instead — `onboarding.go:19` (`OnboardingStep`/`OnboardingModel`) and `console_oauth.go:19` (`OAuthState`/`OAuthFlowModel`) — and neither can adopt `Wizard` as-is since both are `tea.Model`-shaped while `Wizard` has no `Update`/`View`. Wiring means refactoring one of them onto it, which is its own task. | 2 | ~~TEST-07~~ |

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

`pkg/installer` now implements real install/update logic (`INST-01`) and both CLI
handlers are wired to it (`INST-02`).

| ID | Task | Est | Depends on |
|---|---|---|---|
| ~~INST-01~~ | ~~Implement real install/update logic in `pkg/installer` (download release, verify checksum, replace binary)~~ Done — ported the reference's directory layout, versioned installs, atomic symlink, and GC (`installer.ts`/`download.ts`/`pidLock.ts`), scoped down: GitHub Releases + `checksums.txt` instead of the internal GCS/manifest host, no npm/Artifactory branch, no musl detection, `pidLock.ts`'s 434 LOC reduced to a plain PID-liveness lockfile. Also fixed `cmd/gopher/handlers/update.go`'s `DefaultFetchLatestVersion`, which pointed at `anthropics/claude-code` despite its own comment saying "the gopher repository" | 3 | — |
| ~~INST-02~~ | ~~Wire `cmd/gopher/handlers/install.go` to call `pkg/installer` instead of its current placeholder~~ Done — `handlers.Install`'s default engine now calls `installer.Install` and prints `installer.CheckInstall()`'s PATH warnings; `handlers.Update`'s install-delegation stub now calls it too via a new `PerformInstall` seam, and `DefaultFetchLatestVersion` delegates to `installer.ResolveVersion` instead of duplicating the GitHub API call. Behavior change: `gopher install <channel>` now rejects anything that isn't `""`/`latest`/`stable`/semver (previously any string, e.g. `beta`, was accepted and echoed back). New `GOPHER_INSTALL_API_BASE_URL`/`GOPHER_INSTALL_DOWNLOAD_BASE_URL` env overrides let tests point at a fake release server | 1 | ~~INST-01~~ |
| ~~INST-03~~ | ~~`cmd/gopher/handlers/doctor.go:22` — launch the real Doctor TUI screen instead of the current stand-in~~ Done — `gopher doctor` now runs `screens.DoctorModel` full-screen on a TTY (wrapped to turn `DoctorDoneMsg` into `tea.Quit`, since a standalone program has no parent app to do that) and falls back to the same sections as plain text via a new `screens.RenderDoctorText` when stdout isn't a terminal. Also backfilled the six `pkg/doctor.CollectOptions` sections that `Collect(CollectOptions{})` previously left `nil` (agents, settings/keybinding/MCP parse warnings, version locks, install-type/package-manager, dist tags) with real collectors in a new `pkg/doctor/collect.go`, plus wired the two orphaned `pkg/doctor/utils.go` helpers (ripgrep, Linux glob warnings) and `installer.CheckInstall()` into the Diagnostics section — so the in-REPL `/doctor` screen (`pkg/ui/app.go`) gained the same real data for free. Required exporting `installer.ListLocks`/`CleanupStaleLocks` (previously test-only unexported helpers). Left out of scope, unimplemented anywhere: context warnings (CLAUDE.md size / agent-description tokens / MCP tool tokens / unreachable rules) and plugin-error reporting — both are net-new sections with no existing Go collector, and plugins has no renderer at all yet | 2 | — |
| INST-04 | `cmd/gopher/handlers/setup_token.go:82` — launch the real `ConsoleOAuthFlow` TUI component | 2 | — |
| INST-05 | `cmd/gopher/main.go:827` — wire real user config from settings instead of the current placeholder | 1 | — |
| INST-06 | Publish real GitHub releases (CI workflow + goreleaser config, none exist yet) with `gopher_<version>_<goos>_<goarch>[.exe]` assets and a `checksums.txt`, matching the naming `pkg/installer` already expects — `INST-01`'s httptest coverage is not a substitute for one real end-to-end install | 2 | ~~INST-01~~ |
| INST-07 | `pkg/doctor` context-warnings collector: CLAUDE.md size, agent-description token totals, MCP tool token totals, unreachable permission rules — `uidoctor.ContextWarnings`/`RenderContextWarnings` exist but nothing populates them (see `INST-03`) | 3 | — |
| INST-08 | Plugin errors in `/doctor`: no `pkg/ui/doctor` type or renderer exists for plugin load failures at all (see `INST-03`) | 1 | ~~PLUG-01~~ |

## TOOL — Tool completeness

| ID | Task | Est | Depends on |
|---|---|---|---|
| ~~TOOL-01~~ | ~~`pkg/tools/brief.go` (39 LOC) — replace echo-only `send`/`receive` with real cross-session storage~~ Done — the old wording was based on the stub's own invented docstring; the reference `BriefTool` has no cross-session storage at all. It is `SendUserMessage` (legacy alias `Brief`): the model's user-facing output channel in brief-only mode. Ported the real `{message, attachments?, status}` schema, attachment validate/resolve (`BriefTool/attachments.ts`, minus the unrelated bridge-upload leg), the `Message delivered to user.` result, `IsEnabled()` gating on Kairos/opt-in (`pkg/tools/brief_state.go`), the proactive system-prompt section, alias resolution in `ToolRegistry`, and a `BriefDisplay` TUI renderer | 3 | — |
| ~~TOOL-02~~ | ~~`pkg/tools/repltool.go` (73 LOC) — replace one-shot `<lang> -c <cmd>` with a persistent REPL session (process kept alive, stdin/stdout piped across calls)~~ Done — this task's own premise (port a persistent REPL from the reference) was wrong the same way TOOL-01's was: the TS reference's `REPL` tool is a different feature, an ant-only JS VM sandbox that hides Read/Write/Edit/Glob/Grep/Bash/NotebookEdit/Agent from the model and re-exposes them as in-VM calls (`src/tools/REPLTool/constants.ts`), and its implementation file is missing from the leaked checkout. Built the literal ask instead, as a Gopher-native capability: `pkg/tools/repl_session.go` keeps one python/node/ruby/bash interpreter alive per (session, language), framed via a sentinel-terminated driver protocol (no prompt-scraping), with idle reaping, timeout-kills-and-drops, and `CloseAllREPLSessions` wired into `internal/cli/tui_v2.go`'s shutdown. The reference's actual JS-VM REPL is filed separately as `TOOL-10` | 4 | — |
| TOOL-03 | Gate `TestingPermission` tool registration on test mode (it's currently registered unconditionally in `RegisterDefaults`, unlike the TS reference) | 0.5 | — |
| ~~TOOL-04~~ | ~~`pkg/tools/agent.go:61` — build `agentListSection` from loaded agent definitions instead of the static placeholder string~~ Done — `Prompt()` now renders the real agent list via `BuildAgentListSection`/`skills.LoadAllAgents`, and `Execute` resolves `subagent_type` against agent definitions (system prompt, model, maxTurns, tool scoping) instead of hardcoding a generic sub-agent | 1 | ~~TEST-02~~ unblocked |
| TOOL-08 | `pkg/ui/commands/handlers.go:1129` — the `/agents` slash command hardcodes two fake agents (incl. a `bash` agent that exists in neither loader); repoint at `skills.LoadAgents` so it agrees with `gopher agents` | 1 | — |
| TOOL-05 | `pkg/tools/powershell_prompt.go:122,129` — check `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS` env var | 0.5 | — |
| TOOL-06 | `pkg/cli/structured_io.go:402` — implement `update_environment_variables` (currently a no-op stub) | 1 | — |
| TOOL-07 | `pkg/ui/components/input.go:432` — implement undo (`'u'` key is currently a no-op) | 2 | — |
| TOOL-09 | Fork-subagent routing: omitting `subagent_type` forks the caller (shares conversation context + prompt cache) instead of defaulting to `general-purpose`. Gate on an `isForkSubagentEnabled()`-equivalent; add the fork branch in `pkg/tools/agent.go` `Execute` (recursion guard — a fork must reject nested forks), the "When to fork" prompt section and fork-aware examples (`prompt.ts:114-196`), and reuse `SendMessage`-to-continue for fork results. Reference: `AgentTool.tsx:319-329`, `forkSubagent.ts`, `prompt.ts` | 4 | — |
| TOOL-10 | Port the TS reference's actual `REPL` tool: an ant-only JS VM sandbox (`isReplModeEnabled()`, `src/tools/REPLTool/constants.ts`) that hides `Read`/`Write`/`Edit`/`Glob`/`Grep`/`Bash`/`NotebookEdit`/`Agent` from the model (`REPL_ONLY_TOOLS`) and re-exposes them as in-VM function calls, emitting "virtual" tool messages (`collapseReadSearch.ts:152`) that get stripped back out of external transcripts (`sessionStorage.ts:4374`). Needs an embedded JS engine (e.g. `goja` — new dependency), a VM-hosted wrapper per primitive tool, the tool-list filtering, and virtual-message plumbing through `pkg/query`/`pkg/ui`. Partially unrecoverable from the leak: `REPLTool.js`/`toolWrappers.ts` (the actual VM setup and tool-wrapper bridging) are missing from the checkout — only `constants.ts` (47 LOC) and `primitiveTools.ts` (40 LOC) survive, so the VM/wrapper wiring is new design, not a port. Not the same feature as Gopher's `REPL` tool (`pkg/tools/repltool.go`, `TOOL-02`), which is a persistent interpreter session and stays as-is | 6 | — |

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
| ~~CLEAN-01~~ | ~~Fix stale `cmd/gopher-code/main.go` references in 4 scenario fixtures~~ Fixed — all 4 fixtures (`area-04-tools/07-tool-edit-file.json`, `.../32-tool-file-diff-preview.json`, `area-05-permissions/05-perm-file-edit.json`, `.../24-perm-diff-in-edit.json`) now point at `cmd/gopher/main.go` | 0.5 | — |
| CLEAN-02 | Fix self-referential comments left over from the sed-replace rebrand: `pkg/ui/components/statusline.go:103`, `pkg/ui/components/utils.go:8`, `utils.go:21` (all now read "Gopher matches Gopher" where they meant "matches Claude Code") | 0.25 | — |
| CLEAN-03 | `pkg/bridge/init_repl.go:263` — v1/v2 branch selection, version gates, session title (leftover TODO, predates rebrand but adjacent) | 2 | — |

---

## Suggested starting point

~~**TEST-01**~~, ~~**TEST-02**~~, ~~**TEST-03**~~, ~~**TEST-04**~~, ~~**TEST-05**~~, ~~**TEST-06**~~, ~~**TEST-07**~~, ~~**INST-01**~~, ~~**INST-02**~~, ~~**INST-03**~~, ~~**TOOL-02**~~, ~~**CLEAN-01**~~ — done.
Next smallest: **CLEAN-02** or **TOOL-03**.
