package androidapk

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mdyerapis-coder/hermes-provisioning-system/internal/assets"
)

const toolTimeout = 30 * time.Second

var (
	badgingAttributePattern = regexp.MustCompile(`([A-Za-z][A-Za-z0-9_-]*)='([^']*)'`)
	signerDigestPattern     = regexp.MustCompile(`(?im)^Signer #[0-9]+ certificate SHA-256 digest:\s*([0-9A-Fa-f: ]+)\s*$`)
	packageNamePattern      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
)

// Options defines the immutable identity expected from an APK.
type Options struct {
	Path         string
	SHA256       string
	Package      string
	VersionCode  string
	VersionName  string
	SignerSHA256 string
	AAPT2        string
	APKSigner    string
}

// Result records the APK identity verified by Android SDK Build Tools.
type Result struct {
	Path         string
	Size         int64
	SHA256       string
	Package      string
	VersionCode  string
	VersionName  string
	SignerSHA256 string
	AAPT2        string
	APKSigner    string
}

// Validate verifies an APK checksum, manifest identity, and signing certificate.
// It opens the APK read-only and never modifies or installs it.
func Validate(ctx context.Context, options Options) (Result, error) {
	normalized, err := normalizeOptions(options)
	if err != nil {
		return Result{}, err
	}

	info, err := os.Lstat(normalized.Path)
	if err != nil {
		return Result{}, fmt.Errorf("inspect APK: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Result{}, errors.New("APK path must not be a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return Result{}, errors.New("APK must be a regular file")
	}
	if !strings.EqualFold(filepath.Ext(normalized.Path), ".apk") {
		return Result{}, errors.New("APK file name must end in .apk")
	}

	verification, err := assets.VerifyFile(normalized.Path, normalized.SHA256)
	if err != nil {
		return Result{}, fmt.Errorf("verify APK checksum: %w", err)
	}
	if !verification.Match {
		return Result{}, fmt.Errorf("APK SHA-256 mismatch: expected %s, got %s", verification.Expected, verification.Actual)
	}

	aapt2, err := resolveTool(normalized.AAPT2, "aapt2")
	if err != nil {
		return Result{}, err
	}
	apksigner, err := resolveTool(normalized.APKSigner, "apksigner")
	if err != nil {
		return Result{}, err
	}

	badging, err := runTool(ctx, aapt2, "dump", "badging", normalized.Path)
	if err != nil {
		return Result{}, fmt.Errorf("read APK manifest metadata: %w", err)
	}
	metadata, err := parseBadging(string(badging))
	if err != nil {
		return Result{}, err
	}

	certificates, err := runTool(ctx, apksigner, "verify", "--print-certs", normalized.Path)
	if err != nil {
		return Result{}, fmt.Errorf("verify APK signature: %w", err)
	}
	signer, err := parseSignerSHA256(string(certificates))
	if err != nil {
		return Result{}, err
	}

	if metadata.Package != normalized.Package {
		return Result{}, fmt.Errorf("APK package mismatch: expected %q, got %q", normalized.Package, metadata.Package)
	}
	if metadata.VersionCode != normalized.VersionCode {
		return Result{}, fmt.Errorf("APK version code mismatch: expected %q, got %q", normalized.VersionCode, metadata.VersionCode)
	}
	if metadata.VersionName != normalized.VersionName {
		return Result{}, fmt.Errorf("APK version name mismatch: expected %q, got %q", normalized.VersionName, metadata.VersionName)
	}
	if signer != normalized.SignerSHA256 {
		return Result{}, fmt.Errorf("APK signer SHA-256 mismatch: expected %s, got %s", normalized.SignerSHA256, signer)
	}

	finalVerification, err := assets.VerifyFile(normalized.Path, normalized.SHA256)
	if err != nil {
		return Result{}, fmt.Errorf("reverify APK checksum: %w", err)
	}
	if !finalVerification.Match || finalVerification.Size != verification.Size {
		return Result{}, errors.New("APK changed during validation")
	}

	return Result{
		Path:         normalized.Path,
		Size:         finalVerification.Size,
		SHA256:       finalVerification.Actual,
		Package:      metadata.Package,
		VersionCode:  metadata.VersionCode,
		VersionName:  metadata.VersionName,
		SignerSHA256: signer,
		AAPT2:        aapt2,
		APKSigner:    apksigner,
	}, nil
}

type manifestMetadata struct {
	Package     string
	VersionCode string
	VersionName string
}

func normalizeOptions(options Options) (Options, error) {
	options.Path = strings.TrimSpace(options.Path)
	options.SHA256 = normalizeDigest(options.SHA256)
	options.Package = strings.TrimSpace(options.Package)
	options.VersionCode = strings.TrimSpace(options.VersionCode)
	options.VersionName = strings.TrimSpace(options.VersionName)
	options.SignerSHA256 = normalizeDigest(options.SignerSHA256)
	options.AAPT2 = strings.TrimSpace(options.AAPT2)
	options.APKSigner = strings.TrimSpace(options.APKSigner)

	if options.Path == "" {
		return Options{}, errors.New("APK path is required")
	}
	if !validDigest(options.SHA256) {
		return Options{}, errors.New("expected APK SHA-256 must contain exactly 64 hexadecimal characters")
	}
	if !packageNamePattern.MatchString(options.Package) {
		return Options{}, fmt.Errorf("invalid expected Android package name %q", options.Package)
	}
	code, err := strconv.ParseUint(options.VersionCode, 10, 64)
	if err != nil || code == 0 {
		return Options{}, errors.New("expected version code must be a positive decimal integer")
	}
	if options.VersionName == "" || strings.ContainsAny(options.VersionName, "\r\n") {
		return Options{}, errors.New("expected version name must be non-empty and single-line")
	}
	if !validDigest(options.SignerSHA256) {
		return Options{}, errors.New("expected signer SHA-256 must contain exactly 64 hexadecimal characters")
	}
	return options, nil
}

func resolveTool(configured, name string) (string, error) {
	if configured != "" {
		path, err := exec.LookPath(configured)
		if err != nil {
			return "", fmt.Errorf("locate configured %s tool %q: %w", name, configured, err)
		}
		return path, nil
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	var candidates []string
	for _, root := range []string{os.Getenv("ANDROID_HOME"), os.Getenv("ANDROID_SDK_ROOT")} {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(root, "build-tools", "*", name))
		candidates = append(candidates, matches...)
	}
	if len(candidates) > 0 {
		sort.Strings(candidates)
		for index := len(candidates) - 1; index >= 0; index-- {
			if info, err := os.Stat(candidates[index]); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
				return candidates[index], nil
			}
		}
	}
	return "", fmt.Errorf("required Android SDK Build Tools command %q was not found; add it to PATH or provide its path explicitly", name)
}

