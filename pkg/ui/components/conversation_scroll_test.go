package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Haris0059/gopher/pkg/message"
)

func newScrollTestPane(t *testing.T, height int, markers []string) *ConversationPane {
	t.Helper()
	cp := NewConversationPane()
	cp.SetSize(80, height)
	for _, m := range markers {
		cp.AddMessage(message.Message{
			Role:    message.RoleUser,
			Content: []message.ContentBlock{{Type: message.ContentText, Text: m}},
		})
	}
	return cp
}

func TestConversationPane_MouseWheelScroll(t *testing.T) {
	markers := []string{"ZZaaa", "ZZbbb", "ZZccc", "ZZddd", "ZZeee"}
	cp := newScrollTestPane(t, 3, markers)

	if cp.scrollOffset != 0 || !cp.autoScroll {
		t.Fatalf("expected fresh pane at bottom with autoScroll, got offset=%d autoScroll=%v", cp.scrollOffset, cp.autoScroll)
	}

	// Wheel up raises the offset and disables autoScroll.
	cp.handleWheel(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if cp.scrollOffset != wheelScrollLines {
		t.Fatalf("wheel up: expected offset=%d, got %d", wheelScrollLines, cp.scrollOffset)
	}
	if cp.autoScroll {
		t.Fatal("wheel up: autoScroll should be disabled once scrolled")
	}

	// Repeated wheel up clamps at maxScrollOffset instead of growing forever.
	for i := 0; i < 20; i++ {
		cp.handleWheel(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	}
	max := cp.maxScrollOffset()
	if cp.scrollOffset != max {
		t.Fatalf("wheel up clamp: expected offset=%d, got %d", max, cp.scrollOffset)
	}
	if !strings.Contains(stripANSI(cp.View().Content), "ZZaaa") {
		t.Fatalf("at clamp, first message ZZaaa must be visible, got:\n%s", stripANSI(cp.View().Content))
	}

	// Wheel down walks back to the bottom and restores autoScroll exactly at 0.
	for i := 0; i < 20; i++ {
		cp.handleWheel(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	}
	if cp.scrollOffset != 0 {
		t.Fatalf("wheel down: expected offset=0, got %d", cp.scrollOffset)
	}
	if !cp.autoScroll {
		t.Fatal("wheel down: autoScroll should be restored at offset=0")
	}
	if !strings.Contains(stripANSI(cp.View().Content), "ZZeee") {
		t.Fatalf("back at bottom, last message ZZeee must be visible, got:\n%s", stripANSI(cp.View().Content))
	}
}

func TestConversationPane_WheelOnEmptyPaneIsNoop(t *testing.T) {
	cp := NewConversationPane()
	cp.SetSize(80, 3)

	cp.handleWheel(tea.MouseWheelMsg{Button: tea.MouseWheelUp})

	if cp.scrollOffset != 0 {
		t.Fatalf("wheel on empty pane must be a no-op, got offset=%d", cp.scrollOffset)
	}
}
