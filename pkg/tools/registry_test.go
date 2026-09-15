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
