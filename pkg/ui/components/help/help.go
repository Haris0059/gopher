// Package help provides the interactive help screen (HelpV2).
// Source: components/HelpV2/HelpV2.tsx + General.tsx + Commands.tsx
//
// Three-tab help screen: General (blurb + keyboard shortcuts), Commands
// (built-in slash commands), and Custom commands (user/project/skill
// commands discovered on disk).
package help

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Haris0059/gopher/pkg/ui/theme"
)

// docsURL is printed in the footer, matching the "For more help:" line.
const docsURL = "https://github.com/Haris0059/gopher"

// HelpDismissedMsg signals the help screen was closed.
type HelpDismissedMsg struct{}

// CommandInfo describes a slash command for the help screen.
type CommandInfo struct {
	Name        string
	Description string
	// Source identifies where the command came from: "builtin", "user",
	// "project", "skill", "bundled", etc. Builtin commands populate the
	// Commands tab; everything else populates Custom commands.
	Source string
	// IsHidden controls whether the command is hidden from the listing.
	IsHidden bool
}

// FormatDescriptionWithSource returns the description suffixed with its
// origin, matching commands.ts:730 formatDescriptionWithSource — builtin
// commands are shown plain, everything else gets a "(source)" suffix.
func FormatDescriptionWithSource(c CommandInfo) string {
	switch c.Source {
	case "", "builtin":
		return c.Description
	default:
		return fmt.Sprintf("%s (%s)", c.Description, c.Source)
	}
}

// Tab identifies which help tab is active.
type Tab int

const (
	TabGeneral Tab = iota
	TabCommands
	TabCustomCommands
)

var tabTitles = [...]string{"General", "Commands", "Custom commands"}

// Model is the interactive help screen.
type Model struct {
	tab      Tab
	builtins []CommandInfo
	customs  []CommandInfo

	// headerFocused is true while Tab/←/→ cycle tabs; false once ↓ has
	// moved focus into a command list, mirroring useTabHeaderFocus.
	headerFocused bool
	cursor        int // focused row within the active list, when list-focused
	scroll        int // index of the first visible row

	width  int
	height int
}

// New creates a help screen from the full merged command set. Hidden
// commands are dropped; the rest are partitioned into builtins (Source ==
// "builtin") and everything else (Custom commands), each sorted by name.
func New(commands []CommandInfo, width, height int) Model {
	var builtins, customs []CommandInfo
	for _, c := range commands {
		if c.IsHidden {
			continue
		}
		if c.Source == "" || c.Source == "builtin" {
			builtins = append(builtins, c)
		} else {
			customs = append(customs, c)
		}
	}
	sort.Slice(builtins, func(i, j int) bool { return builtins[i].Name < builtins[j].Name })
	sort.Slice(customs, func(i, j int) bool { return customs[i].Name < customs[j].Name })

	return Model{
		builtins:      builtins,
		customs:       customs,
		headerFocused: true,
		width:         width,
		height:        height,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyEscape:
			return m, func() tea.Msg { return HelpDismissedMsg{} }
		case tea.KeyTab, tea.KeyRight:
			m.nextTab()
		case tea.KeyLeft:
			m.prevTab()
		case tea.KeyDown:
			m.moveDown()
		case tea.KeyUp:
			m.moveUp()
		}
	}
	return m, nil
}

func (m *Model) nextTab() {
	m.tab = (m.tab + 1) % Tab(len(tabTitles))
	m.resetListFocus()
}

func (m *Model) prevTab() {
	m.tab = (m.tab - 1 + Tab(len(tabTitles))) % Tab(len(tabTitles))
	m.resetListFocus()
}

func (m *Model) resetListFocus() {
	m.headerFocused = true
	m.cursor = 0
	m.scroll = 0
}

func (m *Model) moveDown() {
	list := m.activeList()
	if list == nil {
		return
	}
	if m.headerFocused {
		if len(list) == 0 {
			return
		}
		m.headerFocused = false
		m.cursor = 0
		m.ensureVisible()
		return
	}
	if m.cursor < len(list)-1 {
		m.cursor++
		m.ensureVisible()
	}
}

func (m *Model) moveUp() {
	list := m.activeList()
	if list == nil || m.headerFocused {
		return
	}
	if m.cursor == 0 {
		m.headerFocused = true
		return
	}
	m.cursor--
	m.ensureVisible()
}

// activeList returns the command list backing the current tab, or nil for
// the General tab.
func (m *Model) activeList() []CommandInfo {
	switch m.tab {
	case TabCommands:
		return m.builtins
	case TabCustomCommands:
		return m.customs
	default:
		return nil
	}
}

