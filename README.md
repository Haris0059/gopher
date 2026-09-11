# Gopher

<p align="center">
  <img src="assets/go-gopher-pixel-art.png" alt="Gopher mascot logo">
</p>

<p align="center">
  <strong>An agentic coding CLI, written in Go. Zero Node.js. Zero Electron. One binary.</strong>
</p>

<p align="center">
  <img src="assets/demo.gif" alt="Gopher demo">
</p>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/badge/Go-1.27+-00ADD8?style=for-the-badge&logo=go&logoColor=white">
    <img src="https://img.shields.io/badge/Go-1.27+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.27+">
  </picture>
  <img src="https://img.shields.io/badge/Tools-41_Built--In-orange?style=for-the-badge" alt="41 Tools">
  <img src="https://img.shields.io/badge/Binary-Single_Static-success?style=for-the-badge" alt="Single Binary">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=for-the-badge" alt="MIT License">
</p>

<p align="center">
  <sub>Starts in milliseconds. No runtime dependencies. Cross-compiles everywhere Go does.</sub>
</p>

---

> [!NOTE]
> **This project is forked from and continues the work of
> [ProjectBarks/gopher-code](https://github.com/ProjectBarks/gopher-code)** — a from-scratch
> Go rewrite of an agentic coding assistant, archived by its original author while still
> early-stage. Gopher picks up that codebase and carries it forward.

---

## Why This Exists

An incredible category of tool trapped inside a huge TypeScript monolith that ships
with Node.js, a bundled Ink/React renderer, native addons for every platform, and a `node_modules`
tree deeper than the Mariana Trench.

Gopher asks: **what if it was just a binary?**

- **Fast cold start** vs multi-second Node.js bootstrap
- **Single static binary** — `go build` and ship, no `npm install`, no native addons
- **Native concurrency** — goroutines for parallel tool execution, not Promise.all
- **Cross-compile in seconds** — `GOOS=linux GOARCH=arm64 go build` and done
- **Memory efficient** — no V8 heap, no garbage collector pauses from React re-renders
- **Hackable** — read the source in an afternoon, not a week

---

## Repository Layout

```text
gopher/
├── cmd/gopher/            # CLI entry point (main.go) + handlers/ subpackage
├── pkg/                   # ~46 packages; the ones that matter most:
│   ├── ui/                # The real TUI — Bubble Tea app, ~80 slash commands, 36 component packages
│   ├── tools/             # 41 built-in tools
│   ├── provider/          # Model providers — Anthropic, OpenAI-compatible (Ollama, vLLM, LM Studio);
│   │                      #   Bedrock/Vertex are present but stubbed, see progress.md
│   ├── query/             # Query loop orchestration
│   ├── session/           # Session state, persistence, teams, worktrees
│   ├── permissions/       # Tool permission evaluation (7 modes)
│   ├── mcp/               # Model Context Protocol client (stdio transport only today)
│   ├── hooks/             # 27 hook events
│   ├── compact/           # Token budget & context compaction
│   ├── skills/            # Skill loading & built-in agents
│   ├── agents/            # Subagent definitions
│   ├── plugins/           # Plugin types (install/uninstall not yet implemented)
│   ├── bridge/            # IDE / remote bridge (SSE, WebSocket, hybrid transports)
│   └── ...                # analytics, auth, keybindings, vim, lsp, ide, voice, and more — see progress.md
├── internal/
│   ├── cli/               # Legacy line-oriented REPL (GOPHER_OLD_UI=1); superseded by pkg/ui
│   └── testharness/       # Golden file / differential test framework
├── testdata/              # Parity test fixtures (JSON)
└── scripts/               # coverage-report.sh, validate-ts-binary.sh, TUI scenario capture
```

Full feature-by-feature status lives in [`progress.md`](progress.md); the backlog to close
remaining gaps is in [`TASKS.md`](TASKS.md).

---

## Built With

Gopher is built on the modern Go ecosystem. No legacy. No baggage.

| Concern | Package | Why |
|---------|---------|-----|
| Terminal UI | `charm.land/bubbletea/v2` | Elm-architecture TUI |
| Styling | `charm.land/lipgloss/v2` | ANSI styling & layout |
| Markdown | `charm.land/glamour/v2` | Terminal markdown rendering |
| Components | `charm.land/bubbles/v2` | Spinner, viewport, text input, progress bar |
| Prompts | `charm.land/huh/v2` | Permission dialogs & interactive forms |
| Syntax HL | `github.com/alecthomas/chroma/v2` | Code highlighting in terminal output |
| API streaming | `github.com/tmaxmax/go-sse` | Server-Sent Events for LLM API |
| HTTP | `github.com/hashicorp/go-retryablehttp` | Resilient HTTP with exponential backoff |
| MCP | `github.com/mark3labs/mcp-go` | Model Context Protocol client SDK |
| Shell parsing | `mvdan.cc/sh/v3` | Bash AST for security analysis |
| Glob | `github.com/bmatcuk/doublestar/v4` | `**` pattern support |
| Git | `github.com/go-git/go-git/v5` | Pure Go git operations |
| GitHub | `github.com/google/go-github/v84` | GitHub API v84 |
| Config | `github.com/knadh/koanf/v2` | Multi-source configuration |
| Schema | `github.com/santhosh-tekuri/jsonschema/v6` | Tool input validation |
| Concurrency | `golang.org/x/sync` | errgroup + semaphore for parallel tools |
| Caching | `github.com/hashicorp/golang-lru/v2` | File state LRU cache |
| Keyring | `github.com/zalando/go-keyring` | OS-native credential storage |
| Observability | `go.opentelemetry.io/otel` | Traces, metrics, spans |

---

## Quickstart

```bash
# Clone
git clone https://github.com/Haris0059/gopher.git
cd gopher

# Build
go build -o gopher ./cmd/gopher

# Run interactive REPL
./gopher

# Run headless (-p is a flag, the prompt is a positional arg or --query)
./gopher -p "explain this codebase"

# Point at a local model instead of Anthropic (e.g. Ollama)
./gopher --provider openai --api-url http://localhost:11434 --model qwen2.5-coder:7b

# Cross-compile for Linux ARM64
GOOS=linux GOARCH=arm64 go build -o gopher-linux-arm64 ./cmd/gopher
```

### CLI Flags

Gopher uses Go's standard `flag` package (no getopt-style bundling). Only six flags have a
short form; the prompt itself is a positional argument, not a flag value.

```text
Usage: gopher [flags] [prompt...]

  -p, --print                Print response and exit (headless mode; bool — pair with a
                              positional prompt or --query)
  -c, --continue              Continue the most recent conversation
  -r, --resume string         Resume a conversation by session ID
  -d, --debug                 Enable debug mode
  -n, --name string           Display name for the session
  -w, --worktree               Create a git worktree for the session
      --model string           Model to use (default: claude-sonnet-4-20250514)
      --provider string        anthropic | bedrock | vertex | openai
      --api-url string         API base URL (custom/local providers)
      --cwd string             Working directory
      --output-format string   text | json | stream-json
      --verbose                Enable verbose output
      --thinking string        enabled | disabled
      --effort string          low | medium | high | max
      --dangerously-skip-permissions
                                Bypass all permission checks
      ...                      ~30 more; run `gopher --help` for the full list
```

Subcommands: `auth`, `mcp`, `plugin`, `agents`, `doctor`, `update`, `install`, `setup-token`,
`completion`, `remote-control`, `auto-mode`.

---

## Architecture

```text
                    ┌─────────────────────────┐
                    │   TUI (pkg/ui) / CLI     │
                    │      (cmd/gopher)        │
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │      Query Engine        │
                    │     (pkg/query)          │
                    └──┬──────────┬──────────┬┘
                       │          │          │
              ┌────────▼───┐ ┌───▼────┐ ┌───▼────────┐
              │  Provider   │ │ Tools  │ │  Session   │
              │(Anthropic/  │ │ (x41)  │ │ (persist)  │
              │ OpenAI-compat)│└───┬────┘ └────────────┘
              └──────┬─────┘     │
              ┌──────▼─────┐ ┌───▼────────────┐
              │ SSE Stream │ │  Permissions    │
              └────────────┘ │  Shell Parse    │
                             │  MCP Client     │
                             └────────────────┘
```

**Key design decisions:**

- **No global state** — all state flows through explicit function parameters and the session store
- **Context-first cancellation** — every goroutine respects `context.Context`
- **Interfaces at boundaries** — provider, tools, and transport are all interface-based for testing
- **Behavioral parity tests** — `pkg/ui/visual_parity_test.go` and `testdata/*.json` assert against
  captured reference behavior; see `progress.md` for exactly what's covered

---

## Status

Gopher is under active development, not yet feature-complete. **Works today:** the interactive
TUI, ~80 slash commands, 41 built-in tools, the Anthropic and OpenAI-compatible providers, hooks
(27 events), permissions, sessions/resume, compact, MCP over stdio, skills, and the IDE/remote
bridge. **Not yet:** Bedrock and Vertex providers (present but stubbed), MCP over HTTP/SSE/WS,
plugin install/uninstall, a real self-updater, and voice input. Full detail in
[`progress.md`](progress.md); the backlog is [`TASKS.md`](TASKS.md).

---

## Contributing

Contributions, ideas, and feedback are welcome.

```bash
# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Update golden files
go test ./... -update

# Format & vet before submitting
gofmt -w . && go vet ./...

# Coverage report
./scripts/coverage-report.sh
```

`scripts/validate-ts-binary.sh` diffs behavior against a sibling checkout of the original TS
Claude Code — only runs if `../research/claude-code-source-build/dist/cli.js` exists locally.

---

<p align="center">
  <sub>Forked from and inspired by <a href="https://github.com/ProjectBarks/gopher-code">ProjectBarks/gopher-code</a></sub><br>
  <sub>Gopher is an independent project. Not affiliated with Anthropic.</sub>
</p>
