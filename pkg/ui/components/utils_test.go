package components

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// TestFuzzyMatchMask_ContiguousPrefix validates the common autocomplete case:
// typing "/he" against "/help" masks exactly the typed prefix.
func TestFuzzyMatchMask_ContiguousPrefix(t *testing.T) {
	mask := FuzzyMatchMask("/he", "/help")
	want := []bool{true, true, true, false, false}
	if len(mask) != len(want) {
		t.Fatalf("mask length = %d, want %d", len(mask), len(want))
	}
	for i := range want {
		if mask[i] != want[i] {
			t.Errorf("mask[%d] = %v, want %v", i, mask[i], want[i])
		}
	}
}

// TestFuzzyMatchMask_NonContiguous validates that a fuzzy (non-prefix) match
// marks each matched character individually rather than a single run.
func TestFuzzyMatchMask_NonContiguous(t *testing.T) {
	mask := FuzzyMatchMask("ab", "xaxb")
	want := []bool{false, true, false, true}
	for i := range want {
		if mask[i] != want[i] {
			t.Errorf("mask[%d] = %v, want %v", i, mask[i], want[i])
		}
	}
}

// TestFuzzyMatchMask_CaseInsensitive validates matching ignores case, as the
// autocomplete prefix and command names are compared case-insensitively.
func TestFuzzyMatchMask_CaseInsensitive(t *testing.T) {
	mask := FuzzyMatchMask("HE", "help")
	want := []bool{true, true, false, false}
	for i := range want {
		if mask[i] != want[i] {
			t.Errorf("mask[%d] = %v, want %v", i, mask[i], want[i])
		}
	}
}

// TestFuzzyMatchMask_EmptyNeedle validates an empty needle matches nothing
// (no highlight when nothing has been typed yet).
func TestFuzzyMatchMask_EmptyNeedle(t *testing.T) {
	mask := FuzzyMatchMask("", "help")
	for i, m := range mask {
		if m {
			t.Errorf("mask[%d] = true, want false for empty needle", i)
		}
	}
}

// TestHighlightMatched_PreservesText validates that styling the matched and
// unmatched runs never drops or reorders characters, regardless of style.
func TestHighlightMatched_PreservesText(t *testing.T) {
	matched := lipgloss.NewStyle().Bold(true)
	unmatched := lipgloss.NewStyle()

	cases := []struct{ needle, name string }{
		{"/he", "/help"},
		{"ab", "xaxb"},
		{"", "help"},
		{"nomatch", "help"},
	}
	for _, c := range cases {
		got := stripANSI(HighlightMatched(c.name, c.needle, matched, unmatched))
		if got != c.name {
			t.Errorf("HighlightMatched(%q, %q) stripped = %q, want %q", c.name, c.needle, got, c.name)
		}
	}
}
