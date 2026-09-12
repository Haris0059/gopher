package handlers

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Haris0059/gopher/pkg/doctor"
)

func TestDoctor_HandlerExists(t *testing.T) {
	var buf bytes.Buffer
	code := Doctor(DoctorOpts{Output: &buf})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if out == "" {
		t.Error("doctor handler produced no output")
	}
}

func TestDoctor_ReturnsZero(t *testing.T) {
	var buf bytes.Buffer
	code := Doctor(DoctorOpts{Output: &buf})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

// TestDoctor_NonTTYPrintsSections verifies the handler falls back to plain
// text (rather than launching the TUI) when Output isn't a terminal, and
// that the section headers show up — a bytes.Buffer is never a *os.File so
// DefaultRunDoctor always takes this path in tests.
func TestDoctor_NonTTYPrintsSections(t *testing.T) {
	var buf bytes.Buffer
	code := Doctor(DoctorOpts{Output: &buf})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	out := buf.String()
	for _, want := range []string{"Diagnostics", "Updates", "Sandbox"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q section in output, got:\n%s", want, out)
		}
	}
}

func TestDoctor_VersionOverride(t *testing.T) {
	var buf bytes.Buffer
	code := Doctor(DoctorOpts{Output: &buf, Version: "9.9.9"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(buf.String(), "9.9.9") {
		t.Errorf("expected overridden version 9.9.9 in output, got:\n%s", buf.String())
	}
}

func TestDoctor_RunErrorReturnsOne(t *testing.T) {
	var buf bytes.Buffer
	code := Doctor(DoctorOpts{
		Output: &buf,
		Run: func(w io.Writer, data *doctor.DiagnosticData) error {
			return errors.New("boom")
		},
	})
	if code != 1 {
		t.Errorf("expected exit 1 on Run error, got %d", code)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Errorf("expected error message in output, got:\n%s", buf.String())
	}
}

func TestDoctor_CustomRunIsUsed(t *testing.T) {
	var buf bytes.Buffer
	called := false
	code := Doctor(DoctorOpts{
		Output: &buf,
		Run: func(w io.Writer, data *doctor.DiagnosticData) error {
			called = true
			if data == nil {
				t.Error("expected non-nil diagnostic data")
			}
			return nil
		},
	})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !called {
		t.Error("expected custom Run to be invoked")
	}
}
