package tools

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// REPL session lifecycle. There is no TS source to port here — the
// reference's REPLTool is a different feature (TASKS.md TOOL-10). This file
// keeps one interpreter process alive per (sessionID, language), reusing the
// same framing pattern (a random sentinel line) for every supported
// language so pkg/tools/repltool.go doesn't need per-language read logic.

// replIdleTimeout is how long an unused REPL session is kept alive before
// the reaper kills it.
const replIdleTimeout = 30 * time.Minute

// replSession wraps one live interpreter process.
type replSession struct {
	mu       sync.Mutex
	lang     replLanguage
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	out      *bufio.Reader
	sentinel string
	started  time.Time
	lastUsed time.Time
	dead     bool
}

// replSessionKey identifies a session by the calling session ID and the
// interpreter language — the scoping this task settled on: one live
// process per (session, language).
type replSessionKey struct {
	sessionID string
	lang      replLanguage
}

var replMgr = struct {
	mu       sync.Mutex
	sessions map[replSessionKey]*replSession
	reaperOn bool
}{sessions: make(map[replSessionKey]*replSession)}

// startREPLReaper lazily starts a background goroutine that kills sessions
// idle longer than replIdleTimeout. Safe to call repeatedly.
func startREPLReaper() {
	replMgr.mu.Lock()
	defer replMgr.mu.Unlock()
	if replMgr.reaperOn {
		return
	}
	replMgr.reaperOn = true
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			reapIdleREPLSessions()
		}
	}()
}

func reapIdleREPLSessions() {
	now := time.Now()
	var stale []*replSession
	replMgr.mu.Lock()
	for key, s := range replMgr.sessions {
		s.mu.Lock()
		idle := now.Sub(s.lastUsed)
		s.mu.Unlock()
		if idle > replIdleTimeout {
			stale = append(stale, s)
			delete(replMgr.sessions, key)
		}
	}
	replMgr.mu.Unlock()
	for _, s := range stale {
		s.kill()
	}
}

// getOrStartREPLSession returns the live session for (sessionID, lang),
// starting a new interpreter process if none exists yet or the prior one
// died. cwd is only used when starting a new process.
func getOrStartREPLSession(sessionID string, lang replLanguage, cwd string) (*replSession, error) {
	startREPLReaper()
	key := replSessionKey{sessionID: sessionID, lang: lang}

	replMgr.mu.Lock()
	s := replMgr.sessions[key]
	replMgr.mu.Unlock()

	if s != nil {
		s.mu.Lock()
		alive := !s.dead
		s.mu.Unlock()
		if alive {
			return s, nil
		}
	}

	ns, err := newREPLSession(lang, cwd)
	if err != nil {
		return nil, err
	}
	replMgr.mu.Lock()
	replMgr.sessions[key] = ns
	replMgr.mu.Unlock()
	return ns, nil
}

// restartREPLSession kills any existing session for (sessionID, lang) and
// starts a fresh one, used for the tool's restart:true input.
func restartREPLSession(sessionID string, lang replLanguage, cwd string) (*replSession, error) {
	key := replSessionKey{sessionID: sessionID, lang: lang}
	replMgr.mu.Lock()
	old := replMgr.sessions[key]
	delete(replMgr.sessions, key)
	replMgr.mu.Unlock()
	if old != nil {
		old.kill()
	}
	return getOrStartREPLSession(sessionID, lang, cwd)
}

// dropREPLSession removes a session from the registry (without necessarily
// killing an already-dead process) — used after a timeout, where the
// process was killed but state can no longer be trusted.
func dropREPLSession(sessionID string, lang replLanguage) {
	key := replSessionKey{sessionID: sessionID, lang: lang}
	replMgr.mu.Lock()
	delete(replMgr.sessions, key)
	replMgr.mu.Unlock()
}

// CloseREPLSessions kills and removes every REPL session belonging to the
// given session ID.
func CloseREPLSessions(sessionID string) {
	var victims []*replSession
	replMgr.mu.Lock()
	for key, s := range replMgr.sessions {
		if key.sessionID == sessionID {
			victims = append(victims, s)
			delete(replMgr.sessions, key)
		}
	}
	replMgr.mu.Unlock()
	for _, s := range victims {
		s.kill()
	}
}

