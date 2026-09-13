package help

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testCommands() []CommandInfo {
	return []CommandInfo{
		{Name: "help", Description: "Show this help screen", Source: "builtin"},
		{Name: "doctor", Description: "Run diagnostics", Source: "builtin"},
		{Name: "compact", Description: "Compact conversation", Source: "builtin"},
		{Name: "hidden", Description: "Hidden command", Source: "builtin", IsHidden: true},
		{Name: "foo", Description: "A custom project command", Source: "project"},
	}
}

func TestHelp_InitialState(t *testing.T) {
	m := New(testCommands(), 80, 40)
	if m.tab != TabGeneral {
		t.Error("should start on General tab")
	}
	if !m.headerFocused {
		t.Error("header should start focused")
	}
	// Hidden commands filtered; builtins vs custom partitioned by Source.
	if len(m.builtins) != 3 {
		t.Errorf("expected 3 visible builtin commands, got %d", len(m.builtins))
	}
	if len(m.customs) != 1 {
		t.Errorf("expected 1 custom command, got %d", len(m.customs))
	}
}

func TestHelp_TabCycle(t *testing.T) {
	m := New(testCommands(), 80, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.tab != TabCommands {
		t.Error("Tab should switch to Commands")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.tab != TabCustomCommands {
		t.Error("Tab again should switch to Custom commands")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.tab != TabGeneral {
		t.Error("Tab a third time should wrap back to General")
	}
}

func TestHelp_Dismiss(t *testing.T) {
	m := New(testCommands(), 80, 40)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("escape should return cmd")
	}
	msg := cmd()
	if _, ok := msg.(HelpDismissedMsg); !ok {
		t.Fatalf("expected HelpDismissedMsg, got %T", msg)
	}
}

func TestHelp_ViewGeneral(t *testing.T) {
	m := New(testCommands(), 80, 40)
	v := m.View()
	if !strings.Contains(v, "Shortcuts") {
		t.Error("general tab should show Shortcuts heading")
	}
	if !strings.Contains(v, "/ for commands") {
		t.Error("should list / for commands")
	}
}

func TestHelp_ViewCommands(t *testing.T) {
	m := New(testCommands(), 80, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	v := m.View()
	if !strings.Contains(v, "/help") {
		t.Error("commands tab should show /help")
	}
	if !strings.Contains(v, "/doctor") {
		t.Error("commands tab should show /doctor")
	}
	if strings.Contains(v, "/foo") {
		t.Error("commands tab should not show custom command /foo")
	}
}

func TestHelp_ViewCustomCommands(t *testing.T) {
	m := New(testCommands(), 80, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	v := m.View()
	if !strings.Contains(v, "/foo") {
		t.Error("custom commands tab should show /foo")
	}
	if strings.Contains(v, "/doctor") {
		t.Error("custom commands tab should not show builtin /doctor")
	}
}

func TestHelp_CustomCommandsEmptyMessage(t *testing.T) {
	m := New([]CommandInfo{{Name: "help", Description: "x", Source: "builtin"}}, 80, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	v := m.View()
	if !strings.Contains(v, "No custom commands found") {
		t.Error("empty custom commands tab should show the empty message")
	}
}

func TestHelp_ListFocusTransitions(t *testing.T) {
	m := New(testCommands(), 80, 400)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // -> Commands
	if !m.headerFocused {
		t.Fatal("header should be focused right after a tab switch")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.headerFocused {
		t.Error("down from header should move focus into the list")
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !m.headerFocused {
		t.Error("up from the first row should return focus to the header")
	}
}

func TestHelp_ScrollClamped(t *testing.T) {
	var cmds []CommandInfo
	for i := 0; i < 50; i++ {
		cmds = append(cmds, CommandInfo{Name: string(rune('a' + i%26)), Description: "d", Source: "builtin"})
	}
	m := New(cmds, 80, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	// Move down far past the end of the list; cursor must clamp, not run away.
	for i := 0; i < 200; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if m.cursor != len(m.builtins)-1 {
		t.Errorf("cursor = %d, want %d (clamped to last row)", m.cursor, len(m.builtins)-1)
	}
	if m.scroll < 0 || m.scroll > m.cursor {
		t.Errorf("scroll = %d out of valid range for cursor %d", m.scroll, m.cursor)
	}
}
