// Package shell provides rendering helpers for shell/bash command output.
// Source: components/shell/ — OutputLine.tsx, ShellProgressMessage.tsx,
// ShellTimeDisplay.tsx, ExpandShellOutputContext.tsx
package shell

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Haris0059/gopher/pkg/text"
	"github.com/Haris0059/gopher/pkg/ui/hooks/display"
	"github.com/Haris0059/gopher/pkg/ui/termio"
	"github.com/Haris0059/gopher/pkg/ui/theme"
	"github.com/Haris0059/gopher/pkg/util"
)

// maxJSONFormatLength caps how large a line can be before we give up trying
// to pretty-print it as JSON.
// Source: ShellProgressMessage.tsx — MAX_JSON_FORMAT_LENGTH
const maxJSONFormatLength = 10_000

// urlInJSON matches http(s) URLs so they can be turned into terminal
// hyperlinks. Conservative: no quotes, no whitespace, no trailing comma/brace
// that would be JSON structure.
// Source: OutputLine.tsx — URL_IN_JSON
var urlInJSON = regexp.MustCompile(`https?://[^\s"'<>\\]+`)

// underlineAnsi matches the ANSI SGR sequences that turn underline on.
// Underline codes in particular tend to leak out of subprocess output for
// reasons that were never root-caused upstream; stripping *all* ANSI was
// tried and reverted because it lost legitimate formatting, so only
// underline is targeted.
// Source: OutputLine.tsx — stripUnderlineAnsi
var underlineAnsi = regexp.MustCompile(`\x1b\[([0-9]+;)*4(;[0-9]+)*m|\x1b\[4(;[0-9]+)*m|\x1b\[([0-9]+;)*4m`)

// tryFormatJSON attempts to pretty-print a single line of JSON. If the line
// doesn't parse, or if round-tripping it would lose numeric precision (large
// integers exceeding float64's safe integer range), the line is returned
// unchanged.
// Source: ShellProgressMessage.tsx — tryFormatJson
func tryFormatJSON(line string) string {
	dec := json.NewDecoder(strings.NewReader(line))
	dec.UseNumber()
	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return line
	}

	stringified, err := json.Marshal(parsed)
	if err != nil {
		return line
	}

	// Normalize both sides by stripping whitespace and the optional (but
	// valid) `\/` escape before comparing, so cosmetic differences don't
	// trigger a false precision-loss positive.
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, `\/`, "/")
		return strings.Join(strings.Fields(s), "")
	}
	if normalize(line) != normalize(string(stringified)) {
		return line
	}

	indented, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return line
	}
	return string(indented)
}

// tryJSONFormatContent runs tryFormatJSON over every line of content, unless
// the content is too large to bother.
// Source: ShellProgressMessage.tsx — tryJsonFormatContent
func tryJSONFormatContent(content string) string {
	if len(content) > maxJSONFormatLength {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = tryFormatJSON(line)
	}
	return strings.Join(lines, "\n")
}

// linkifyURLsInText wraps any http(s) URLs found in content in OSC 8
// terminal hyperlinks.
// Source: OutputLine.tsx — linkifyUrlsInText
func linkifyURLsInText(content string) string {
	return urlInJSON.ReplaceAllStringFunc(content, func(url string) string {
		return termio.Hyperlink(url, url)
	})
}

// stripUnderlineAnsi removes underline ANSI escape codes from content.
// Source: OutputLine.tsx — stripUnderlineAnsi
func stripUnderlineAnsi(content string) string {
	return underlineAnsi.ReplaceAllString(content, "")
}

// OutputLine renders a line (or block) of shell output: JSON-pretty-printing
// it, optionally linkifying URLs, truncating to the terminal width unless
// showFull is set, and stripping stray underline ANSI codes. th may be nil,
// in which case no color styling is applied.
// Source: components/shell/OutputLine.tsx
func OutputLine(content string, columns int, showFull, linkifyURLs, isError, isWarning bool, th theme.Theme) string {
	formatted := tryJSONFormatContent(content)
	if linkifyURLs {
		formatted = linkifyURLsInText(formatted)
	}
	if showFull {
		formatted = stripUnderlineAnsi(formatted)
	} else {
		formatted = stripUnderlineAnsi(text.TruncateAnsi(formatted, columns, text.TruncateEnd))
	}

	if th == nil {
		return formatted
	}
	switch {
	case isError:
		return th.TextError().Render(formatted)
	case isWarning:
		return th.TextWarning().Render(formatted)
	default:
		return formatted
	}
}

