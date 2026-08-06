package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mdyerapis-coder/hermes-provisioning-system/internal/androidapk"
)

func runAndroid(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printAndroidHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printAndroidHelp(stdout)
		return 0
	case "validate":
		return runAndroidValidate(args[1:], stdout, stderr)
	case "stage":
		return runAndroidStage(args[1:], stdout, stderr)
	case "channel":
		return runAndroidChannel(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown android command %q\n\n", args[0])
		printAndroidHelp(stderr)
		return 2
	}
}

func runAndroidValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("android validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	apk := fs.String("apk", "", "path to the APK file")
	digest := fs.String("sha256", "", "expected APK SHA-256 digest")
	packageName := fs.String("package", "", "expected Android application ID")
	versionCode := fs.String("version-code", "", "expected Android version code")
	versionName := fs.String("version-name", "", "expected Android version name")
	signer := fs.String("signer-sha256", "", "expected signing certificate SHA-256 fingerprint")
	aapt2 := fs.String("aapt2", os.Getenv("HPS_AAPT2"), "optional aapt2 executable path")
	apksigner := fs.String("apksigner", os.Getenv("HPS_APKSIGNER"), "optional apksigner executable path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	for name, value := range map[string]string{
		"--apk":           *apk,
		"--sha256":        *digest,
		"--package":       *packageName,
		"--version-code":  *versionCode,
		"--version-name":  *versionName,
		"--signer-sha256": *signer,
	} {
		if strings.TrimSpace(value) == "" {
			fmt.Fprintf(stderr, "%s is required\n", name)
			return 2
		}
	}

	result, err := androidapk.Validate(context.Background(), androidapk.Options{
		Path:         *apk,
		SHA256:       *digest,
		Package:      *packageName,
		VersionCode:  *versionCode,
		VersionName:  *versionName,
		SignerSHA256: *signer,
		AAPT2:        *aapt2,
		APKSigner:    *apksigner,
	})
	if err != nil {
		fmt.Fprintf(stderr, "APK validation failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "apk: %s\n", result.Path)
	fmt.Fprintf(stdout, "size: %d\n", result.Size)
	fmt.Fprintf(stdout, "sha256: %s\n", result.SHA256)
	fmt.Fprintf(stdout, "package: %s\n", result.Package)
	fmt.Fprintf(stdout, "version-code: %s\n", result.VersionCode)
	fmt.Fprintf(stdout, "version-name: %s\n", result.VersionName)
	fmt.Fprintf(stdout, "signer-sha256: %s\n", result.SignerSHA256)
	fmt.Fprintf(stdout, "aapt2: %s\n", result.AAPT2)
	fmt.Fprintf(stdout, "apksigner: %s\n", result.APKSigner)
	fmt.Fprintln(stdout, "APK identity verified")
	return 0
}

func printAndroidHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  hps android validate \
    --apk FILE \
    --sha256 DIGEST \
    --package APPLICATION_ID \
    --version-code CODE \
    --version-name NAME \
    --signer-sha256 CERT_DIGEST
  hps android stage plan|apply [options]
  hps android channel plan|promote|show [options]

Commands:
  validate   Verify APK checksum, manifest identity, and signing certificate
  stage      Plan or apply immutable APK staging
  channel    Plan, promote, or inspect debug/beta/stable pointers

Tools:
  Android SDK Build Tools must provide aapt2 and apksigner. HPS discovers them
  from PATH, ANDROID_HOME, or ANDROID_SDK_ROOT. Explicit paths can be supplied
  with --aapt2 and --apksigner, or HPS_AAPT2 and HPS_APKSIGNER.

Safety:
  Validation and planning are read-only. Stage apply and channel promote require
  machine-readable approval bound to the SHA-256 of the exact plan file. HPS
  never builds, signs, or installs an APK and never handles a signing private key.
`)
}
