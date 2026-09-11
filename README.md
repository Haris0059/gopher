# Gopher

<p align="center">
  <img src="assets/go-gopher-pixel-art.png" width="200" alt="Gopher mascot — pixel art Go gopher">
</p>

<p align="center">
  <strong>An agentic coding CLI, written in Go. Zero Node.js. Zero Electron. One binary.</strong>
</p>

<p align="center">
  <img src="assets/demo.gif" alt="Gopher demo">
</p>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white">
    <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.24+">
  </picture>
  <img src="https://img.shields.io/badge/Tools-33_Built--In-orange?style=for-the-badge" alt="33 Tools">
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
├── cmd/gopher/            # CLI entry point & REPL
│   └── main.go
├── pkg/                   # Core packages
│   ├── compact/           # Token budget & context compaction
│   ├── message/           # Message types & normalization
│   ├── mcp/               # Model Context Protocol client
│   ├── permissions/       # Tool permission evaluation
│   ├── prompt/            # System prompt assembly
│   ├── provider/          # Model providers — Anthropic, OpenAI-compatible (Ollama, vLLM, LM Studio), Bedrock, Vertex
│   ├── query/             # Query loop orchestration
│   ├── session/           # Session state & persistence
│   └── tools/             # 33 built-in tools
├── internal/
│   ├── cli/               # Bubble Tea TUI renderer
│   └── testharness/       # Golden file test framework
├── testdata/              # Parity test fixtures
└── notes/                 # Architecture & dependency docs
```

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

# Run headless
./gopher -p "explain this codebase"

# Point at a local model instead of Anthropic (e.g. Ollama)
./gopher --provider openai --api-url http://localhost:11434 --model qwen2.5-coder:7b

# Cross-compile for Linux ARM64
GOOS=linux GOARCH=arm64 go build -o gopher-linux-arm64 ./cmd/gopher
```

### CLI Flags

```text
Usage: gopher [flags]

Flags:
  -p, --print string       Run a single query in headless mode
  -m, --model string       Model to use (default: claude-sonnet-4-20250514)
  --provider string        Provider: anthropic, bedrock, vertex, openai
  --api-url string         API base URL (for custom/local providers)
  -c, --cwd string         Working directory
  -r, --resume string      Resume a previous session by ID
  -o, --output-format      Output format: text, json, stream-json
  -v, --verbose            Enable verbose logging
```

---

## Architecture

```text
                    ┌─────────────────────────┐
                    │     CLI / Bubble Tea     │
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
              │(Anthropic/  │ │ (x33)  │ │ (persist)  │
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
- **Golden file tests** — parity tests run against captured terminal-UI transcripts

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
```

---

<p align="center">
  <sub>Forked from and inspired by <a href="https://github.com/ProjectBarks/gopher-code">ProjectBarks/gopher-code</a></sub><br>
  <sub>Gopher is an independent project. Not affiliated with Anthropic.</sub>
</p>
