package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Haris0059/gopher/pkg/message"
	"github.com/Haris0059/gopher/pkg/ui/theme"
)

// AddMessageMsg triggers adding a message to the conversation pane.
type AddMessageMsg struct {
	Message message.Message
}

// ClearMessagesMsg clears all messages from the conversation pane.
type ClearMessagesMsg struct{}

// ConversationPane displays scrollable message history.
// It uses MessageBubble for rendering individual messages with Glamour
// markdown, tool call styling, and role-based formatting.
type ConversationPane struct {
	messages []message.Message
	rendered []string // Pre-rendered message strings (via MessageBubble)
	bubble   *MessageBubble
	width    int
	height   int
	focused  bool

	// Scroll state
	scrollOffset int // Lines scrolled from bottom (0 = at bottom)
	autoScroll   bool

	// Streaming state
	streamingText string
}

// NewConversationPane creates a new empty conversation pane.
func NewConversationPane() *ConversationPane {
	t := theme.Current()
	return &ConversationPane{
		messages:   make([]message.Message, 0),
		rendered:   make([]string, 0),
		bubble:     NewMessageBubble(t, 80),
		autoScroll: true,
	}
}

// Init initializes the conversation pane.
func (cp *ConversationPane) Init() tea.Cmd {
	return nil
}

// Update handles messages for the conversation pane.
func (cp *ConversationPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AddMessageMsg:
		cp.AddMessage(msg.Message)
		return cp, nil

	case ClearMessagesMsg:
		cp.messages = cp.messages[:0]
		cp.rendered = cp.rendered[:0]
		cp.scrollOffset = 0
		return cp, nil

	case tea.KeyPressMsg:
		return cp.handleKey(msg)

	case tea.MouseWheelMsg:
		return cp.handleWheel(msg)

	case tea.WindowSizeMsg:
		cp.SetSize(msg.Width, msg.Height)
		return cp, nil
	}
	return cp, nil
}

// View renders the conversation pane.
func (cp *ConversationPane) View() tea.View {
	if cp.width == 0 || cp.height == 0 {
		return tea.NewView("")
	}

	if len(cp.rendered) == 0 && cp.streamingText == "" {
		// No placeholder text — an empty conversation area just stays blank
		// under the header, matching Claude Code's actual behavior.
		return tea.NewView("")
	}

	allLines := cp.allLines()

	// Apply viewport: show last `height` lines (with scroll offset)
	totalLines := len(allLines)
	if totalLines == 0 {
		return tea.NewView("")
	}

	viewStart := totalLines - cp.height - cp.scrollOffset
	if viewStart < 0 {
		viewStart = 0
	}
	viewEnd := viewStart + cp.height
	if viewEnd > totalLines {
		viewEnd = totalLines
	}

	visible := allLines[viewStart:viewEnd]

	// Pad to fill height
	for len(visible) < cp.height {
		visible = append(visible, "")
	}

	return tea.NewView(strings.Join(visible, "\n"))
}

// allLines collects all rendered lines, with exactly one blank line
// separating each message (and the in-progress streaming reply) from the
// next.
func (cp *ConversationPane) allLines() []string {
	var allLines []string
	for i, r := range cp.rendered {
		if i > 0 {
			allLines = append(allLines, "")
		}
		allLines = append(allLines, strings.Split(r, "\n")...)
	}

	if cp.streamingText != "" {
		if len(cp.rendered) > 0 {
			allLines = append(allLines, "")
		}
		allLines = append(allLines, strings.Split(cp.streamingText, "\n")...)
	}
	return allLines
}

// SetSize sets the dimensions of the conversation pane.
func (cp *ConversationPane) SetSize(width, height int) {
	cp.width = width
	cp.height = height
	cp.bubble.SetWidth(width)
	// Re-render all messages with new width
	cp.rerenderAll()
}

// Focus gives focus to this pane.
func (cp *ConversationPane) Focus() {
	cp.focused = true
}

// Blur removes focus from this pane.
func (cp *ConversationPane) Blur() {
	cp.focused = false
}

// Focused returns whether this pane has focus.
func (cp *ConversationPane) Focused() bool {
	return cp.focused
}

// AddMessage adds a message to the conversation and renders it via MessageBubble.
func (cp *ConversationPane) AddMessage(msg message.Message) {
	cp.messages = append(cp.messages, msg)
	cp.rendered = append(cp.rendered, cp.bubble.Render(&msg))
	if cp.autoScroll {
		cp.scrollOffset = 0
	}
}

// SetStreamingText sets the current streaming text buffer.
func (cp *ConversationPane) SetStreamingText(text string) {
	cp.streamingText = text
}

// ClearStreamingText clears the streaming text buffer.
func (cp *ConversationPane) ClearStreamingText() {
	cp.streamingText = ""
}

// MessageCount returns the number of messages.
func (cp *ConversationPane) MessageCount() int {
	return len(cp.messages)
}

// IsEmpty reports whether the pane has nothing to show — no rendered
// messages and no in-progress streaming text. Callers use this to omit
// the conversation section entirely rather than rendering a blank line
// for it (e.g. the moment welcome is dismissed but before any message
// has been added).
func (cp *ConversationPane) IsEmpty() bool {
	return len(cp.rendered) == 0 && cp.streamingText == ""
}

// --- Internal ---

func (cp *ConversationPane) handleKey(msg tea.KeyPressMsg) (*ConversationPane, tea.Cmd) {
	switch msg.Code {
	case tea.KeyUp:
		cp.scrollBy(1)
	case tea.KeyDown:
		cp.scrollBy(-1)
	case tea.KeyPgUp:
		cp.scrollBy(cp.height)
	case tea.KeyPgDown:
		cp.scrollBy(-cp.height)
	}
	return cp, nil
}

const wheelScrollLines = 3

func (cp *ConversationPane) handleWheel(msg tea.MouseWheelMsg) (*ConversationPane, tea.Cmd) {
	switch msg.Button {
	case tea.MouseWheelUp:
		cp.scrollBy(wheelScrollLines)
	case tea.MouseWheelDown:
		cp.scrollBy(-wheelScrollLines)
	}
	return cp, nil
}

// scrollBy moves the viewport by delta lines (positive = up, toward older
// messages), clamping to [0, maxScrollOffset] and toggling autoScroll when
// the viewport returns to the bottom.
func (cp *ConversationPane) scrollBy(delta int) {
	cp.scrollOffset += delta
	if max := cp.maxScrollOffset(); cp.scrollOffset > max {
		cp.scrollOffset = max
	}
	if cp.scrollOffset < 0 {
		cp.scrollOffset = 0
	}
	cp.autoScroll = cp.scrollOffset == 0
}

func (cp *ConversationPane) maxScrollOffset() int {
	max := len(cp.allLines()) - cp.height
	if max < 0 {
		return 0
	}
	return max
}

func (cp *ConversationPane) rerenderAll() {
	cp.rendered = make([]string, len(cp.messages))
	for i := range cp.messages {
		cp.rendered[i] = cp.bubble.Render(&cp.messages[i])
	}
}
