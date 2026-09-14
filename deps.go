//go:build deps

// This file anchors all declared dependencies so `go mod tidy` does not
// remove them before they are imported in real source files.  Delete this
// file once every package below has at least one real import.

package deps

import (
	// Terminal UI — Charm v2
	_ "charm.land/bubbles/v2"
	_ "charm.land/bubbletea/v2"
	_ "charm.land/glamour/v2"
	_ "charm.land/huh/v2"
	_ "charm.land/lipgloss/v2"
	_ "charm.land/log/v2"
	_ "github.com/charmbracelet/x/ansi"
	_ "github.com/charmbracelet/x/term"

	// API & streaming
	_ "github.com/coder/websocket"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/hashicorp/go-retryablehttp"
	_ "github.com/tmaxmax/go-sse"

	// MCP
	_ "github.com/mark3labs/mcp-go/client"
	_ "github.com/mark3labs/mcp-go/mcp"

	// Shell & file operations
	_ "github.com/alecthomas/chroma/v2"
	_ "github.com/bmatcuk/doublestar/v4"
	_ "github.com/fsnotify/fsnotify"
	_ "github.com/google/renameio"
	_ "github.com/sabhiram/go-gitignore"
	_ "github.com/saintfish/chardet"
	_ "github.com/sergi/go-diff/diffmatchpatch"
	_ "mvdan.cc/sh/v3/syntax"

	// Config & validation
	_ "github.com/goccy/go-yaml"
	_ "github.com/knadh/koanf/v2"
	_ "github.com/santhosh-tekuri/jsonschema/v6"
	_ "github.com/spf13/cobra"

	// Git & GitHub
	_ "github.com/go-git/go-git/v5"
	_ "github.com/google/go-github/v84/github"
	_ "golang.org/x/oauth2"

	// Concurrency
	_ "golang.org/x/sync/errgroup"
	_ "golang.org/x/sync/semaphore"

	// Caching
	_ "github.com/hashicorp/golang-lru/v2"

	// Security
	_ "github.com/zalando/go-keyring"

	// Observability
	_ "github.com/growthbook/growthbook-golang"
	_ "go.opentelemetry.io/otel"
	_ "go.opentelemetry.io/otel/trace"

	// Image & PDF
	_ "github.com/disintegration/imaging"
	_ "github.com/ledongthuc/pdf"
)
