package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Haris0059/gopher/pkg/message"
	"github.com/Haris0059/gopher/pkg/session"
)

func TestExpandAtMentions_FileContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "name.txt")
	if err := os.WriteFile(path, []byte("hello from file"), 0o644); err != nil {
		t.Fatal(err)
	}

	sess := session.New(session.SessionConfig{}, dir)
	sess.PushMessage(message.UserMessage("please read @name.txt now"))

	expandAtMentions(sess)

	if len(sess.Messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(sess.Messages))
	}
	got := sess.Messages[1]
	if got.Role != message.RoleUser {
		t.Fatalf("attachment message role = %q, want user", got.Role)
	}
	text := got.Content[0].Text
	if !strings.Contains(text, "hello from file") {
		t.Errorf("attachment content = %q, want it to contain file contents", text)
	}
	if !strings.HasPrefix(text, message.SystemReminderPrefix) {
		t.Errorf("attachment content should be wrapped in a system-reminder, got %q", text)
	}
}

func TestExpandAtMentions_Directory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	sess := session.New(session.SessionConfig{}, dir)
	sess.PushMessage(message.UserMessage("look at @sub"))

	expandAtMentions(sess)

	if len(sess.Messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(sess.Messages))
	}
}

func TestExpandAtMentions_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	sess := session.New(session.SessionConfig{}, dir)
	sess.PushMessage(message.UserMessage("check @nope.txt please"))

	expandAtMentions(sess)

	if len(sess.Messages) != 1 {
		t.Fatalf("got %d messages, want 1 (no attachment for nonexistent file)", len(sess.Messages))
	}
}

func TestExpandAtMentions_NoMention(t *testing.T) {
	dir := t.TempDir()
	sess := session.New(session.SessionConfig{}, dir)
	sess.PushMessage(message.UserMessage("just a plain message"))

	expandAtMentions(sess)

	if len(sess.Messages) != 1 {
		t.Fatalf("got %d messages, want 1 (no @ in text)", len(sess.Messages))
	}
}

func TestExpandAtMentions_NotLastMessageIsUser(t *testing.T) {
	dir := t.TempDir()
	sess := session.New(session.SessionConfig{}, dir)
	sess.PushMessage(message.UserMessage("@name.txt"))
	sess.PushMessage(message.Message{Role: message.RoleAssistant, Content: []message.ContentBlock{{Type: message.ContentText, Text: "ok"}}})

	expandAtMentions(sess)

	if len(sess.Messages) != 2 {
		t.Fatalf("got %d messages, want 2 (no expansion when last message isn't from user)", len(sess.Messages))
	}
}

func TestExtractAtMentions_LineRange(t *testing.T) {
	got := extractAtMentions("see @pkg/foo.go:10-20 and @bar.go:5")
	if len(got) != 2 {
		t.Fatalf("got %d mentions, want 2", len(got))
	}
	if got[0].filePath != "pkg/foo.go" || got[0].lineStart != 10 || got[0].lineEnd != 20 {
		t.Errorf("mention[0] = %+v, want pkg/foo.go:10-20", got[0])
	}
	if got[1].filePath != "bar.go" || got[1].lineStart != 5 || got[1].lineEnd != 0 {
		t.Errorf("mention[1] = %+v, want bar.go:5", got[1])
	}
}