func runTool(parent context.Context, tool string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, toolTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, tool, args...)
	output, err := command.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("%s timed out after %s", filepath.Base(tool), toolTimeout)
	}
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if len(detail) > 4096 {
			detail = detail[:4096] + "..."
		}
		if detail == "" {
			return nil, fmt.Errorf("%s failed: %w", filepath.Base(tool), err)
		}
		return nil, fmt.Errorf("%s failed: %w: %s", filepath.Base(tool), err, detail)
	}
	return output, nil
}

func parseBadging(output string) (manifestMetadata, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "package:") {
			continue
		}
		attributes := make(map[string]string)
		for _, match := range badgingAttributePattern.FindAllStringSubmatch(line, -1) {
			attributes[match[1]] = match[2]
		}
		metadata := manifestMetadata{
			Package:     attributes["name"],
			VersionCode: attributes["versionCode"],
			VersionName: attributes["versionName"],
		}
		if metadata.Package == "" || metadata.VersionCode == "" || metadata.VersionName == "" {
			return manifestMetadata{}, errors.New("aapt2 badging output omitted package, versionCode, or versionName")
		}
		return metadata, nil
	}
	return manifestMetadata{}, errors.New("aapt2 badging output did not contain a package record")
}

func parseSignerSHA256(output string) (string, error) {
	matches := signerDigestPattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return "", errors.New("apksigner output did not contain a signer certificate SHA-256 digest")
	}

	unique := make(map[string]struct{})
	for _, match := range matches {
		digest := normalizeDigest(match[1])
		if !validDigest(digest) {
			return "", fmt.Errorf("apksigner returned an invalid certificate SHA-256 digest %q", match[1])
		}
		unique[digest] = struct{}{}
	}
	if len(unique) != 1 {
		return "", fmt.Errorf("APK contains %d distinct signer certificate identities; exactly one is required", len(unique))
	}
	for digest := range unique {
		return digest, nil
	}
	panic("unreachable")
}

func normalizeDigest(value string) string {
	replacer := strings.NewReplacer(":", "", " ", "", "\t", "", "\r", "", "\n", "")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(value)))
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}
