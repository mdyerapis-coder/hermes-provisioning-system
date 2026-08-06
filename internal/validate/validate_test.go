package validate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryPassesForExpectedLayout(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"bootstrap", "fedora", "kickstarts", "recovery", "scripts"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	result := Repository(root)
	if !result.OK() {
		t.Fatalf("expected repository validation to pass: %#v", result.Checks)
	}
}

func TestRepositoryFailsWhenDirectoryIsMissing(t *testing.T) {
	root := t.TempDir()
	result := Repository(root)
	if result.OK() {
		t.Fatal("expected repository validation to fail")
	}
}

func TestEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	check := Endpoint(context.Background(), server.URL)
	if !check.OK {
		t.Fatalf("expected endpoint check to pass: %#v", check)
	}
}
