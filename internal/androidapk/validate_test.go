package androidapk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testSigner = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestValidate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-specific")
	}
	directory := t.TempDir()
	apk := filepath.Join(directory, "hermes-debug.apk")
	payload := []byte("fixture APK bytes\n")
	if err := os.WriteFile(apk, payload, 0o600); err != nil {
		t.Fatalf("write APK fixture: %v", err)
	}
	digestBytes := sha256.Sum256(payload)
	digest := hex.EncodeToString(digestBytes[:])

	aapt2 := writeTool(t, directory, "aapt2", `#!/bin/sh
printf "%s\n" "package: name='com.hermesandroid.app.debug' versionCode='1' versionName='0.1.0-phase1' platformBuildVersionName=''"
`)
	apksigner := writeTool(t, directory, "apksigner", `#!/bin/sh
printf "%s\n" "Verifies"
printf "%s\n" "Signer #1 certificate SHA-256 digest: 01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef"
`)

	result, err := Validate(context.Background(), Options{
		Path:         apk,
		SHA256:       digest,
		Package:      "com.hermesandroid.app.debug",
		VersionCode:  "1",
		VersionName:  "0.1.0-phase1",
		SignerSHA256: testSigner,
		AAPT2:        aapt2,
		APKSigner:    apksigner,
	})
	if err != nil {
		t.Fatalf("validate APK: %v", err)
	}
	if result.Package != "com.hermesandroid.app.debug" || result.SignerSHA256 != testSigner || result.SHA256 != digest {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestValidateRejectsPackageMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-specific")
	}
	directory := t.TempDir()
	apk := filepath.Join(directory, "hermes.apk")
	payload := []byte("apk")
	if err := os.WriteFile(apk, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	aapt2 := writeTool(t, directory, "aapt2", `#!/bin/sh
printf "%s\n" "package: name='com.other.app' versionCode='1' versionName='0.1.0-phase1'"
`)
	apksigner := writeTool(t, directory, "apksigner", `#!/bin/sh
printf "%s\n" "Signer #1 certificate SHA-256 digest: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`)
	_, err := Validate(context.Background(), Options{
		Path:         apk,
		SHA256:       digest,
		Package:      "com.hermesandroid.app.debug",
		VersionCode:  "1",
		VersionName:  "0.1.0-phase1",
		SignerSHA256: testSigner,
		AAPT2:        aapt2,
		APKSigner:    apksigner,
	})
	if err == nil || !strings.Contains(err.Error(), "package mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseSignerRejectsMultipleIdentities(t *testing.T) {
	_, err := parseSignerSHA256("Signer #1 certificate SHA-256 digest: " + testSigner + "\nSigner #2 certificate SHA-256 digest: " + strings.Repeat("a", 64) + "\n")
	if err == nil || !strings.Contains(err.Error(), "distinct signer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeOptionsRejectsInvalidIdentity(t *testing.T) {
	_, err := normalizeOptions(Options{
		Path:         "x.apk",
		SHA256:       strings.Repeat("a", 64),
		Package:      "bad package",
		VersionCode:  "0",
		VersionName:  "",
		SignerSHA256: strings.Repeat("b", 64),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func writeTool(t *testing.T, directory, name, content string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write tool: %v", err)
	}
	return path
}
