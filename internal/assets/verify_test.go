package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(path, []byte("hermes\n"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	const expected = "0bd98f000b9a1dbf63733dc3020959ec31018cb76d65ce2e15784b43f48d5eae"
	result, err := VerifyFile(path, expected)
	if err != nil {
		t.Fatalf("verify file: %v", err)
	}
	if !result.Match || result.Actual != expected || result.Size != 7 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestVerifyFileReportsMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(path, []byte("hermes\n"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	result, err := VerifyFile(path, strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("verify file: %v", err)
	}
	if result.Match {
		t.Fatalf("expected mismatch: %#v", result)
	}
}

func TestVerifyFileRejectsInvalidDigest(t *testing.T) {
	_, err := VerifyFile("unused", "not-a-digest")
	if err == nil {
		t.Fatal("expected digest validation error")
	}
}
