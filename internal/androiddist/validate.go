package androiddist

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func validateStageManifest(manifest StageManifest) error {
	if manifest.APIVersion != APIVersion {
		return fmt.Errorf("stage manifest apiVersion must be %q", APIVersion)
	}
	if manifest.Kind != KindStageManifest {
		return fmt.Errorf("stage manifest kind must be %q", KindStageManifest)
	}
	if err := validateSegment("metadata.name", manifest.Metadata.Name); err != nil {
		return err
	}
	if err := validateSegment("metadata.version", manifest.Metadata.Version); err != nil {
		return err
	}
	if manifest.Metadata.Version != manifest.Spec.VersionName {
		return errors.New("metadata.version must exactly match spec.versionName")
	}
	if err := validateChannel(manifest.Spec.Channel); err != nil {
		return err
	}
	if strings.TrimSpace(manifest.Spec.APK) == "" {
		return errors.New("spec.apk is required")
	}
	if err := validateAPKFileName(manifest.Spec.FileName); err != nil {
		return err
	}
	if err := validatePackageAndVersion(manifest.Spec.Channel, manifest.Spec.Package, manifest.Spec.VersionCode, manifest.Spec.VersionName, manifest.Metadata.Version); err != nil {
		return err
	}
	if !validDigest(manifest.Spec.SHA256) {
		return errors.New("spec.sha256 must contain exactly 64 hexadecimal characters")
	}
	if !validDigest(manifest.Spec.SignerSHA256) {
		return errors.New("spec.signerSha256 must contain exactly 64 hexadecimal characters")
	}
	if strings.TrimSpace(manifest.Spec.Provenance.SourceRepository) == "" {
		return errors.New("spec.provenance.sourceRepository is required")
	}
	if strings.TrimSpace(manifest.Spec.Provenance.Commit) == "" {
		return errors.New("spec.provenance.commit is required")
	}
	return nil
}

