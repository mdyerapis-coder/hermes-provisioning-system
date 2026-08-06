package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mdyerapis-coder/hermes-provisioning-system/internal/server"
	validation "github.com/mdyerapis-coder/hermes-provisioning-system/internal/validate"
)

// BuildInfo identifies the running binary.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// Run executes the CLI and returns a process exit code.
func Run(args []string, build BuildInfo, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	case "version", "--version":
		fmt.Fprintf(stdout, "hps %s\ncommit: %s\nbuilt: %s\n", build.Version, build.Commit, build.Date)
		return 0
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], build, stdout, stderr)
	case "serve":
		return runServe(args[1:], build, stdout, stderr)
	case "manifest":
		return runManifest(args[1:], stdout, stderr)
	case "asset":
		return runAsset(args[1:], stdout, stderr)
	case "android":
		return runAndroid(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repository := fs.String("repo", envOr("HPS_REPOSITORY", "/srv/hermes"), "provisioning repository root")
	endpoint := fs.String("server", os.Getenv("HPS_SERVER"), "optional provisioning server URL")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	result := validation.Repository(*repository)
	for _, check := range result.Checks {
		printCheck(stdout, check)
	}
	ok := result.OK()

	if strings.TrimSpace(*endpoint) != "" {
		check := validation.Endpoint(context.Background(), *endpoint)
		printCheck(stdout, check)
		ok = ok && check.OK
	}

	if !ok {
		fmt.Fprintln(stderr, "validation failed")
		return 1
	}
	fmt.Fprintln(stdout, "validation passed")
	return 0
}

func runDoctor(args []string, build BuildInfo, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repository := fs.String("repo", envOr("HPS_REPOSITORY", "/srv/hermes"), "provisioning repository root")
	endpoint := fs.String("server", os.Getenv("HPS_SERVER"), "optional provisioning server URL")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	fmt.Fprintf(stdout, "HPS doctor %s (%s/%s)\n", build.Version, runtime.GOOS, runtime.GOARCH)
	ok := true
	for _, command := range []string{"git", "curl"} {
		path, err := exec.LookPath(command)
		check := validation.Check{Name: "command-" + command, OK: err == nil, Detail: path}
		if err != nil {
			check.Detail = err.Error()
		}
		printCheck(stdout, check)
		ok = ok && check.OK
	}

	for _, command := range []string{"dnsmasq", "nginx"} {
		path, err := exec.LookPath(command)
		if err != nil {
			fmt.Fprintf(stdout, "WARN  command-%-12s %s\n", command, err)
			continue
		}
		fmt.Fprintf(stdout, "PASS  command-%-12s %s\n", command, path)
	}

	result := validation.Repository(*repository)
	for _, check := range result.Checks {
		printCheck(stdout, check)
		ok = ok && check.OK
	}
	if strings.TrimSpace(*endpoint) != "" {
		check := validation.Endpoint(context.Background(), *endpoint)
		printCheck(stdout, check)
		ok = ok && check.OK
	}

	if !ok {
		fmt.Fprintln(stderr, "doctor found blocking problems")
		return 1
	}
	fmt.Fprintln(stdout, "doctor passed")
	return 0
}

func runServe(args []string, build BuildInfo, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", envOr("HPS_REPOSITORY", "/srv/hermes"), "repository root to serve")
	listen := fs.String("listen", envOr("HPS_LISTEN", "127.0.0.1:8080"), "listen address")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	handler, err := server.New(server.Config{
		Root: *root,
		BuildInfo: server.BuildInfo{
			Version: build.Version,
			Commit:  build.Commit,
			Date:    build.Date,
		},
	})
	if err != nil {
		fmt.Fprintf(stderr, "configure server: %v\n", err)
		return 1
	}

	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	fmt.Fprintf(stdout, "serving %s on http://%s\n", *root, *listen)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "server stopped")
	return 0
}

func printCheck(w io.Writer, check validation.Check) {
	status := "FAIL"
	if check.OK {
		status = "PASS"
	}
	fmt.Fprintf(w, "%-5s %-24s %s\n", status, check.Name, check.Detail)
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `Hermes Provisioning System (HPS)

Usage:
  hps <command> [options]

Commands:
  version    Print build information
  validate   Validate repository structure and an optional HTTP endpoint
  doctor     Run host and provisioning readiness checks
  serve      Serve provisioning assets over HTTP
  manifest   Validate versioned provisioning manifests
  asset      Verify immutable local assets
  android    Validate Android APK identity and signing metadata
  help       Show this help

Environment:
  HPS_REPOSITORY  Repository root (default: /srv/hermes)
  HPS_SERVER      Provisioning endpoint checked by validate and doctor
  HPS_LISTEN      Embedded server address (default: 127.0.0.1:8080)
  HPS_AAPT2       Optional aapt2 executable path
  HPS_APKSIGNER   Optional apksigner executable path

Safety:
  The current commands are read-only except serve, which only opens an HTTP listener.
  Disk partitioning, installation, rebuild, recovery, APK staging, and APK installation
  are not implemented yet.
`)
}