// ShellTimeDisplay renders the elapsed/timeout annotation shown alongside
// running or completed shell output: "(5s)", "(timeout 2m)", or
// "(5s · timeout 2m)". Returns "" when there is nothing to show.
// Source: components/shell/ShellTimeDisplay.tsx
func ShellTimeDisplay(elapsedSeconds *int, timeoutMs *int) string {
	var timeout string
	if timeoutMs != nil && *timeoutMs > 0 {
		timeout = display.FormatDuration(time.Duration(*timeoutMs) * time.Millisecond)
	}

	if elapsedSeconds == nil {
		if timeout == "" {
			return ""
		}
		return fmt.Sprintf("(timeout %s)", timeout)
	}

	elapsed := display.FormatDuration(time.Duration(*elapsedSeconds) * time.Second)
	if timeout != "" {
		return fmt.Sprintf("(%s · timeout %s)", elapsed, timeout)
	}
	return fmt.Sprintf("(%s)", elapsed)
}

// ShellProgressOptions carries the fields ShellProgressMessage needs to
// render a running or completed shell command's progress.
// Source: components/shell/ShellProgressMessage.tsx — Props
type ShellProgressOptions struct {
	Output             string
	FullOutput         string
	ElapsedTimeSeconds *int
	TotalLines         int
	TotalBytes         int
	TimeoutMs          *int
	Verbose            bool
}

// ShellProgressMessage renders a progress indicator for a running or
// completed shell command: the last few lines of output (or the full output
// in verbose mode), a line-count/byte-size status, and the elapsed/timeout
// annotation.
// Source: components/shell/ShellProgressMessage.tsx
func ShellProgressMessage(opts ShellProgressOptions) string {
	strippedFull := strings.TrimSpace(text.StripANSI(opts.FullOutput))
	strippedOutput := strings.TrimSpace(text.StripANSI(opts.Output))

	var lines []string
	for _, l := range strings.Split(strippedOutput, "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}

	timeDisplay := ShellTimeDisplay(opts.ElapsedTimeSeconds, opts.TimeoutMs)

	if len(lines) == 0 {
		if timeDisplay == "" {
			return "Running… "
		}
		return "Running…  " + timeDisplay // Source: two spaces before the status row, matching Box{gap:1} in the reference
	}

	var displayLines string
	if opts.Verbose {
		displayLines = strippedFull
	} else {
		tail := lines
		if len(tail) > 5 {
			tail = tail[len(tail)-5:]
		}
		displayLines = strings.Join(tail, "\n")
	}

	extraLines := 0
	if opts.TotalLines > 5 {
		extraLines = opts.TotalLines - 5
	}
	var lineStatus string
	switch {
	case !opts.Verbose && opts.TotalBytes > 0 && opts.TotalLines > 0:
		lineStatus = fmt.Sprintf("~%d lines", opts.TotalLines)
	case !opts.Verbose && extraLines > 0:
		lineStatus = fmt.Sprintf("+%d lines", extraLines)
	}

	statusParts := make([]string, 0, 3)
	if lineStatus != "" {
		statusParts = append(statusParts, lineStatus)
	}
	if timeDisplay != "" {
		statusParts = append(statusParts, timeDisplay)
	}
	if opts.TotalBytes > 0 {
		statusParts = append(statusParts, util.FormatFileSize(opts.TotalBytes))
	}

	result := displayLines
	if len(statusParts) > 0 {
		result += "\n" + strings.Join(statusParts, " ")
	}
	return result
}

// TruncateOutput truncates shell output to maxLines, keeping roughly equal
// halves from the start and end and reporting how many lines were dropped
// in between. Not present in the TS reference — a Gopher-native addition
// for displaying long tool output.
func TruncateOutput(output string, maxLines int) string {
	if maxLines < 0 {
		maxLines = 0
	}
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}

	// Keep at least one line at the head and tail when the caller asked for
	// any lines at all, so a tiny maxLines doesn't discard everything.
	headCount := maxLines / 2
	tailCount := maxLines - headCount
	if maxLines > 0 {
		if headCount == 0 {
			headCount = 1
		}
		if tailCount == 0 {
			tailCount = 1
		}
	}

	first := lines[:headCount]
	last := lines[len(lines)-tailCount:]
	omitted := len(lines) - headCount - tailCount

	return strings.Join(first, "\n") +
		fmt.Sprintf("\n... (%d lines omitted) ...\n", omitted) +
		strings.Join(last, "\n")
}
