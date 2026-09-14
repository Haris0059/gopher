package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Haris0059/gopher/pkg/ui/theme"
)

// WelcomeScreenWidth matches Claude Code's WELCOME_V2_WIDTH.
const WelcomeScreenWidth = 58

// Version is the current gopher version.
// Patch segment is the count of Haris0059's commits on this project.
const Version = "0.3.043"

// gopherIcon is the compact block-character mark shown at startup, in the
// style of Claude Code's v2 splash. Same shape, recolored to the Gopher
// accent instead of Claude's color.
var gopherIcon = [3]string{
	" ▐▛███▛█",
	"▝▜██████▀",
	"  ▝▝ ▝▝",
}

// gopherIconWidth is the rune width of the widest icon row, used to pad
// the shorter rows so the text column after the icon stays aligned.
const gopherIconWidth = 9

// WelcomeScreen renders the initial greeting: a 3-line icon + version/model/cwd
// splash, no border.
type WelcomeScreen struct {
	model   string
	cwd     string
	version string
	theme   theme.Theme
	width   int
	height  int
}

// NewWelcomeScreen creates a welcome screen with session context.
func NewWelcomeScreen(t theme.Theme, model, cwd string) *WelcomeScreen {
	return &WelcomeScreen{
		model:   model,
		cwd:     cwd,
		version: Version,
		theme:   t,
		width:   WelcomeScreenWidth,
		height:  14,
	}
}

// Init implements tea.Model.
func (ws *WelcomeScreen) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (ws *WelcomeScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return ws, nil
}

// View renders the welcome splash: a 3-line block icon next to the
// version, model, and working directory. No border, no box.
func (ws *WelcomeScreen) View() tea.View {
	return tea.NewView(renderGopherSplash(ws.theme.Colors(), ws.width, ws.version, ws.model, ws.cwd))
}

// renderGopherSplash renders the persistent 3-line identity block — icon +
// "Gopher v{version}" / model / cwd — shared verbatim between WelcomeScreen
// and Header so the top of the screen never changes look, before or after
// the welcome screen is dismissed.
func renderGopherSplash(cs theme.ColorScheme, width int, version, model, cwd string) string {
	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cs.Primary)).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cs.TextPrimary)).Bold(true)
	subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cs.TextSecondary))

	// Text column stays fixed regardless of which icon row is widest.
	padIcon := func(s string) string {
		n := gopherIconWidth - len([]rune(s))
		if n < 0 {
			n = 0
		}
		return s + strings.Repeat(" ", n)
	}

	maxCWD := width - gopherIconWidth - 4
	if maxCWD < 10 {
		maxCWD = 10
	}

	lines := []string{
		iconStyle.Render(padIcon(gopherIcon[0])) + "  " + titleStyle.Render(fmt.Sprintf("Gopher v%s", version)),
		iconStyle.Render(padIcon(gopherIcon[1])) + "  " + subtleStyle.Render(model),
		iconStyle.Render(padIcon(gopherIcon[2])) + "  " + subtleStyle.Render(abbreviateCWD(cwd, maxCWD)),
	}

	return strings.Join(lines, "\n")
}

// SetSize updates the screen dimensions.
// The box adapts to the terminal width — it can be narrower than
// WelcomeScreenWidth for small terminals, or expand up to terminal width.
func (ws *WelcomeScreen) SetSize(width, height int) {
	// Box content width = terminal width - 2 (for │ borders)
	ws.width = width - 2
	if ws.width < 20 {
		ws.width = 20
	}
	ws.height = height
}

// abbreviateCWD shortens a path for display.
func abbreviateCWD(path string, maxLen int) string {
	if len([]rune(path)) <= maxLen {
		return path
	}
	// Replace home dir prefix
	if strings.HasPrefix(path, "/Users/") {
		parts := strings.SplitN(path, "/", 4)
		if len(parts) >= 4 {
			path = "~/" + parts[3]
		}
	}
	runes := []rune(path)
	if len(runes) <= maxLen {
		return path
	}
	if maxLen <= 1 {
		return "…"
	}
	return "…" + string(runes[len(runes)-maxLen+1:])
}

var _ tea.Model = (*WelcomeScreen)(nil)
