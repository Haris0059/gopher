package tools

import (
	"os"
	"strings"
	"sync"
)

// Source: bootstrap/state.ts (kairosActive, userMsgOptIn) and
// tools/BriefTool/BriefTool.ts (isBriefEntitled, isBriefEnabled).
//
// ToolContext carries no session/app state and RegisterDefaults takes no
// arguments, so — mirroring the TS reference's own module-level getters and
// setters — this gate lives as package state, set by whoever owns the
// session (main.go at startup, the /brief handler on toggle).

var briefState struct {
	mu           sync.RWMutex
	kairosActive bool
	userMsgOptIn bool
}

// SetKairosActive records whether Kairos (brief-only) mode is active for the
// current session. Source: bootstrap/state.ts — kairosActive
func SetKairosActive(v bool) {
	briefState.mu.Lock()
	defer briefState.mu.Unlock()
	briefState.kairosActive = v
}

// SetUserMsgOptIn records whether the user has opted into brief mode (e.g.
// via /brief). Source: bootstrap/state.ts — userMsgOptIn
func SetUserMsgOptIn(v bool) {
	briefState.mu.Lock()
	defer briefState.mu.Unlock()
	briefState.userMsgOptIn = v
}

// GetUserMsgOptIn reports the current opt-in state, e.g. for the /brief
// toggle handler to read before flipping it.
func GetUserMsgOptIn() bool {
	_, userMsgOptIn := getBriefFlags()
	return userMsgOptIn
}

func getBriefFlags() (kairosActive, userMsgOptIn bool) {
	briefState.mu.RLock()
	defer briefState.mu.RUnlock()
	return briefState.kairosActive, briefState.userMsgOptIn
}

// BriefEntitled reports whether the current process is entitled to use the
// Brief/SendUserMessage tool. There is no Go equivalent of the GrowthBook
// feature-flag gate wired here, so entitlement is Kairos activity plus the
// CLAUDE_CODE_BRIEF dev/testing override.
// Source: BriefTool.ts:86-98 (isBriefEntitled)
func BriefEntitled() bool {
	kairosActive, _ := getBriefFlags()
	return kairosActive || isEnvTruthy(os.Getenv("CLAUDE_CODE_BRIEF"))
}

// BriefEnabled is the unified activation gate for the Brief tool: activation
// requires explicit opt-in (userMsgOptIn) or Kairos mode, AND entitlement.
// Source: BriefTool.ts:120-127 (isBriefEnabled)
func BriefEnabled() bool {
	kairosActive, userMsgOptIn := getBriefFlags()
	return (kairosActive || userMsgOptIn) && BriefEntitled()
}

// isEnvTruthy mirrors the TS isEnvTruthy helper (also duplicated in several
// other Gopher packages, e.g. pkg/query/config.go).
func isEnvTruthy(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "1" || v == "true" || v == "yes"
}
