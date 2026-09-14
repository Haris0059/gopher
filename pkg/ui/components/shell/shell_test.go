package shell

import (
	"strings"
	"testing"

	"github.com/Haris0059/gopher/pkg/text"
)

func TestTryFormatJSON(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "valid object",
			line: `{"a":1,"b":2}`,
			want: "{\n  \"a\": 1,\n  \"b\": 2\n}",
		},
		{
			name: "not json",
			line: "just some text",
			want: "just some text",
		},
		{
			name: "empty string",
			line: "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tryFormatJSON(tt.line); got != tt.want {
				t.Errorf("tryFormatJSON(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestTryFormatJSON_LargeIntegersKeepPrecision(t *testing.T) {
	// The TS reference falls back to the original, unformatted line when a
	// large integer would lose precision round-tripping through JS's
	// float64-based JSON.parse. Go's json.Number is string-backed and never
	// loses integer precision, so the round-trip guard never fires here —
	// the number is formatted (correctly) instead of bailing out.
	line := `{"id":9223372036854775807}`
	want := "{\n  \"id\": 9223372036854775807\n}"
	if got := tryFormatJSON(line); got != want {
		t.Errorf("tryFormatJSON(%q) = %q, want %q (exact precision preserved)", line, got, want)
	}
}

func TestTryJSONFormatContent_TooLargeSkipsFormatting(t *testing.T) {
	huge := `{"a":1}` + strings.Repeat(" ", maxJSONFormatLength)
	if got := tryJSONFormatContent(huge); got != huge {
		t.Error("content over maxJSONFormatLength should be returned unchanged")
	}
}

func TestLinkifyURLsInText(t *testing.T) {
	in := "see https://example.com/path for details"
	got := linkifyURLsInText(in)
	if !strings.Contains(got, "https://example.com/path") {
		t.Errorf("linkifyURLsInText(%q) = %q, should still contain the URL text", in, got)
	}
	if got == in {
		t.Error("linkifyURLsInText should wrap the URL in a hyperlink escape sequence")
	}
}

func TestStripUnderlineAnsi(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain underline on", "\x1b[4mhello\x1b[0m", "hello\x1b[0m"},
		{"combined bold+underline", "\x1b[1;4mhi", "hi"},
		{"non-underline codes survive", "\x1b[1mbold\x1b[0m", "\x1b[1mbold\x1b[0m"},
		{"no ansi", "plain text", "plain text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripUnderlineAnsi(tt.in); got != tt.want {
				t.Errorf("stripUnderlineAnsi(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestOutputLine(t *testing.T) {
	got := OutputLine("hello world", 80, true, false, false, false, nil)
	if !strings.Contains(got, "hello world") {
		t.Errorf("OutputLine should contain the content, got %q", got)
	}
}

func TestOutputLine_TruncatesWhenNotShowingFull(t *testing.T) {
	long := strings.Repeat("x", 200)
	got := OutputLine(long, 20, false, false, false, false, nil)
	if text.DisplayWidth(text.StripANSI(got)) > 20 {
		t.Errorf("OutputLine with showFull=false should truncate to the column width, got width %d", text.DisplayWidth(text.StripANSI(got)))
	}
}

func TestShellTimeDisplay(t *testing.T) {
	sec := func(n int) *int { return &n }

	tests := []struct {
		name      string
		elapsed   *int
		timeoutMs *int
		want      string
	}{
		{"nothing set", nil, nil, ""},
		{"elapsed only", sec(5), nil, "(5s)"},
		{"elapsed over a minute", sec(90), nil, "(1m 30s)"},
		{"timeout only", nil, sec(120000), "(timeout 2m)"},
		{"elapsed and timeout", sec(5), sec(120000), "(5s · timeout 2m)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShellTimeDisplay(tt.elapsed, tt.timeoutMs); got != tt.want {
				t.Errorf("ShellTimeDisplay(%v, %v) = %q, want %q", tt.elapsed, tt.timeoutMs, got, tt.want)
			}
		})
	}
}

func TestShellProgressMessage_EmptyShowsRunning(t *testing.T) {
	got := ShellProgressMessage(ShellProgressOptions{})
	if !strings.Contains(got, "Running…") {
		t.Errorf("ShellProgressMessage with no output should show a running indicator, got %q", got)
	}
}

func TestShellProgressMessage_ShowsLastFiveLines(t *testing.T) {
	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, "line"+string(rune('0'+i%10)))
	}
	output := strings.Join(lines, "\n")
	got := ShellProgressMessage(ShellProgressOptions{Output: output, TotalLines: 10})

	if strings.Contains(got, "line1\n") {
		t.Error("non-verbose progress message should not include lines beyond the last 5")
	}
	if !strings.Contains(got, "+5 lines") {
		t.Errorf("expected a '+5 lines' status for 10 total lines showing 5, got %q", got)
	}
}

func TestShellProgressMessage_VerboseShowsFullOutput(t *testing.T) {
	full := "line-a\nline-b\nline-c\nline-d\nline-e\nline-f"
	got := ShellProgressMessage(ShellProgressOptions{
		Output:     full,
		FullOutput: full,
		TotalLines: 6,
		Verbose:    true,
	})
	if !strings.Contains(got, "line-a") {
		t.Error("verbose mode should include the full output, including early lines")
	}
	if strings.Contains(got, "+") {
		t.Error("verbose mode should not print a truncation line-count status")
	}
}

func TestShellProgressMessage_TotalBytesStatus(t *testing.T) {
	output := "one\ntwo"
	got := ShellProgressMessage(ShellProgressOptions{Output: output, TotalLines: 2, TotalBytes: 2048})
	if !strings.Contains(got, "2KB") {
		t.Errorf("expected a formatted byte size in the status row, got %q", got)
	}
}

func TestTruncateOutput_NoTruncationNeeded(t *testing.T) {
	in := "a\nb\nc"
	if got := TruncateOutput(in, 10); got != in {
		t.Errorf("TruncateOutput should return input unchanged when under maxLines, got %q", got)
	}
}

func TestTruncateOutput_ReportsAccurateOmittedCount(t *testing.T) {
	// Regression: the original implementation kept 2*(maxLines/2) lines but
	// reported len(lines)-maxLines omitted, undercounting by one whenever
	// maxLines was odd (5 kept as 4, reported 5 omitted instead of 6).
	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, "L")
	}
	got := TruncateOutput(strings.Join(lines, "\n"), 5)

	kept := 0
	for _, part := range strings.Split(got, "\n") {
		if part == "L" {
			kept++
		}
	}
	if kept != 5 {
		t.Errorf("TruncateOutput(_, 5) kept %d lines, want 5", kept)
	}
	if !strings.Contains(got, "(5 lines omitted)") {
		t.Errorf("TruncateOutput(_, 5) over 10 lines should report 5 omitted (10-5), got %q", got)
	}
}

