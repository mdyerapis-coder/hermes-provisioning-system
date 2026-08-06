package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/mdyerapis-coder/hermes-provisioning-system/internal/assets"
)

func runAsset(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printAssetHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printAssetHelp(stdout)
		return 0
	case "verify":
		return runAssetVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown asset command %q\n\n", args[0])
		printAssetHelp(stderr)
		return 2
	}
}

func runAssetVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("asset verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("file", "", "path to the local asset")
	digest := fs.String("sha256", "", "expected SHA-256 digest")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*path) == "" || strings.TrimSpace(*digest) == "" {
		fmt.Fprintln(stderr, "--file and --sha256 are required")
		return 2
	}

	result, err := assets.VerifyFile(*path, *digest)
	if err != nil {
		fmt.Fprintf(stderr, "asset verification failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "file: %s\nsize: %d\nexpected: %s\nactual: %s\n", result.Path, result.Size, result.Expected, result.Actual)
	if !result.Match {
		fmt.Fprintln(stderr, "asset checksum mismatch")
		return 1
	}
	fmt.Fprintln(stdout, "asset checksum verified")
	return 0
}

func printAssetHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  hps asset verify --file FILE --sha256 DIGEST

Commands:
  verify   Calculate and verify the SHA-256 digest of a local file

Safety:
  Asset verification opens the specified file read-only and performs no network access.
`)
}