func validateStagePlan(plan StagePlan) error {
	if plan.APIVersion != APIVersion || plan.Kind != KindStagePlan {
		return errors.New("invalid Android stage plan identity")
	}
	if _, err := time.Parse(time.RFC3339, plan.CreatedAt); err != nil {
		return errors.New("stage plan createdAt must be RFC3339")
	}
	if filepath.Clean(plan.Spec.Repository) != plan.Spec.Repository || !filepath.IsAbs(plan.Spec.Repository) {
		return errors.New("stage plan repository must be a clean absolute path")
	}
	if filepath.Clean(plan.Spec.SourceAPK) != plan.Spec.SourceAPK || !filepath.IsAbs(plan.Spec.SourceAPK) {
		return errors.New("stage plan sourceApk must be a clean absolute path")
	}
	if err := validateChannel(plan.Spec.Channel); err != nil {
		return err
	}
	if err := validateSegment("version", plan.Spec.Version); err != nil {
		return err
	}
	if err := validateAPKFileName(plan.Spec.FileName); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"releaseDirectory": plan.Spec.ReleaseDirectory,
		"apkDestination":   plan.Spec.APKDestination,
		"releaseManifest":  plan.Spec.ReleaseManifest,
		"checksums":        plan.Spec.Checksums,
	} {
		if err := validateRelativePath(value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	expectedDir := filepath.ToSlash(filepath.Join("android", plan.Spec.Channel, plan.Spec.Version))
	if plan.Spec.ReleaseDirectory != expectedDir ||
		plan.Spec.APKDestination != filepath.ToSlash(filepath.Join(expectedDir, plan.Spec.FileName)) ||
		plan.Spec.ReleaseManifest != filepath.ToSlash(filepath.Join(expectedDir, "release.json")) ||
		plan.Spec.Checksums != filepath.ToSlash(filepath.Join(expectedDir, "SHA256SUMS")) {
		return errors.New("stage plan destination paths are inconsistent")
	}
	if err := validatePackageAndVersion(plan.Spec.Channel, plan.Spec.Package, plan.Spec.VersionCode, plan.Spec.VersionName, plan.Spec.Version); err != nil {
		return err
	}
	if !validDigest(plan.Spec.SHA256) || !validDigest(plan.Spec.SignerSHA256) || plan.Spec.Size <= 0 {
		return errors.New("stage plan contains invalid APK digest or size metadata")
	}
	if strings.TrimSpace(plan.Spec.Provenance.SourceRepository) == "" || strings.TrimSpace(plan.Spec.Provenance.Commit) == "" {
		return errors.New("stage plan provenance is incomplete")
	}
	return nil
}

func validateRelease(release Release, channel, version string) error {
	if release.APIVersion != APIVersion || release.Kind != KindRelease {
		return errors.New("invalid Android release manifest identity")
	}
	if release.Metadata.Version != version || release.Spec.Channel != channel {
		return errors.New("release manifest does not match the selected channel and version")
	}
	if release.Metadata.Name != release.Spec.Package {
		return errors.New("release metadata.name must match spec.package")
	}
	if _, err := time.Parse(time.RFC3339, release.Metadata.CreatedAt); err != nil {
		return errors.New("release metadata.createdAt must be RFC3339")
	}
	if err := validateAPKFileName(release.Spec.File); err != nil {
		return err
	}
	if err := validatePackageAndVersion(channel, release.Spec.Package, release.Spec.VersionCode, release.Spec.VersionName, version); err != nil {
		return err
	}
	if !validDigest(release.Spec.SHA256) || !validDigest(release.Spec.SignerSHA256) || release.Spec.Size <= 0 {
		return errors.New("release manifest contains invalid APK digest or size metadata")
	}
	if !validDigest(release.Spec.StagePlanSHA256) || !validDigest(release.Spec.ApprovalSHA256) {
		return errors.New("release manifest contains invalid approval provenance digests")
	}
	if strings.TrimSpace(release.Spec.Provenance.SourceRepository) == "" || strings.TrimSpace(release.Spec.Provenance.Commit) == "" {
		return errors.New("release manifest provenance is incomplete")
	}
	return nil
}

func validateChannelPlan(plan ChannelPlan) error {
	if plan.APIVersion != APIVersion || plan.Kind != KindChannelPlan {
		return errors.New("invalid Android channel plan identity")
	}
	if _, err := time.Parse(time.RFC3339, plan.CreatedAt); err != nil {
		return errors.New("channel plan createdAt must be RFC3339")
	}
	if filepath.Clean(plan.Spec.Repository) != plan.Spec.Repository || !filepath.IsAbs(plan.Spec.Repository) {
		return errors.New("channel plan repository must be a clean absolute path")
	}
	if err := validateChannel(plan.Spec.Channel); err != nil {
		return err
	}
	if err := validateSegment("version", plan.Spec.Version); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"releaseManifest": plan.Spec.ReleaseManifest,
		"channelManifest": plan.Spec.ChannelManifest,
		"apk":             plan.Spec.APK,
	} {
		if err := validateRelativePath(value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if err := validateAPKFileName(filepath.Base(filepath.FromSlash(plan.Spec.APK))); err != nil {
		return err
	}
	expectedReleaseDir := filepath.ToSlash(filepath.Join("android", plan.Spec.Channel, plan.Spec.Version))
	if plan.Spec.ReleaseManifest != filepath.ToSlash(filepath.Join(expectedReleaseDir, "release.json")) ||
		plan.Spec.ChannelManifest != filepath.ToSlash(filepath.Join("android", plan.Spec.Channel, "current.json")) ||
		filepath.ToSlash(filepath.Dir(filepath.FromSlash(plan.Spec.APK))) != expectedReleaseDir {
		return errors.New("channel plan paths are inconsistent")
	}
	if err := validatePackageAndVersion(plan.Spec.Channel, plan.Spec.Package, plan.Spec.VersionCode, plan.Spec.VersionName, plan.Spec.Version); err != nil {
		return err
	}
	if !validDigest(plan.Spec.ReleaseSHA256) || !validDigest(plan.Spec.APKSHA256) || !validDigest(plan.Spec.SignerSHA256) {
		return errors.New("channel plan contains invalid SHA-256 metadata")
	}
	return nil
}

func validateChannelDocument(channelDoc Channel, expectedChannel string) error {
	if channelDoc.APIVersion != APIVersion || channelDoc.Kind != KindChannel {
		return errors.New("invalid Android channel manifest identity")
	}
	if channelDoc.Metadata.Name != expectedChannel {
		return errors.New("channel manifest name mismatch")
	}
	if _, err := time.Parse(time.RFC3339, channelDoc.Metadata.UpdatedAt); err != nil {
		return errors.New("channel metadata.updatedAt must be RFC3339")
	}
	if err := validateSegment("version", channelDoc.Spec.Version); err != nil {
		return err
	}
	if err := validateRelativePath(channelDoc.Spec.ReleaseManifest); err != nil {
		return err
	}
	if err := validateRelativePath(channelDoc.Spec.APK); err != nil {
		return err
	}
	if err := validateAPKFileName(filepath.Base(filepath.FromSlash(channelDoc.Spec.APK))); err != nil {
		return err
	}
	expectedReleaseDir := filepath.ToSlash(filepath.Join("android", expectedChannel, channelDoc.Spec.Version))
	if channelDoc.Spec.ReleaseManifest != filepath.ToSlash(filepath.Join(expectedReleaseDir, "release.json")) ||
		filepath.ToSlash(filepath.Dir(filepath.FromSlash(channelDoc.Spec.APK))) != expectedReleaseDir {
		return errors.New("channel manifest paths are inconsistent")
	}
	if err := validatePackageAndVersion(expectedChannel, channelDoc.Spec.Package, channelDoc.Spec.VersionCode, channelDoc.Spec.VersionName, channelDoc.Spec.Version); err != nil {
		return err
	}
	if !validDigest(channelDoc.Spec.ReleaseSHA256) || !validDigest(channelDoc.Spec.APKSHA256) || !validDigest(channelDoc.Spec.SignerSHA256) || !validDigest(channelDoc.Spec.PromotionPlanSHA256) || !validDigest(channelDoc.Spec.ApprovalSHA256) {
		return errors.New("channel manifest contains invalid SHA-256 metadata")
	}
	return nil
}

func validateApproval(approval Approval, expectedAction, planDigest string, now time.Time) error {
	if approval.APIVersion != APIVersion || approval.Kind != KindApproval {
		return errors.New("invalid approval document identity")
	}
	if approval.Action != expectedAction {
		return fmt.Errorf("approval action must be %q", expectedAction)
	}
	if !approval.Approved {
		return errors.New("approval document does not approve the action")
	}
	if normalizeDigest(approval.PlanSHA256) != planDigest {
		return errors.New("approval planSha256 does not match the exact plan file")
	}
	if strings.TrimSpace(approval.ApprovedBy) == "" {
		return errors.New("approval approvedBy is required")
	}
	approvedAt, err := time.Parse(time.RFC3339, approval.ApprovedAt)
	if err != nil {
		return errors.New("approval approvedAt must be RFC3339")
	}
	if approvedAt.After(now.Add(5 * time.Minute)) {
		return errors.New("approval approvedAt is unexpectedly in the future")
	}
	if strings.TrimSpace(approval.ExpiresAt) != "" {
		expiresAt, err := time.Parse(time.RFC3339, approval.ExpiresAt)
		if err != nil {
			return errors.New("approval expiresAt must be RFC3339")
		}
		if !expiresAt.After(now) {
			return errors.New("approval has expired")
		}
	}
	return nil
}

func validateChannel(channel string) error {
	if !channelPattern.MatchString(channel) {
		return fmt.Errorf("channel must be one of debug, beta, or stable")
	}
	return nil
}

func validatePackageAndVersion(channel, packageName, versionCode, versionName, version string) error {
	if !packageNamePattern.MatchString(packageName) {
		return fmt.Errorf("invalid Android package name %q", packageName)
	}
	if channel == "stable" && strings.HasSuffix(packageName, ".debug") {
		return errors.New("stable channel cannot contain a debug application ID")
	}
	if _, err := positiveVersionCode(versionCode); err != nil {
		return err
	}
	if versionName == "" || strings.ContainsAny(versionName, "\r\n") {
		return errors.New("versionName must be non-empty and single-line")
	}
	if versionName != version {
		return errors.New("version must exactly match versionName")
	}
	return nil
}

func validateSegment(name, value string) error {
	if !segmentPattern.MatchString(value) || value == "." || value == ".." || strings.Contains(value, "..") {
		return fmt.Errorf("%s must be a safe repository path segment", name)
	}
	return nil
}

func validateAPKFileName(name string) error {
	if err := validateSegment("fileName", name); err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Ext(name), ".apk") || filepath.Base(name) != name {
		return errors.New("fileName must be a single .apk file name")
	}
	return nil
}

func validateRelativePath(value string) error {
	if value == "" || strings.Contains(value, "\\") || strings.Contains(value, ":") || filepath.IsAbs(value) {
		return errors.New("path must be repository-relative and use forward slashes")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return errors.New("path must be clean and must not traverse parent directories")
	}
	return nil
}

func positiveVersionCode(value string) (uint64, error) {
	code, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || code == 0 {
		return 0, errors.New("versionCode must be a positive decimal integer")
	}
	return code, nil
}
