package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestValidateCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	content := `{
  "apiVersion": "hps.hermes/v1",
  "kind": "ProvisioningManifest",
  "metadata": {"name": "fedora44-kde", "version": "1"},
  "spec": {
    "os": {"distribution": "fedora", "release": "44", "architecture": "x86_64"}
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"manifest", "validate", "--file", path}, BuildInfo{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "manifest valid") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestAssetVerifyCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(path, []byte("hermes\n"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"asset", "verify",
		"--file", path,
		"--sha256", "d53ba9ce30ffd743f4f905a61ddcf2c4fe0e5c72a2cc57638657fdd4171d1f6f",
	}, BuildInfo{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "asset checksum verified") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}