func TestTruncateOutput_MaxLinesOneKeepsContent(t *testing.T) {
	// Regression: maxLines=1 previously computed half=0, discarding both
	// the head and tail entirely and returning only the omission marker.
	in := strings.Join([]string{"first", "middle1", "middle2", "last"}, "\n")
	got := TruncateOutput(in, 1)
	if !strings.Contains(got, "first") {
		t.Errorf("TruncateOutput(_, 1) should keep at least the first line, got %q", got)
	}
	if !strings.Contains(got, "last") {
		t.Errorf("TruncateOutput(_, 1) should keep at least the last line, got %q", got)
	}
}

func TestTruncateOutput_NegativeMaxLinesDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TruncateOutput panicked on negative maxLines: %v", r)
		}
	}()
	in := "a\nb\nc"
	TruncateOutput(in, -1)
}

func TestTruncateOutput_ZeroMaxLines(t *testing.T) {
	in := "a\nb\nc"
	got := TruncateOutput(in, 0)
	if strings.Contains(got, "a") || strings.Contains(got, "c") {
		t.Errorf("TruncateOutput(_, 0) should keep no content lines, got %q", got)
	}
	if !strings.Contains(got, "lines omitted") {
		t.Errorf("TruncateOutput(_, 0) should still report an omission count, got %q", got)
	}
}
