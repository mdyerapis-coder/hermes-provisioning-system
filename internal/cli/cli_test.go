package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version"}, BuildInfo{
		Version: "1.2.3",
		Commit:  "abc123",
		Date:    "2026-08-06T00:00:00Z",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("unexpected exit code: %d; stderr=%q", code, stderr.String())
	}
	for _, expected := range []string{"hps 1.2.3", "abc123", "2026-08-06"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("output %q does not contain %q", stdout.String(), expected)
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"explode"}, BuildInfo{}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected usage error exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestHelpIsSafeAndExplicit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, BuildInfo{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code: %d", code)
	}
	if !strings.Contains(stdout.String(), "Disk partitioning") {
		t.Fatalf("help should state destructive actions are not implemented: %q", stdout.String())
	}
}
