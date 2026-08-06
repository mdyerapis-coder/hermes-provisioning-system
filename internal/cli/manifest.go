package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/mdyerapis-coder/hermes-provisioning-system/internal/manifest"
)

func runManifest(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printManifestHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printManifestHelp(stdout)
		return 0
	case "validate":
		return runManifestValidate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown manifest command %q\n\n", args[0])
		printManifestHelp(stderr)
		return 2
	}
}

func runManifestValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("manifest validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("file", "", "path to a JSON provisioning manifest")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*path) == "" {
		fmt.Fprintln(stderr, "--file is required")
		return 2
	}

	loaded, err := manifest.LoadFile(*path)
	if err != nil {
		fmt.Fprintf(stderr, "manifest validation failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "manifest valid\nname: %s\nversion: %s\nos: %s %s %s\nassets: %d\n",
		loaded.Metadata.Name,
		loaded.Metadata.Version,
		loaded.Spec.OS.Distribution,
		loaded.Spec.OS.Release,
		loaded.Spec.OS.Architecture,
		len(loaded.Spec.Assets),
	)
	return 0
}

func printManifestHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  hps manifest validate --file MANIFEST.json

Commands:
  validate   Strictly decode and validate a provisioning manifest

Safety:
  Manifest validation performs no downloads, writes, installation, or execution.
`)
}
