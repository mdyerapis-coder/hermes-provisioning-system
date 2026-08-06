package validate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestEndpointFallsBackToBoundedGETWhenHEADIsForbidden(t *testing.T) {
	var receivedRange string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusForbidden)
		case http.MethodGet:
			receivedRange = r.Header.Get("Range")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("repository listing"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	check := Endpoint(context.Background(), server.URL)
	if !check.OK {
		t.Fatalf("expected GET fallback to pass: %#v", check)
	}
	if receivedRange != "bytes=0-0" {
		t.Fatalf("expected bounded GET range, got %q", receivedRange)
	}
	if !strings.Contains(check.Detail, "GET fallback") {
		t.Fatalf("expected fallback detail, got %q", check.Detail)
	}
}
