# progress.md

Feature-by-feature parity matrix between Gopher and the reference Claude Code implementation.

**Snapshot date:** 2026-09-11 · **Gopher:** 805 commits (2026-04-02 → 2026-09-11) ·
**Reference:** `claude-code-main/` — a vendored, leaked source checkout, `package.json` version
`0.0.0-leaked`, dated 2026-03-31, no official version number recoverable (newest model strings
seen: `claude-opus-4-6` / `claude-sonnet-4-6`).

This file replaces an earlier TUI-parity work log that was last updated 2026-04-05 and had gone
stale relative to the rebrand and ~400 subsequent commits. That log's content is summarized in
the [Appendix](#appendix-history) rather than carried forward.

See `TASKS.md` for the backlog this matrix feeds, and `CLAUDE.md` for build/test commands.

---

## At a glance

- ~107,000 LOC production Go, ~125,000 LOC test Go, 4,994 `func Test*` functions, 126 packages.
- `go build ./...` — clean.
- `go vet ./...` — clean. The one prior failure (`internal/cli/startup_bench_test.go:23` calling
  `RunTUIV2` with a stale signature, `TASKS.md` → `TEST-01`) is fixed.
- 41 tools registered by default (`pkg/tools/defaults.go:RegisterDefaults`, plus the separately
  wired `Agent` tool). Down from an earlier README claim of 33, up from the TS reference's ~40
  in `src/tools/` (some of which are `USER_TYPE=ant`/feature-flag gated and never ship).
- ~80 slash commands implemented in `pkg/ui/commands/handlers.go` (4,222 LOC), against ~102
  command source directories in the TS reference (some are internal/debug-only or plugin-migrated).

---

## Parity matrix

Status: **Done** (implemented, tested, no known gaps) · **Partial** (works, but narrower than
reference) · **Stub** (present but non-functional or placeholder) · **Missing** (no code at all).

| Area | Go package | TS reference | Status | Notes |
|---|---|---|---|---|
| Tools | `pkg/tools` (50 files, 11,941 LOC) | `src/tools/` (43 dirs) | **Done** | 41 tools registered; 1 near-stub inside this set (`REPL` — see below). `Brief`/`SendUserMessage` is a real port as of `TOOL-01` |
| Slash commands | `pkg/ui/commands/handlers.go` | `src/commands/` (102 entries) | **Done** | ~80 commands; gap is mostly internal/debug/plugin-migrated commands in the reference |
| Query loop | `pkg/query` (11 files, 1,904 LOC) | `src/QueryEngine.ts`, `src/query.ts` | **Done** | Has its own `parity_test.go` / `parity_gaps_test.go` against `testdata/parity_rules.json` |
| Provider: Anthropic | `pkg/provider/anthropic.go` | `src/services/api/claude.ts` | **Done** | Real SSE streaming, retries, betas, cost tracking |
| Provider: OpenAI-compatible | `pkg/provider/openai.go` | — | **Done** | No direct TS equivalent; Gopher-only addition for Ollama/vLLM/LM Studio |
| Provider: Bedrock | `pkg/provider/bedrock.go` (39 LOC) | `@aws-sdk/client-bedrock-runtime` | **Stub** | Always returns an error; no SigV4 signing. `pkg/auth/aws` (164 LOC of credential resolution) exists but is unused. See `TASKS.md` → `PROV-01/02` |
| Provider: Vertex | `pkg/provider/vertex.go` (40 LOC) | `google-auth-library` | **Stub** | Always returns an error. See `TASKS.md` → `PROV-03/04` |
| Provider: Foundry | — | (gated in TS reference) | **Missing** | Exists only as an enum value in `model.go`/`effort.go` for capability gating. See `TASKS.md` → `PROV-05` |
| Permissions | `pkg/permissions` (17 files, 2,260 LOC) | `src/utils/permissions/` (23 files) | **Partial** | 7 modes implemented (`default`, `acceptEdits`, `bypassPermissions`, `dontAsk`, `plan`, `auto`, `deny`); waterfall rule resolution, dangerous-command detection. Gap: `auto` mode's classifier is unimplemented and falls back to `ask` (`rules.go:58`) |
| Hooks | `pkg/hooks` (1,531 LOC) | `src/utils/hooks/` | **Exceeds reference** | 27 events implemented vs. 12 in the TS reference (`PreToolUse`, `PostToolUse`, `Notification`, `UserPromptSubmit`, `Stop`, `SubagentStart`, `SubagentStop`, `PreCompact`, `PostCompact`, `SessionStart`, `SessionEnd`, `Setup`) |
| Sessions & resume | `pkg/session` (22 files, 3,796 LOC) | `src/utils/sessionStorage.ts` (5,106 LOC) | **Done** | Storage, transcript, history, listing, mailbox, teams/teammates + memory sync, worktree, redaction, path sanitization, plan approval, turn budget |
| Compact & context | `pkg/compact` (15 files, 1,392 LOC) + `pkg/context` (535 LOC) | `src/services/compact/` (11 files) | **Done** | Auto-compact, micro-compact, token/budget, grouping, images, post-cleanup |
| Memory | `pkg/memory` (179 LOC), `pkg/memdir` (490 LOC) | `src/memdir/`, `src/utils/memory/` | **Done** | Thin but functionally complete; CLAUDE.md handling, relevance extraction, team memory sync |
| MCP: client core | `pkg/mcp` (2,012 LOC) | `src/services/mcp/` (22 files) | **Partial** | `initialize`, `notifications/initialized`, `tools/list`, `tools/call`, `resources/list`, `resources/read` implemented |
| MCP: transports | `pkg/mcp/client.go` | stdio, SSE, HTTP, WS | **Partial** | **stdio only** — zero `net/http` imports in `pkg/mcp`. `TransportSSE`/`TransportHTTP`/`TransportWS` are recognized config values with no implementation. See `TASKS.md` → `MCP-01..03` |
| MCP: OAuth | — | `src/services/mcp/auth.ts` (2,466 LOC) | **Missing** | Config fields only, no PKCE/token exchange. See `TASKS.md` → `MCP-04` |
| MCP: prompts | — | `prompts/list`, `prompts/get` | **Missing** | Not implemented at all, also no sampling/elicitation/subscriptions. See `TASKS.md` → `MCP-05..08` |
| Skills | `pkg/skills` (1,256 LOC) | `src/skills/` + `src/utils/skills/` | **Done** | Frontmatter parsing, built-in agents, overrides, agent memory dirs |
| Agents & subagents | `pkg/skills/agents.go` (~1,000 LOC, tested) | `src/tools/AgentTool/loadAgentsDir.ts`, `agentDisplay.ts` | **Done** | `pkg/agents` (the weaker of two duplicate loaders) deleted; `gopher agents` now runs on `pkg/skills`, gaining frontmatter parsing and built-in agents. `TEST-02` done |
| Teams / multi-agent | `pkg/session` (team/teammate), `pkg/tools/teamtools.go` | `src/utils/swarm/`, `src/coordinator/` | **Done** | `TeamCreate`/`TeamDelete` tools, coordinator mode, teammate memory sync |
| Plugins | `pkg/plugins` (341 LOC) + `bundled` | `src/plugins/`, `src/utils/plugins/` (43 files) | **Stub** | `builtin.go`/`types.go` are real; `operations.go` — install/uninstall/list are all TODO no-ops. See `TASKS.md` → `PLUG-01..05` |
| Output styles | `pkg/output_styles` (376 LOC) | `src/outputStyles/` | **Done** | Built-ins + markdown dir loader + frontmatter |
| Keybindings | `pkg/keybindings` (12 files, 1,834 LOC) | `src/keybindings/` (14 files, 3,173 LOC) | **Done** | Parser, matcher, resolver, chords, reserved keys, schema validation |
| Vim mode | `pkg/vim` (5 files, 1,747 LOC) | `src/vim/` (1,518 LOC) | **Done** | Motions, operators, text objects, transitions |
| TUI | `pkg/ui` (~34,000 LOC, 36 component pkgs, 17 hook pkgs) | `src/ink/` + `src/native-ts/yoga-layout/` (~24,000 LOC) + `src/components/` (81,963 LOC) | **Done, different architecture by design** | Bubble Tea/Lip Gloss instead of the vendored Ink fork + pure-TS Yoga flexbox reimplementation — that ~24k LOC of UI plumbing was deliberately not ported, see [Notable divergences](#notable-divergences) |
| Legacy REPL | `internal/cli` (1,445 LOC) | — | **Done, deprecated path** | Line-oriented fallback reachable via `GOPHER_OLD_UI=1`; superseded by `pkg/ui`. Weakest-tested corner of the repo (90 LOC of tests total), but now compiles and vets clean (`TEST-01`, fixed) |
| IDE bridge / remote | `pkg/bridge` (38 files, 13,261 LOC), `pkg/remote` (1,565 LOC), `pkg/server` (856 LOC) | `src/bridge/` (32 files, 12,741 LOC) | **Done** | SSE + WebSocket + hybrid transports, CCR client, JWT, trusted device, work secret, serial batch uploader |
| IDE integration (local attach) | `pkg/ide` (167 LOC) | `src/hooks/useIDEIntegration.tsx` + friends | **Partial** | Process detection (VS Code/Cursor/JetBrains) and lockfile paths only — no RPC/attach protocol. See `TASKS.md` → `IDE-01/02` |
| LSP | `pkg/lsp` (431 LOC) | `src/services/lsp/` | **Partial** | Bare JSON-RPC transport (`NewClient`, `Initialize`, `SendRequest`, `SendNotification`, `Capabilities`); no document-sync or diagnostics helpers. See `TASKS.md` → `LSP-01/02` |
| Auth | `pkg/auth` + `pkg/auth/aws` (1,396 LOC) | `src/services/oauth/`, `src/utils/auth.ts` (2,003 LOC) | **Done** | OAuth + PKCE, FD-based token passing, AWS credential chain (unused by Bedrock today — see `PROV-01`) |
| Telemetry & analytics | `pkg/analytics` (817 LOC), `pkg/telemetry` (250 LOC) | `src/utils/telemetry/`, `src/services/analytics/` | **Done** | Datadog sink, GrowthBook feature flags, killswitch, metadata |
| Stats | `pkg/stats` (142 LOC, tested) | — | **Done** | `TEST-03` done. `GetAll()` diverges from `createStatsStore()` on 3 keys, and the package has zero consumers — see `TASKS.md` → `TEST-09` |
| Tasks & scheduling | `pkg/tools/tasks.go` (868 LOC), `pkg/tools/cron.go` (662 LOC) | `src/tasks/`, `ScheduleCronTool` | **Done** | `TaskCreate/Get/Update/List/Output/Stop`, `CronCreate/Delete/List`, `RemoteTrigger` |
| Installer & updater | `pkg/installer` (11 files, tested) | `src/utils/nativeInstaller/` | **Done** | XDG directory layout, versioned installs with atomic symlink activation, SHA-256-verified downloads with stall/network retry, PID-liveness lockfiles, GC/retention. Scoped down from the reference: GitHub Releases + `checksums.txt` (not the internal GCS host + `manifest.json`), no npm/Artifactory branch, no musl detection. `cmd/gopher/handlers/install.go` and `update.go` are wired to it (`INST-02`), exercised end-to-end against a local fake release server in `cmd/gopher/setup_doctor_install_integration_test.go`. No real releases are published yet — see `TASKS.md` → `INST-06` |
| Voice | `pkg/voice` (106 LOC) | `src/voice/`, `src/services/voice*.ts` | **Stub** | State machine only — no audio capture, no STT. See `TASKS.md` → `VOICE-01/02` |
| Sandbox | — | `@anthropic-ai/sandbox-runtime`, `src/utils/sandbox/` | **Missing** | No Go equivalent found; not yet scoped into `TASKS.md` |
| Shell parsing / bash security | `pkg/tools/bash_security.go` (376 LOC), `bash_validation.go` (221 LOC), `shellparse.go` (270 LOC) | `src/utils/bash/` (~10,000 LOC incl. `bashParser.ts`, `treeSitterAnalysis.ts`) | **Partial** | Functional bash AST parsing and permission classification exists at roughly a tenth of the reference's LOC — narrower coverage, not verified line-for-line |
| Cost tracking | `pkg/provider/cost.go` (170 LOC) | `src/cost-tracker.ts`, `src/costHook.ts` | **Done** | — |

---

## Notable divergences

**Where Gopher exceeds the reference:**
- 27 hook events vs. 12 in the TS source.
- A first-class OpenAI-compatible provider (`pkg/provider/openai.go`) with no equivalent in the
  reference — lets Gopher talk to Ollama, vLLM, LM Studio directly.

**Where Gopher deliberately differs, not a gap:**
- The TUI is built on Bubble Tea/Lip Gloss (`pkg/ui`), not a port of the vendored Ink fork
  (`src/ink/`, ~19,996 LOC) or the pure-TypeScript Yoga flexbox reimplementation
  (`src/native-ts/yoga-layout/`, 2,579 LOC). That ~24,000 LOC of UI plumbing was replaced
  wholesale by design, not left undone.
- Golden-file/transcript-diffing infrastructure (`internal/testharness`) exists, but the captured
  transcript corpus it once compared against (`data/claude/`) is gitignored and not checked in —
  current coverage comes from `pkg/ui/visual_parity_test.go` (66 hand-written behavioral
  assertions) and 19 JSON fixtures under `testdata/`, not transcript replay.

---

## Known-missing summary

Condensed list of every Stub/Missing item above, cross-referenced to `TASKS.md`:

| # | Item | TASKS.md |
|---|---|---|
| 1 | Bedrock provider always errors | `PROV-01`, `PROV-02` |
| 2 | Vertex provider always errors | `PROV-03`, `PROV-04` |
| 3 | Foundry provider doesn't exist | `PROV-05` |
| 4 | MCP client is stdio-only | `MCP-01`, `MCP-02`, `MCP-03` |
| 5 | No MCP OAuth | `MCP-04`, `MCP-09` |
| 6 | No MCP prompts/sampling/elicitation/subscriptions | `MCP-05`–`MCP-08` |
| 7 | Plugin install/uninstall/list are no-ops | `PLUG-01`–`PLUG-05` |
| ~~8~~ | ~~No real installer/updater~~ Fixed — `pkg/installer` implements install/update/GC against GitHub Releases, and both `install`/`update` CLI handlers are wired to it, tested against `httptest` (no real releases published yet — `INST-06`) | `INST-01`, `INST-02` |
| 9 | Doctor and setup-token TUI screens stubbed | `INST-03`, `INST-04` |
| 10 | Voice is a state machine with no audio/STT | `VOICE-01`, `VOICE-02` |
| ~~11~~ | ~~`Brief` tool is echo-only~~ Fixed — ported as `SendUserMessage` (legacy alias `Brief`, resolved by `ToolRegistry.Get`): real `{message, attachments?, status}` schema, attachment validation/resolution, `Message delivered to user.` model-facing result, `IsEnabled()` gated on Kairos/opt-in, proactive system-prompt section, and a `BriefDisplay` renderer in `message_bubble.go`. Bridge attachment upload (`upload.ts`) remains unported | `TOOL-01` |
| 12 | `REPL` tool is one-shot, not persistent | `TOOL-02` |
| 13 | `TestingPermission` tool registered unconditionally | `TOOL-03` |
| ~~14~~ | ~~Agent tool's agent-list section is a static placeholder~~ Fixed — `Prompt()` renders the loaded agent list, `subagent_type` resolves a real agent definition, and `Execute` now enforces `Agent(<type>)` deny rules and `requiredMcpServers` gating (fork-subagent routing remains unported — no such feature exists in Gopher) | `TOOL-04` |
| 15 | `pkg/ide` has no RPC/attach protocol | `IDE-01`, `IDE-02` |
| 16 | `pkg/lsp` has no diagnostics/document-sync | `LSP-01`, `LSP-02` |
| 17 | `auto` permission mode classifier unimplemented | `PERM-01` |
| ~~18~~ | ~~Untested packages (`pkg/async`, ...)~~ `pkg/async` and `pkg/installer` fixed. Remaining: `pkg/commands/install_github_app`, `pkg/ui/components/{shell,wizard}` | `TEST-05`–`TEST-07` |
| ~~19~~ | ~~`pkg/agents` duplicates `pkg/skills/agents.go`~~ Fixed — `pkg/agents` deleted | `TEST-02` |
| 20 | Stale rebrand references (4 scenario fixtures, self-referential comments) | `CLEAN-01`, `CLEAN-02` |
| ~~21~~ | ~~`internal/cli/startup_bench_test.go` fails to compile~~ Fixed | `TEST-01` |
| 22 | Sandbox has no Go equivalent | not yet in `TASKS.md` |
| 23 | `pkg/async` has zero importers anywhere in the repo (not in `deps.go` either) | `TEST-10` |

---

## Appendix: history

**Commit conventions.** 805 commits span 2026-04-02 to 2026-09-11. 422 of them (52%) used a
`T<number>: implement <feature>` prefix tied to an internal task tracker (highest observed:
T590); that convention was retired as of the September rebrand. Other prefixes that coexisted
at various points: `phase-b` (66 commits), `parity-audit` (38), `feat` (14), `Phase 1C` (7),
`fix` (7), `Phase 2C` (6), `Phase 6` (5), `chore` (5), plus one-off `Phase 1A/1B/2A/2B/3A/4A/5A`
prefixes (4 each). Going forward, commits use plain imperative messages with no prefix
(see `CLAUDE.md`).

**The previous progress.md.** From 2026-04-02 to 2026-04-05 this file tracked a TUI visual-parity
effort: capturing 375 terminal-UI scenarios from the original Claude Code (`scripts/capture-tui/`,
still present and valid), then writing Go tests to compare Gopher's rendered output against
captured snapshots. That snapshot corpus (`data/claude/`) was never checked in and is gitignored,
so the comparison strategy became unreproducible from a clean checkout. The log itself froze
mid-effort (last state: "Phase 2 IN PROGRESS", "Next B67", 66 of a nominal 126 planned parity
tests written) and was never updated again — five months and roughly 400 commits, including the
September rebrand, passed with no record here. Several specific things it recorded as fixed were
later reverted by the rebrand (e.g. the welcome-screen box border and two-column layout it
describes fixing no longer exist; `pkg/ui/components/welcome.go` was rewritten to a simpler
134-line gopher splash in commit `1990166`). None of that phase-by-phase detail is preserved
above; the 375-scenario capture corpus under `scripts/capture-tui/scenarios/` remains a valid,
reusable asset for future parity work.
