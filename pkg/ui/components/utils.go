package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// UI characters matching Gopher's visual language.
const (
	// PromptPrefix is the "❯" (U+276F) character used for input prompts and user messages.
	// Gopher uses figures.pointer from the npm figures package.
	PromptPrefix = "❯ "

	// ResponseConnector is the "⎿" (U+23BF) character for tool results/responses.
	// Claude uses ⎿ (DENTISTRY SYMBOL LIGHT DOWN AND HORIZONTAL), not └ (U+2514).
	// Source: GrepTool/UI.tsx:66 — <Text dimColor={true}>  ⎿  </Text>
	ResponseConnector = "  ⎿  "

	// ResponseContinuation is the indent for continuation lines under a connector.
	ResponseContinuation = "    "

	// DividerChar is the light horizontal line (U+2500) for section dividers.
	// Gopher uses ─ (light), not ━ (heavy).
	DividerChar = "─"
)

// FuzzyMatch returns true if needle is a subsequence of haystack.
func FuzzyMatch(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	ni := 0
	for _, c := range haystack {
		if c == rune(needle[ni]) {
			ni++
			if ni == len(needle) {
				return true
			}
		}
	}
	return false
}

// FuzzyMatchMask reports, for each rune of haystack, whether it was consumed
// by the same greedy left-to-right subsequence scan FuzzyMatch uses to test
// a match. On a literal prefix (needle == haystack[:len(needle)]) this masks
// exactly that prefix; on a non-contiguous fuzzy match it marks each matched
// character individually. Case-insensitive.
func FuzzyMatchMask(needle, haystack string) []bool {
	runes := []rune(haystack)
	mask := make([]bool, len(runes))
	if needle == "" {
		return mask
	}
	needleRunes := []rune(strings.ToLower(needle))
	lowerRunes := []rune(strings.ToLower(haystack))
	ni := 0
	for i, c := range lowerRunes {
		if ni < len(needleRunes) && c == needleRunes[ni] {
			mask[i] = true
			ni++
		}
	}
	return mask
}

// HighlightMatched renders an autocomplete suggestion's name so the
// characters the user has typed so far (needle) stand out: runs matched by
// FuzzyMatchMask are styled with `matched` (bold, typically bright white),
// the rest with `unmatched` (typically dim/gray). Contiguous runs of the
// same state are rendered as a single styled segment.
func HighlightMatched(name, needle string, matched, unmatched lipgloss.Style) string {
	mask := FuzzyMatchMask(needle, name)
	runes := []rune(name)
	var b strings.Builder
	start := 0
	cur := false
	flush := func(end int) {
		if end <= start {
			return
		}
		seg := string(runes[start:end])
		if cur {
			b.WriteString(matched.Render(seg))
		} else {
			b.WriteString(unmatched.Render(seg))
		}
	}
	for i := range runes {
		m := i < len(mask) && mask[i]
		if i == 0 {
			cur = m
		} else if m != cur {
			flush(i)
			start = i
			cur = m
		}
	}
	flush(len(runes))
	return b.String()
}

// truncateField truncates a string to maxLen, appending "…" if truncated.
func truncateField(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// truncateLines truncates multi-line content to maxLines.
func truncateLines(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

// WrapText wraps text to fit within width using simple word-wrap.
// Exported for callers outside this package (e.g. streaming display) that
// need the same wrapping behavior used by the message bubble renderer.
func WrapText(text string, width int) string {
	return wrapText(text, width)
}

// wrapText wraps text to fit within width using simple word-wrap.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= width {
			lines = append(lines, line)
			continue
		}
		words := strings.Fields(line)
		current := ""
		for _, word := range words {
			if current == "" {
				current = word
			} else if len(current)+1+len(word) <= width {
				current += " " + word
			} else {
				lines = append(lines, current)
				current = word
			}
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return strings.Join(lines, "\n")
}