// CloseAllREPLSessions kills and removes every live REPL session. Intended
// to run at process/TUI shutdown so no interpreter child outlives Gopher —
// see internal/cli/tui_v2.go:RunTUIV2.
func CloseAllREPLSessions() {
	var victims []*replSession
	replMgr.mu.Lock()
	for key, s := range replMgr.sessions {
		victims = append(victims, s)
		delete(replMgr.sessions, key)
	}
	replMgr.mu.Unlock()
	for _, s := range victims {
		s.kill()
	}
}

func newSentinel() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "GOPHER_REPL_" + hex.EncodeToString(b)
}

func newREPLSession(lang replLanguage, cwd string) (*replSession, error) {
	sentinel := newSentinel()

	var cmd *exec.Cmd
	if lang == replBash {
		shell := getUserShell()
		cmd = exec.Command(shell, "-s")
	} else {
		spec, ok := replDrivers[lang]
		if !ok {
			return nil, fmt.Errorf("unsupported REPL language: %s", lang)
		}
		argv := spec.Argv(sentinel)
		cmd = exec.Command(argv[0], argv[1:]...)
	}
	cmd.Dir = cwd
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("repl: creating stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("repl: creating stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into the same framed stream

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("repl: starting %s interpreter: %w", lang, err)
	}

	now := time.Now()
	return &replSession{
		lang:     lang,
		cmd:      cmd,
		stdin:    stdin,
		out:      bufio.NewReader(stdout),
		sentinel: sentinel,
		started:  now,
		lastUsed: now,
	}, nil
}

// replRunResult is the outcome of executing one code block in a session.
type replRunResult struct {
	Output    string
	ExitCode  int
	TimedOut  bool
	Restarted bool // true if the underlying process had to be replaced
}

// Run executes code in the session, blocking until the sentinel comes back
// or ctx is done. On timeout the process is killed — its in-memory state is
// no longer trustworthy — and the caller must drop the session.
func (s *replSession) Run(ctx context.Context, code string) (*replRunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dead {
		return nil, fmt.Errorf("repl session is no longer running")
	}
	s.lastUsed = time.Now()

	frame := replDrivers[s.lang].FrameCode(code, s.sentinel)

	writeErr := make(chan error, 1)
	go func() {
		_, err := io.WriteString(s.stdin, frame)
		writeErr <- err
	}()

	type readResult struct {
		lines []string
		err   error
	}
	readDone := make(chan readResult, 1)
	go func() {
		var lines []string
		for {
			line, err := s.out.ReadString('\n')
			if line != "" {
				trimmed := strings.TrimRight(line, "\r\n")
				if trimmed == s.sentinel {
					// Next line is the status.
					statusLine, err2 := s.out.ReadString('\n')
					status := strings.TrimRight(statusLine, "\r\n")
					if err2 != nil && status == "" {
						readDone <- readResult{lines: lines, err: err2}
						return
					}
					lines = append(lines, "\x00STATUS\x00"+status)
					readDone <- readResult{lines: lines}
					return
				}
				lines = append(lines, trimmed)
			}
			if err != nil {
				readDone <- readResult{lines: lines, err: err}
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		s.killLocked()
		return &replRunResult{TimedOut: true}, nil
	case err := <-writeErr:
		if err != nil {
			s.killLocked()
			return nil, fmt.Errorf("repl: writing code to interpreter: %w", err)
		}
	}

	select {
	case <-ctx.Done():
		s.killLocked()
		return &replRunResult{TimedOut: true}, nil
	case res := <-readDone:
		if res.err != nil && len(res.lines) == 0 {
			s.killLocked()
			return nil, fmt.Errorf("repl: interpreter exited: %w", res.err)
		}
		status := 0
		out := res.lines
		if n := len(out); n > 0 && strings.HasPrefix(out[n-1], "\x00STATUS\x00") {
			status, _ = strconv.Atoi(strings.TrimPrefix(out[n-1], "\x00STATUS\x00"))
			out = out[:n-1]
		}
		return &replRunResult{
			Output:   strings.Join(out, "\n"),
			ExitCode: status,
		}, nil
	}
}

func (s *replSession) kill() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.killLocked()
}

func (s *replSession) killLocked() {
	if s.dead {
		return
	}
	s.dead = true
	_ = s.stdin.Close()
	if s.cmd.Process != nil {
		// Kill the whole process group (Setpgid above), matching bash.go's
		// approach, so a REPL that spawned children doesn't leak them.
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	}
	go func() { _ = s.cmd.Wait() }()
}