// visibleCount returns how many list rows fit, mirroring HelpV2.tsx's
// maxHeight = floor(rows/2); Commands.tsx's visibleCount = max(1,
// floor((maxHeight-10)/2)).
func (m *Model) visibleCount() int {
	maxHeight := m.height / 2
	n := (maxHeight - 10) / 2
	if n < 1 {
		n = 1
	}
	return n
}

func (m *Model) ensureVisible() {
	vc := m.visibleCount()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+vc {
		m.scroll = m.cursor - vc + 1
	}
}

func (m Model) View() string {
	t := theme.Current()
	var sb strings.Builder

	sb.WriteString(m.renderHeader(t))
	sb.WriteString("\n\n")

	switch m.tab {
	case TabGeneral:
		sb.WriteString(m.viewGeneral(t))
	case TabCommands:
		sb.WriteString(m.viewCommandList(t, m.builtins, "Browse default commands", ""))
	case TabCustomCommands:
		sb.WriteString(m.viewCommandList(t, m.customs, "Browse custom commands", "No custom commands found"))
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("For more help: %s\n", docsURL))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Italic(true).Render("Esc to cancel"))
	return sb.String()
}

// renderHeader renders "Help  General   Commands   Custom commands" with
// the active tab shown as a blue block, matching design-system/Tabs.tsx
// (Tabs color="professionalBlue" + inverseText).
func (m Model) renderHeader(t theme.Theme) string {
	cs := t.Colors()
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(cs.Primary))
	activeStyle := lipgloss.NewStyle().Bold(true).
		Background(lipgloss.Color(cs.ProfessionalBlue)).
		Foreground(lipgloss.Color(cs.TextInverse))

	parts := []string{titleStyle.Render("Help")}
	for i, title := range tabTitles {
		s := lipgloss.NewStyle()
		if Tab(i) == m.tab {
			s = activeStyle
		}
		parts = append(parts, s.Render(" "+title+" "))
	}
	return strings.Join(parts, " ")
}

// viewGeneral renders the intro blurb and a shortcut grid. Only shortcuts
// gopher actually implements are listed.
func (m Model) viewGeneral(t theme.Theme) string {
	dim := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)

	var sb strings.Builder
	sb.WriteString("Gopher understands your codebase, makes edits with your permission, and executes commands — right from your terminal.\n\n")
	sb.WriteString(bold.Render("Shortcuts"))
	sb.WriteString("\n")

	col1 := []string{
		"/ for commands",
		"@ for file paths",
		"/btw for side question",
	}
	col2 := []string{
		"ctrl + t to toggle tasks",
		"/keybindings to customize",
	}

	col1Style := dim.Width(24)
	col2Style := dim
	rows := len(col1)
	if len(col2) > rows {
		rows = len(col2)
	}
	var lines []string
	for i := 0; i < rows; i++ {
		c1 := ""
		if i < len(col1) {
			c1 = col1[i]
		}
		c2 := ""
		if i < len(col2) {
			c2 = col2[i]
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, col1Style.Render(c1), col2Style.Render(c2)))
	}
	sb.WriteString(strings.Join(lines, "\n"))
	return sb.String()
}

// viewCommandList renders a scrollable list of commands: gutter (cursor /
// scroll indicator), "/name", and a dimmed description on the next line.
func (m Model) viewCommandList(t theme.Theme, list []CommandInfo, title, emptyMessage string) string {
	if len(list) == 0 && emptyMessage != "" {
		return lipgloss.NewStyle().Faint(true).Render(emptyMessage)
	}

	cs := t.Colors()
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(cs.TextPrimary))
	focusedNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(cs.Secondary))
	descStyle := lipgloss.NewStyle().Faint(true)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cs.Secondary))
	arrowStyle := lipgloss.NewStyle().Faint(true)

	vc := m.visibleCount()
	start := m.scroll
	if start > len(list) {
		start = len(list)
	}
	end := start + vc
	if end > len(list) {
		end = len(list)
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n\n")

	for i := start; i < end; i++ {
		cmd := list[i]
		focused := !m.headerFocused && i == m.cursor
		gutter := "  "
		if focused {
			gutter = cursorStyle.Render("❯") + " "
		} else if i == start && start > 0 {
			gutter = arrowStyle.Render("↑") + " "
		} else if i == end-1 && end < len(list) {
			gutter = arrowStyle.Render("↓") + " "
		}
		name := nameStyle
		if focused {
			name = focusedNameStyle
		}
		sb.WriteString(gutter + name.Render("/"+cmd.Name) + "\n")
		sb.WriteString("    " + descStyle.Render(FormatDescriptionWithSource(cmd)) + "\n")
	}

	return sb.String()
}
