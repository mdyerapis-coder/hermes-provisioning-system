package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAndroidValidateCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-specific")
	}

	directory := t.TempDir()
	apk := filepath.Join(directory, "app-debug.apk")
	payload := []byte("fixture APK\n")
	if err := os.WriteFile(apk, payload, 0o600); err != nil {
		t.Fatalf("write APK: %v", err)
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	signer := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	aapt2 := writeCLITool(t, directory, "aapt2", `#!/bin/sh
printf "%s\n" "package: name='com.hermesandroid.app.debug' versionCode='1' versionName='0.1.0-phase1'"
`)
	apksigner := writeCLITool(t, directory, "apksigner", `#!/bin/sh
printf "%s\n" "Signer #1 certificate SHA-256 digest: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"android", "validate",
		"--apk", apk,
		"--sha256", digest,
		"--package", "com.hermesandroid.app.debug",
		"--version-code", "1",
		"--version-name", "0.1.0-phase1",
		"--signer-sha256", signer,
		"--aapt2", aapt2,
		"--apksigner", apksigner,
	}, BuildInfo{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "APK identity verified") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestAndroidValidateRequiresIdentityFields(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"android", "validate", "--apk", "app.apk"}, BuildInfo{}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected usage exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func writeCLITool(t *testing.T, directory, name, content string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write tool: %v", err)
	}
	return path
}
