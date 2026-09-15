package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestOpenAICompatProvider_SkipsDeferredTools verifies that tools flagged
// DeferLoading never reach the wire for OpenAI-compatible providers, which
// (unlike Anthropic) have no server-side deferral — sending them all would
// blow past a local model's context window.
func TestOpenAICompatProvider_SkipsDeferredTools(t *testing.T) {
	var gotBody openAIChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// A minimal valid SSE stream so Stream() doesn't error reading it.
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewOpenAICompatProvider(server.URL, "test-key", "test-model")

	req := ModelRequest{
		Model:     "test-model",
		MaxTokens: 1024,
		Messages: []RequestMessage{
			{Role: "user", Content: []RequestContent{{Type: "text", Text: "hi"}}},
		},
		Tools: []ToolDefinition{
			{Name: "Read", Description: "reads a file", InputSchema: json.RawMessage(`{}`)},
			{Name: "TaskCreate", Description: "creates a task", InputSchema: json.RawMessage(`{}`), DeferLoading: true},
		},
	}

	ch, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	for range ch {
		// Drain until closed.
	}

	if len(gotBody.Tools) != 1 {
		t.Fatalf("got %d tools in request, want 1 (deferred tool should be skipped): %+v", len(gotBody.Tools), gotBody.Tools)
	}
	if gotBody.Tools[0].Function.Name != "Read" {
		t.Errorf("got tool %q, want Read", gotBody.Tools[0].Function.Name)
	}
}
