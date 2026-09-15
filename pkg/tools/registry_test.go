package tools

import (
	"strings"
	"testing"
)

func TestToolDefinitions_PrefersPromptOverDescription(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&FileReadTool{})
	reg.Register(&BashTool{}) // no Prompt() — exercises the Description() fallback

	defs := reg.ToolDefinitions()
	byName := make(map[string]string, len(defs))
	for _, d := range defs {
		byName[d.Name] = d.Description
	}

	readDesc, ok := byName["Read"]
	if !ok {
		t.Fatal("Read tool missing from ToolDefinitions()")
	}
	if !strings.Contains(readDesc, "absolute path") {
		t.Errorf("Read description = %q, want it to include the Prompt() usage text (absolute path)", readDesc)
	}
	if readDesc == (&FileReadTool{}).Description() {
		t.Error("Read description should be the full Prompt(), not the short Description()")
	}

	bashDesc, ok := byName["Bash"]
	if !ok {
		t.Fatal("Bash tool missing from ToolDefinitions()")
	}
	if bashDesc != (&BashTool{}).Description() {
		t.Errorf("Bash description = %q, want the plain Description() fallback since Bash has no Prompt()", bashDesc)
	}
}

// TestRegistry_UnregisterRemovesFromToolDefinitions guards the interactive-mode
// gating in cmd/gopher/main.go, which unregisters SyntheticOutputTool
// (SDK/headless-only, per SyntheticOutputTool.ts:22-26) when not running
// non-interactively — otherwise a model can call it as pointless noise on an
// ordinary chat turn.
func TestRegistry_UnregisterRemovesFromToolDefinitions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&SyntheticOutputTool{})
	reg.Register(&BashTool{})

	reg.Unregister((&SyntheticOutputTool{}).Name())

	for _, d := range reg.ToolDefinitions() {
		if d.Name == "SyntheticOutput" {
			t.Fatal("SyntheticOutput still present in ToolDefinitions() after Unregister")
		}
	}
	if reg.Get("Bash") == nil {
		t.Error("unrelated tool Bash should still be registered")
	}
}
