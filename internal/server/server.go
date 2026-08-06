package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// BuildInfo describes the running HPS build.
type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Config controls the embedded provisioning HTTP server.
type Config struct {
	Root      string
	BuildInfo BuildInfo
}

// New returns an HTTP handler for health, status, and repository assets.
func New(cfg Config) (http.Handler, error) {
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("open repository root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repository root is not a directory: %s", root)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /api/v1/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := struct {
			Name       string    `json:"name"`
			Repository string    `json:"repository"`
			BuildInfo  BuildInfo `json:"build"`
			Time       time.Time `json:"time"`
		}{
			Name:       "Hermes Provisioning System",
			Repository: root,
			BuildInfo:  cfg.BuildInfo,
			Time:       time.Now().UTC(),
		}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, "encode status response", http.StatusInternalServerError)
		}
	})
	mux.Handle("/", http.FileServer(http.Dir(root)))
	return securityHeaders(mux), nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
