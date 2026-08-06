package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileAcceptsValidManifest(t *testing.T) {
	path := writeManifest(t, `{
  "apiVersion": "hps.hermes/v1",
  "kind": "ProvisioningManifest",
  "metadata": {
    "name": "fedora44-kde",
    "version": "1"
  },
  "spec": {
    "os": {
      "distribution": "fedora",
      "release": "44",
      "architecture": "x86_64",
      "variant": "kde"
    },
    "assets": [
      {
        "name": "installer-iso",
        "type": "iso",
        "source": "https://example.invalid/fedora.iso",
        "destination": "fedora/44/fedora.iso",
        "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "required": true
      }
    ]
  }
}`)

	result, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if result.Metadata.Name != "fedora44-kde" || len(result.Spec.Assets) != 1 {
		t.Fatalf("unexpected manifest: %#v", result)
	}
}

func TestLoadFileRejectsUnknownFields(t *testing.T) {
	path := writeManifest(t, `{
  "apiVersion": "hps.hermes/v1",
  "kind": "ProvisioningManifest",
  "metadata": {"name": "fedora44-kde", "version": "1", "surprise": true},
  "spec": {
    "os": {"distribution": "fedora", "release": "44", "architecture": "x86_64"}
  }
}`)

	_, err := LoadFile(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field error, got %v", err)
	}
}

func TestValidateRejectsUnsafeDestinationsAndDuplicateAssets(t *testing.T) {
	manifest := Manifest{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata:   Metadata{Name: "fedora44-kde", Version: "1"},
		Spec: Spec{
			OS: OperatingSystem{Distribution: "fedora", Release: "44", Architecture: "x86_64"},
			Assets: []Asset{
				{
					Name:        "installer",
					Type:        "iso",
					Source:      "https://example.invalid/one.iso",
					Destination: "../outside.iso",
					SHA256:      strings.Repeat("a", 64),
				},
				{
					Name:        "installer",
					Type:        "iso",
					Source:      "file:///tmp/two.iso",
					Destination: "/absolute.iso",
					SHA256:      "not-a-digest",
				},
			},
		},
	}

	err := manifest.Validate()
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	for _, expected := range []string{"duplicated", "escapes", "relative", "scheme", "64 hexadecimal"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("validation error %q does not contain %q", err, expected)
		}
	}
}

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}
