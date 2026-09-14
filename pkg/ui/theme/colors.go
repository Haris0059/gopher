package theme

// colors.go — ColorScheme defines the complete set of semantic color roles
// that a theme must provide. Components reference these roles rather than
// hard-coded palette values, so themes can be swapped at runtime.

// ColorScheme holds every color role that components may reference.
// All values are lipgloss-compatible color strings (hex "#rrggbb" or
// ANSI "123").
//
// Brand usage contract:
//   - Primary (#6ad4e1, brand aqua) is the default brand voice: logo, mascot,
//     welcome banner, focused borders, spinner, cursor, prompt char, tool
//     names, active tab, dialog titles.
//   - Secondary (#b1b9f9, brand purple) marks the selected/focused row in the
//     "/" command autocomplete, the "@" file autocomplete, and the /help
//     command list. Nothing else.
//   - TextPrimary is pure white body text.
type ColorScheme struct {
	// --- Surfaces -----------------------------------------------------------

	// Background is the root terminal background.
	Background string
	// Surface is the default panel / card background.
	Surface string
	// SurfaceElevated is a raised surface (modals, dropdowns, floating panels).
	SurfaceElevated string
	// SurfaceOverlay is for overlays that sit on top of everything (dialogs).
	SurfaceOverlay string

	// --- Text ---------------------------------------------------------------

	// TextPrimary is the main body text color.
	TextPrimary string
	// TextSecondary is de-emphasized text (descriptions, timestamps).
	TextSecondary string
	// TextMuted is very low-contrast text (placeholders, disabled hints).
	TextMuted string
	// TextInverse is text on a colored/accent background.
	TextInverse string

	// --- Borders & Dividers -------------------------------------------------

	// Border is the default border color.
	Border string
	// BorderFocused is the border color for focused/active elements.
	BorderFocused string
	// BorderSubtle is for subtle internal dividers.
	BorderSubtle string

	// --- Primary action (brand aqua) -----------------------------------------

	// Primary is the brand color: logo, mascot, borders, spinner, cursor,
	// prompt char, tool names, active tab, dialog titles.
	Primary string
	// PrimaryMuted is a low-contrast version for backgrounds/badges.
	PrimaryMuted string

	// --- Secondary (brand purple) — selected/focused list rows --------------

	// Secondary is the text color for a focused/highlighted list item (e.g.
	// the selected row in the "/" and "@" autocompletes, and the /help
	// command list). Source: utils/theme.ts — `suggestion`.
	Secondary string
	// SecondaryMuted is a low-contrast secondary for subtle highlights.
	SecondaryMuted string
	// ProfessionalBlue is the fixed accent used by the HelpV2 tab bar and
	// pane border — the same hex in every non-ANSI reference theme.
	// Source: utils/theme.ts — `professionalBlue`.
	ProfessionalBlue string

	// --- Semantic status colors ---------------------------------------------

	// Success is for positive outcomes (pass, created, completed).
	Success string
	// SuccessMuted is a dim success for backgrounds.
	SuccessMuted string

	// Warning is for caution states (pending, approaching limit).
	Warning string
	// WarningMuted is a dim warning for backgrounds.
	WarningMuted string

	// Error is for failures, destructive actions, critical.
	Error string
	// ErrorMuted is a dim error for backgrounds.
	ErrorMuted string

	// Info is for informational messages, hints, links.
	Info string
	// InfoMuted is a dim info for backgrounds.
	InfoMuted string

	// --- Diff colors --------------------------------------------------------

	// DiffAdded is for added lines in diffs.
	DiffAdded string
	// DiffRemoved is for removed lines in diffs.
	DiffRemoved string
	// DiffContext is for unchanged context lines.
	DiffContext string

	// --- Spinner / progress -------------------------------------------------

	// Spinner is the spinner animation color.
	Spinner string

	// --- Selection / cursor -------------------------------------------------

	// Cursor is the text cursor / caret color.
	Cursor string
	// Selection is the background of selected text.
	Selection string

	// --- Component-specific roles -------------------------------------------

	// ToolName is the color for tool names in streaming output.
	ToolName string
	// ToolBorder is the border color for tool call boxes.
	ToolBorder string
	// Prompt is the color of the input prompt character (">").
	Prompt string
	// StatusBarBg is the status bar background.
	StatusBarBg string
	// StatusBarFg is the status bar foreground text.
	StatusBarFg string
	// TabActive is the active tab indicator color.
	TabActive string
	// TabInactive is the inactive tab color.
	TabInactive string
}
