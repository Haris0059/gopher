// T399: File/context suggestion integration for @-mention autocomplete.
// Source: useInputSuggestion.tsx — file path autocomplete triggered by @ prefix.
// Interaction model (Tab/Up/Down/Enter/Escape) ported from
// claude-code-main/src/hooks/useTypeahead.tsx (handleTab, handleEnter,
// handleAutocompleteNext/Previous, handleAutocompleteDismiss).
//
// This file wires pkg/ui/hooks.FileSuggester into the AppModel so it is
// reachable from main(). The FileSuggester provides file path completion
// when the user types @<partial> in the input pane.
package ui

import (
	"path/filepath"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/Haris0059/gopher/pkg/ui/components"
	"github.com/Haris0059/gopher/pkg/ui/hooks"
	"github.com/Haris0059/gopher/pkg/ui/theme"
)

// initFileSuggester creates the FileSuggester rooted at the given cwd and
// attaches it to the AppModel. Called from NewAppModel.
func (a *AppModel) initFileSuggester(cwd string) {
	a.fileSuggester = hooks.NewFileSuggester(cwd)
}

// refreshFileAutocomplete detects @-mention partial paths in the input buffer
// and generates file suggestions. Called after each key press alongside
// refreshSlashAutocomplete.
func (a *AppModel) refreshFileAutocomplete() {
	if a.fileSuggester == nil || a.input == nil {
		a.dismissFileSuggestions()
		return
	}
	text := a.input.Value()
	partial, _, ok := extractAtToken(text)
	if !ok {
		a.dismissFileSuggestions()
		return
	}
	items := a.fileSuggester.GenerateSuggestions(partial, true)
	a.fileSuggestions = items
	a.fileSuggestActive = len(items) > 0
	if a.fileSuggestSelected < 0 || a.fileSuggestSelected >= len(items) {
		a.fileSuggestSelected = 0
	}
}

// dismissFileSuggestions closes the @-mention popup without touching the
// input text. Source: useTypeahead.tsx clearSuggestions / handleAutocompleteDismiss.
func (a *AppModel) dismissFileSuggestions() {
	a.fileSuggestActive = false
	a.fileSuggestions = nil
	a.fileSuggestSelected = 0
}

// moveFileSuggestSelection moves the selected suggestion by delta, wrapping
// around both ends. Source: useTypeahead.tsx handleAutocompleteNext/Previous.
func (a *AppModel) moveFileSuggestSelection(delta int) {
	n := len(a.fileSuggestions)
	if n == 0 {
		return
	}
	a.fileSuggestSelected = ((a.fileSuggestSelected+delta)%n + n) % n
}

// acceptFileSuggestion applies the active @-mention suggestion to the input.
// When allowPrefixCommit is true (Tab), suggestions sharing a common prefix
// longer than the typed partial are completed up to that prefix and the list
// stays open for further narrowing; otherwise (or when no such prefix
// exists) the selected item is committed and the list closes — except for a
// directory, which re-triggers one level deeper instead of closing.
// Source: useTypeahead.tsx handleTab (prefix-commit branch) vs handleEnter
// (always commits the selection).
// Returns false if there was nothing to accept.
func (a *AppModel) acceptFileSuggestion(allowPrefixCommit bool) bool {
	if !a.fileSuggestActive || len(a.fileSuggestions) == 0 || a.input == nil {
		return false
	}
	text := a.input.Value()
	partial, startPos, ok := extractAtToken(text)
	if !ok {
		return false
	}

	if allowPrefixCommit {
		if prefix := hooks.FindLongestCommonPrefix(a.fileSuggestions); len(prefix) > len(partial) {
			a.spliceInput(prefix, text, partial, startPos)
			// Keep the popup open, refiltered against the now-longer partial,
			// so repeated Tabs can keep narrowing down to a unique match.
			a.refreshFileAutocomplete()
			return true
		}
	}

	if a.fileSuggestSelected < 0 || a.fileSuggestSelected >= len(a.fileSuggestions) {
		a.fileSuggestSelected = 0
	}
	chosen := a.fileSuggestions[a.fileSuggestSelected].DisplayText
	isDir := strings.HasSuffix(chosen, string(filepath.Separator))
	suffix := " "
	if isDir {
		suffix = "" // trailing separator is already part of chosen
	}
	a.spliceInput(chosen+suffix, text, partial, startPos)
	if isDir {
		// Drill down: re-query suggestions for the deeper partial instead of
		// closing the popup.
		a.refreshFileAutocomplete()
	} else {
		a.dismissFileSuggestions()
	}
	return true
}

