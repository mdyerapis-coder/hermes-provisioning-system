package validate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Check is one validation result.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

// Result is the combined result of a validation run.
type Result struct {
	Checks []Check
}

// OK reports whether every check passed.
func (r Result) OK() bool {
	if len(r.Checks) == 0 {
		return false
	}
	for _, check := range r.Checks {
		if !check.OK {
			return false
		}
	}
	return true
}

// Repository validates the expected provisioning repository structure.
func Repository(root string) Result {
	checks := make([]Check, 0, 8)
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return Result{Checks: []Check{{Name: "repository-path", OK: false, Detail: err.Error()}}}
	}

	info, err := os.Stat(cleanRoot)
	if err != nil {
		checks = append(checks, Check{Name: "repository-root", OK: false, Detail: err.Error()})
		return Result{Checks: checks}
	}
	if !info.IsDir() {
		checks = append(checks, Check{Name: "repository-root", OK: false, Detail: cleanRoot + " is not a directory"})
		return Result{Checks: checks}
	}
	checks = append(checks, Check{Name: "repository-root", OK: true, Detail: cleanRoot})

	for _, name := range []string{"bootstrap", "fedora", "kickstarts", "recovery", "scripts"} {
		path := filepath.Join(cleanRoot, name)
		entry, statErr := os.Stat(path)
		if statErr != nil {
			checks = append(checks, Check{Name: "directory-" + name, OK: false, Detail: statErr.Error()})
			continue
		}
		if !entry.IsDir() {
			checks = append(checks, Check{Name: "directory-" + name, OK: false, Detail: path + " is not a directory"})
			continue
		}
		checks = append(checks, Check{Name: "directory-" + name, OK: true, Detail: path})
	}

	return Result{Checks: checks}
}

// Endpoint checks that an HTTP or HTTPS provisioning endpoint responds.
func Endpoint(ctx context.Context, rawURL string) Check {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return Check{Name: "server-endpoint", OK: false, Detail: err.Error()}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Check{Name: "server-endpoint", OK: false, Detail: "URL scheme must be http or https"}
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return Check{Name: "server-endpoint", OK: false, Detail: "URL host is missing"}
	}

	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	status, err := endpointRequest(requestCtx, http.MethodHead, rawURL)
	if err != nil {
		return Check{Name: "server-endpoint", OK: false, Detail: err.Error()}
	}
	if status >= 200 && status < 400 {
		return Check{Name: "server-endpoint", OK: true, Detail: fmt.Sprintf("HTTP %d (HEAD)", status)}
	}

	// Some directory indexes and hardened reverse proxies reject HEAD while a
	// normal GET succeeds. The response body is closed without being consumed,
	// so validation does not download an installer asset.
	if status == http.StatusForbidden || status == http.StatusMethodNotAllowed || status == http.StatusNotImplemented {
		getStatus, getErr := endpointRequest(requestCtx, http.MethodGet, rawURL)
		if getErr != nil {
			return Check{Name: "server-endpoint", OK: false, Detail: fmt.Sprintf("HEAD HTTP %d; GET failed: %v", status, getErr)}
		}
		if getStatus >= 200 && getStatus < 400 {
			return Check{Name: "server-endpoint", OK: true, Detail: fmt.Sprintf("HTTP %d (GET fallback after HEAD %d)", getStatus, status)}
		}
		return Check{Name: "server-endpoint", OK: false, Detail: fmt.Sprintf("HEAD HTTP %d; GET HTTP %d", status, getStatus)}
	}

	return Check{Name: "server-endpoint", OK: false, Detail: fmt.Sprintf("HTTP %d (HEAD)", status)}
}

func endpointRequest(ctx context.Context, method, rawURL string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
