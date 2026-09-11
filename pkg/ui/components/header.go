package components

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Haris0059/gopher/pkg/ui/theme"
)

// HeaderUpdateMsg updates the header display.
type HeaderUpdateMsg struct {
	Model       string
	SessionName string
	CWD         string
}

// Header displays the top bar with model, session, and cwd.
type Header struct {
	modelName   string
	sessionName string
	cwd         string
	theme       theme.Theme
	width       int
	height      int
	focused     bool
}

// NewHeader creates a new header component.
func NewHeader(t theme.Theme) *Header {
	return &Header{
		theme: t,
		width: 80,
		height: 1,
	}
}

// SetModel sets the model name display.
func (h *Header) SetModel(name string) { h.modelName = name }

// SetSessionName sets the session name display.
func (h *Header) SetSessionName(name string) { h.sessionName = name }

// SetCWD sets the current working directory display.
func (h *Header) SetCWD(cwd string) { h.cwd = cwd }

// Init initializes the component.
func (h *Header) Init() tea.Cmd { return nil }

// Update handles header update messages.
func (h *Header) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case HeaderUpdateMsg:
		if msg.Model != "" {
			h.modelName = msg.Model
		}
		if msg.SessionName != "" {
			h.sessionName = msg.SessionName
		}
		if msg.CWD != "" {
			h.cwd = msg.CWD
		}
	case tea.WindowSizeMsg:
		h.SetSize(msg.Width, msg.Height)
	}
	return h, nil
}

// View renders the header: the exact same 3-line icon splash as
// WelcomeScreen, so the top of the screen looks identical before and after
// the welcome screen is dismissed — nothing about it should change.
func (h *Header) View() tea.View {
	return tea.NewView(renderGopherSplash(h.theme.Colors(), h.width, Version, h.modelName, h.cwd))
}

// ModelName returns the current model name.
func (h *Header) ModelName() string { return h.modelName }

// SessionName returns the current session name.
func (h *Header) SessionName() string { return h.sessionName }

// CWD returns the current working directory.
func (h *Header) CWD() string { return h.cwd }

// SetSize sets the dimensions.
func (h *Header) SetSize(width, height int) {
	h.width = width
	h.height = height
}

func (h *Header) Focus()        { h.focused = true }
func (h *Header) Blur()         { h.focused = false }
func (h *Header) Focused() bool { return h.focused }

// abbreviatePath shortens a path to fit within maxLen.
func abbreviatePath(path string, maxLen int) string {
	if maxLen <= 0 || len(path) <= maxLen {
		return path
	}
	// Try to show just the last N components
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := 1; i < len(parts); i++ {
		shortened := "…/" + strings.Join(parts[i:], "/")
		if len(shortened) <= maxLen {
			return shortened
		}
	}
	return truncateField(path, maxLen)
}

var _ tea.Model = (*Header)(nil)