// spliceInput applies hooks.ApplySuggestion and writes the result (with the
// cursor converted from a byte offset to the rune offset InputPane expects)
// into the input pane.
func (a *AppModel) spliceInput(replacement, text, partial string, startPos int) {
	newInput, cursorByte := hooks.ApplySuggestion(replacement, text, partial, startPos)
	if cursorByte > len(newInput) {
		cursorByte = len(newInput)
	}
	cursorRune := utf8.RuneCountInString(newInput[:cursorByte])
	a.input.SetValueWithCursor(newInput, cursorRune)
}

// extractAtPartial finds the last @-mention partial in text.
// Returns the partial path after @ and true if found.
// Source: useInputSuggestion.tsx — regex /@([^\s@]*)$/
func extractAtPartial(text string) (string, bool) {
	partial, _, ok := extractAtToken(text)
	return partial, ok
}

// extractAtToken is extractAtPartial plus the byte offset where the partial
// (i.e. the text right after "@") starts, needed to splice a completion back
// into the buffer.
func extractAtToken(text string) (partial string, startPos int, ok bool) {
	idx := strings.LastIndex(text, "@")
	if idx < 0 {
		return "", 0, false
	}
	// @ must be at start or preceded by whitespace.
	if idx > 0 && text[idx-1] != ' ' && text[idx-1] != '\t' {
		return "", 0, false
	}
	partial = text[idx+1:]
	// No spaces allowed in the partial.
	if strings.ContainsAny(partial, " \t") {
		return "", 0, false
	}
	return partial, idx + 1, true
}

// renderFileSuggestions returns the file suggestion dropdown lines, or empty
// string if no suggestions are active. The selected row is highlighted,
// matching the slash-command popup (components/slash_input.go). Unselected
// rows are muted gray with the typed @-mention partial picked out in bold
// white, same scheme as the slash-command popup.
func (a *AppModel) renderFileSuggestions() string {
	if !a.fileSuggestActive || len(a.fileSuggestions) == 0 {
		return ""
	}
	cs := theme.Current().Colors()
	matchedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cs.TextPrimary)).Bold(true)
	unmatchedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cs.TextMuted))
	selNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cs.Secondary)).Bold(true)

	var partial string
	if a.input != nil {
		partial, _ = extractAtPartial(a.input.Value())
	}

	lines := make([]string, len(a.fileSuggestions))
	for i, item := range a.fileSuggestions {
		if i == a.fileSuggestSelected {
			// Selected row is a single color, not per-letter highlighted —
			// matches the slash-command popup's selection treatment.
			lines[i] = "  " + selNameStyle.Render(item.DisplayText)
		} else {
			lines[i] = "  " + components.HighlightMatched(item.DisplayText, partial, matchedStyle, unmatchedStyle)
		}
	}
	return strings.Join(lines, "\n")
}

// FileSuggester returns the file suggester for testing and external access.
func (a *AppModel) FileSuggester() *hooks.FileSuggester {
	return a.fileSuggester
}

// FileSuggestionsActive reports whether file autocomplete is showing.
func (a *AppModel) FileSuggestionsActive() bool {
	return a.fileSuggestActive
}

// FileSuggestions returns the current file suggestion items.
func (a *AppModel) FileSuggestions() []hooks.SuggestionItem {
	return a.fileSuggestions
}

// FileSuggestSelected returns the index of the currently highlighted
// suggestion, for testing.
func (a *AppModel) FileSuggestSelected() int {
	return a.fileSuggestSelected
}
