package installer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveVersion_ExplicitPassthrough(t *testing.T) {
	got, err := ResolveVersion(context.Background(), nil, "", "1.2.3")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "1.2.3" {
		t.Errorf("got %s, want 1.2.3", got)
	}
}

func TestResolveVersion_ExplicitWithVPrefix(t *testing.T) {
	got, err := ResolveVersion(context.Background(), nil, "", "v1.2.3")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "1.2.3" {
		t.Errorf("got %s, want 1.2.3", got)
	}
}

func TestResolveVersion_InvalidChannel(t *testing.T) {
	_, err := ResolveVersion(context.Background(), nil, "", "nightly")
	if err == nil {
		t.Fatal("expected error for invalid channel")
	}
}

func TestResolveVersion_LatestFetchesFromAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/Haris0059/gopher/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(gitHubRelease{TagName: "v9.9.9"})
	}))
	defer srv.Close()

	for _, channel := range []string{"", "latest", "stable"} {
		got, err := ResolveVersion(context.Background(), srv.Client(), srv.URL, channel)
		if err != nil {
			t.Fatalf("ResolveVersion(%q): %v", channel, err)
		}
		if got != "9.9.9" {
			t.Errorf("ResolveVersion(%q) = %s, want 9.9.9", channel, got)
		}
	}
}

func TestResolveVersion_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := ResolveVersion(context.Background(), srv.Client(), srv.URL, "latest")
	if err == nil {
		t.Fatal("expected error for 404 status")
	}
}

func TestResolveVersion_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := ResolveVersion(context.Background(), srv.Client(), srv.URL, "latest")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestResolveVersion_EmptyTagName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gitHubRelease{TagName: ""})
	}))
	defer srv.Close()

	_, err := ResolveVersion(context.Background(), srv.Client(), srv.URL, "latest")
	if err == nil {
		t.Fatal("expected error for empty tag_name")
	}
}
