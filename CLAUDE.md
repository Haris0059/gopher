# CLAUDE.md

This file provides guidance to Gopher, an open-source alternative to Claude Code, when working with this repository.

## Project

Gopher (module `github.com/Haris0059/gopher`) is a from-scratch Go reimplementation of Anthropic's
Claude Code CLI, targeting behavioral parity with Claude Code v2.1.88. It is an independent,
unaffiliated research project — not a wrapper or transpilation. Single Go module, no monorepo.
Forked from `ProjectBarks/gopher-code` and rebranded; the old `gopher-code` name and
`cmd/gopher-code/` path still appear in some leftovers (e.g. `scripts/capture-tui/scenarios/*.json`
— tracked as `CLEAN-01` in `TASKS.md`).

See `progress.md` for a feature-by-feature parity matrix against the reference TS implementation
(`claude-code-main/`), and `TASKS.md` for the backlog it feeds.

## Layout

- `cmd/gopher/` — CLI entry point (`main.go`, `print.go`, `exit.go`), plus a `handlers/` subpackage and
  many top-level `*_integration_test.go` files (bridge, MCP, IDE, permissions, OAuth, compact, plugin, etc.)
- `internal/cli/` — Bubble Tea TUI (repl, render, diff, spinner, markdown, permission dialogs, statusbar)
- `internal/testharness/` — golden-file/differential test framework
- `pkg/` — ~40 packages (agents, auth, bridge, commands, compact, config, mcp, message, permissions,
  prompt, provider, query, session, tools, ui, etc.)
- `testdata/` — parity fixtures (JSON): tool schemas, system prompts, API contracts

## Build & test

```
go build -o gopher ./cmd/gopher
go test ./...
go test -race ./...
go test ./... -update      # regenerate golden files
./scripts/coverage-report.sh
```

`scripts/validate-ts-binary.sh` diffs against a sibling checkout of the original TS Claude Code
(`../research/claude-code-source-build/dist/cli.js`); it won't run unless that sibling directory exists.

`deps.go` (`//go:build deps`) blank-imports declared dependencies so `go mod tidy` doesn't prune them
before real usage exists — don't "clean it up" by removing entries.

## Code style

- No lint config exists yet and the tree is not currently gofmt-clean overall. Still, run `gofmt -w` and
  `go vet` on any file you edit before finishing — don't assume the rest of the repo passes.
- Comment density is moderate-to-high; ported tool/provider code often carries `// Source: <ts-file>:<line>`
  comments tracing back to the original TypeScript — informative when present, not required on new code.

## Commits

Recent history used `T<number>: implement <feature>` tied to an internal task tracker; that convention is
being retired going forward. Use plain, descriptive imperative commit messages instead (no ticket prefix).

## Env vars

The provider layer (`pkg/provider/`) mirrors real Claude Code's environment surface, including
`ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`, `ANTHROPIC_DEFAULT_SONNET_MODEL`/`_OPUS_MODEL`/`_HAIKU_MODEL`,
`ANTHROPIC_SMALL_FAST_MODEL`, `CLAUDE_CODE_USE_BEDROCK`/`_VERTEX`/`_FOUNDRY`, `CLAUDE_CODE_MAX_RETRIES`,
`USER_TYPE` (`=ant` unlocks internal/employee behavior), and several beta/effort-level flags.
