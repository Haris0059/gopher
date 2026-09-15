package query

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Haris0059/gopher/pkg/message"
	"github.com/Haris0059/gopher/pkg/session"
	"github.com/Haris0059/gopher/pkg/tools"
)

// MaxAtMentionDirEntries caps directory-listing output for an @-mentioned
// directory. Source: utils/attachments.ts:1922 (MAX_DIR_ENTRIES)
const MaxAtMentionDirEntries = 1000

// atMentionRe matches @-references in user input text: @path, @path:10,
// @path:10-20. Source: utils/attachments.ts extractAtMentionedFiles.
var atMentionRe = regexp.MustCompile(`@([\w./_\-]+(?::(\d+)(?:-(\d+))?)?)\b`)

type atMention struct {
	filePath  string
	lineStart int // 1-based; 0 means unset
	lineEnd   int // 1-based; 0 means unset
}

// extractAtMentions parses @file references from input text.
func extractAtMentions(input string) []atMention {
	matches := atMentionRe.FindAllStringSubmatch(input, -1)
	result := make([]atMention, 0, len(matches))
	for _, m := range matches {
		full := m[1]
		am := atMention{}
		if idx := strings.LastIndex(full, ":"); idx >= 0 {
			am.filePath = full[:idx]
			lineSpec := full[idx+1:]
			if i := strings.IndexByte(lineSpec, '-'); i >= 0 {
				am.lineStart = atoiOrZero(lineSpec[:i])
				am.lineEnd = atoiOrZero(lineSpec[i+1:])
			} else {
				am.lineStart = atoiOrZero(lineSpec)
			}
		} else {
			am.filePath = full
		}
		result = append(result, am)
	}
	return result
}

func atoiOrZero(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// expandAtMentions scans the last message in sess for @-mentioned files and
// appends a system-reminder attachment message with their contents, so the
// model receives file contents without having to call Read itself.
// Source: utils/attachments.ts:1894 (processAtMentionedFiles), :776 (getAttachments)
func expandAtMentions(sess *session.SessionState) {
	if len(sess.Messages) == 0 {
		return
	}
	last := sess.Messages[len(sess.Messages)-1]
	if last.Role != message.RoleUser {
		return
	}
	var text string
	for _, b := range last.Content {
		if b.Type == message.ContentText {
			text += b.Text
		}
	}
	if !strings.Contains(text, "@") {
		return
	}
	mentions := extractAtMentions(text)
	if len(mentions) == 0 {
		return
	}

	reader := &tools.FileReadTool{}
	tc := &tools.ToolContext{CWD: sess.CWD}
	seen := make(map[string]bool)

	for _, m := range mentions {
		path := m.filePath
		if strings.HasPrefix(path, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				path = filepath.Join(home, path[2:])
			}
		}
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(sess.CWD, absPath)
		}
		if seen[absPath] {
			continue
		}
		seen[absPath] = true

		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			content, ok := readDirListing(absPath)
			if !ok {
				continue
			}
			sess.PushMessage(message.NormalizeAttachment(message.Attachment{
				Type:    message.AttachmentFile,
				Content: content,
				Name:    absPath,
			}))
			continue
		}

		in := struct {
			FilePath string `json:"file_path"`
			Offset   int    `json:"offset"`
			Limit    int    `json:"limit"`
		}{FilePath: path}
		if m.lineStart > 0 {
			in.Offset = m.lineStart
			if m.lineEnd > 0 {
				in.Limit = m.lineEnd - m.lineStart + 1
			}
		}
		inputJSON, err := json.Marshal(in)
		if err != nil {
			continue
		}
		out, err := reader.Execute(context.Background(), tc, inputJSON)
		if err != nil || out == nil || out.IsError {
			continue
		}
		sess.PushMessage(message.NormalizeAttachment(message.Attachment{
			Type:    message.AttachmentFile,
			Content: out.Content,
			Name:    path,
		}))
	}
}

// readDirListing lists directory entries, capped at MaxAtMentionDirEntries.
// Source: utils/attachments.ts:1918-1936
func readDirListing(path string) (string, bool) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	truncated := len(names) > MaxAtMentionDirEntries
	if truncated {
		remaining := len(names) - MaxAtMentionDirEntries
		names = names[:MaxAtMentionDirEntries]
		names = append(names, "… and "+strconv.Itoa(remaining)+" more entries")
	}
	return strings.Join(names, "\n"), true
}
