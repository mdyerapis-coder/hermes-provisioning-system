package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerHealthStatusAndAssets(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hermes\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	handler, err := New(Config{
		Root: root,
		BuildInfo: BuildInfo{
			Version: "test",
			Commit:  "abc123",
			Date:    "2026-08-06T00:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}

	t.Run("health", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if recorder.Code != http.StatusOK || recorder.Body.String() != "ok\n" {
			t.Fatalf("unexpected health response: %d %q", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("status", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unexpected status code: %d", recorder.Code)
		}
		var payload struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if payload.Name != "Hermes Provisioning System" {
			t.Fatalf("unexpected name: %q", payload.Name)
		}
	})

	t.Run("asset", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/hello.txt", nil))
		if recorder.Code != http.StatusOK || recorder.Body.String() != "hermes\n" {
			t.Fatalf("unexpected asset response: %d %q", recorder.Code, recorder.Body.String())
		}
	})
}
