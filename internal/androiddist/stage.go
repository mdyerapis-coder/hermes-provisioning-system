package androiddist

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mdyerapis-coder/hermes-provisioning-system/internal/androidapk"
)

func LoadStageManifest(path string) (StageManifest, error) {
	var manifest StageManifest
	if err := decodeStrictFile(path, maxManifestBytes, &manifest); err != nil {
		return StageManifest{}, fmt.Errorf("load stage manifest: %w", err)
	}
	if err := validateStageManifest(manifest); err != nil {
		return StageManifest{}, err
	}
	return manifest, nil
}

// PlanStage validates the APK and returns an exact, non-mutating stage plan.
func PlanStage(ctx context.Context, manifest StageManifest, repository string, tools ToolPaths, now time.Time) (StagePlan, error) {
	if err := validateStageManifest(manifest); err != nil {
		return StagePlan{}, err
	}
	repository, err := canonicalDirectory(repository)
	if err != nil {
		return StagePlan{}, fmt.Errorf("repository: %w", err)
	}
	source, err := canonicalRegularFile(manifest.Spec.APK)
	if err != nil {
		return StagePlan{}, fmt.Errorf("resolve APK path: %w", err)
	}

	result, err := androidapk.Validate(ctx, androidapk.Options{
		Path:         source,
		SHA256:       manifest.Spec.SHA256,
		Package:      manifest.Spec.Package,
		VersionCode:  manifest.Spec.VersionCode,
		VersionName:  manifest.Spec.VersionName,
		SignerSHA256: manifest.Spec.SignerSHA256,
		AAPT2:        tools.AAPT2,
		APKSigner:    tools.APKSigner,
	})
	if err != nil {
		return StagePlan{}, fmt.Errorf("validate APK before planning: %w", err)
	}

	releaseDir := filepath.ToSlash(filepath.Join("android", manifest.Spec.Channel, manifest.Metadata.Version))
	apkDest := filepath.ToSlash(filepath.Join(releaseDir, manifest.Spec.FileName))
	return StagePlan{
		APIVersion: APIVersion,
		Kind:       KindStagePlan,
		CreatedAt:  now.UTC().Format(time.RFC3339),
		Spec: StagePlanSpec{
			Repository:       repository,
			SourceAPK:        source,
			Channel:          manifest.Spec.Channel,
			Version:          manifest.Metadata.Version,
			FileName:         manifest.Spec.FileName,
			ReleaseDirectory: releaseDir,
			APKDestination:   apkDest,
			ReleaseManifest:  filepath.ToSlash(filepath.Join(releaseDir, "release.json")),
			Checksums:        filepath.ToSlash(filepath.Join(releaseDir, "SHA256SUMS")),
			Package:          result.Package,
			VersionCode:      result.VersionCode,
			VersionName:      result.VersionName,
			SHA256:           result.SHA256,
			Size:             result.Size,
			SignerSHA256:     result.SignerSHA256,
			Provenance:       manifest.Spec.Provenance,
		},
	}, nil
}

// WriteJSON writes a canonical indented JSON document without overwriting an
// existing file. The returned digest is the SHA-256 of the exact bytes written.
func WriteJSON(path string, value any) (string, error) {
	data, err := marshalJSON(value)
	if err != nil {
		return "", err
	}
	if err := writeExclusiveFile(path, data, 0o600); err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

// ApplyStage revalidates the exact plan and source APK, requires an approval
// bound to the plan digest, and atomically creates one immutable release dir.
func ApplyStage(ctx context.Context, planPath, approvalPath, repository string, tools ToolPaths, now time.Time) (StageResult, error) {
	var plan StagePlan
	planBytes, err := decodeStrictFileBytes(planPath, maxPlanBytes, &plan)
	if err != nil {
		return StageResult{}, fmt.Errorf("load stage plan: %w", err)
	}
	if err := validateStagePlan(plan); err != nil {
		return StageResult{}, err
	}
	planDigest := digestBytes(planBytes)

	var approval Approval
	approvalBytes, err := decodeStrictFileBytes(approvalPath, maxApprovalBytes, &approval)
	if err != nil {
		return StageResult{}, fmt.Errorf("load approval: %w", err)
	}
	if err := validateApproval(approval, ActionStage, planDigest, now); err != nil {
		return StageResult{}, err
	}
	approvalDigest := digestBytes(approvalBytes)

	repository, err = canonicalDirectory(repository)
	if err != nil {
		return StageResult{}, fmt.Errorf("repository: %w", err)
	}
	if repository != plan.Spec.Repository {
		return StageResult{}, fmt.Errorf("repository mismatch: plan requires %q, command selected %q", plan.Spec.Repository, repository)
	}

	source, err := canonicalRegularFile(plan.Spec.SourceAPK)
	if err != nil {
		return StageResult{}, fmt.Errorf("source APK: %w", err)
	}
	if source != plan.Spec.SourceAPK {
		return StageResult{}, errors.New("source APK path no longer matches the approved plan")
	}

	validation, err := androidapk.Validate(ctx, androidapk.Options{
		Path:         source,
		SHA256:       plan.Spec.SHA256,
		Package:      plan.Spec.Package,
		VersionCode:  plan.Spec.VersionCode,
		VersionName:  plan.Spec.VersionName,
		SignerSHA256: plan.Spec.SignerSHA256,
		AAPT2:        tools.AAPT2,
		APKSigner:    tools.APKSigner,
	})
	if err != nil {
		return StageResult{}, fmt.Errorf("revalidate APK before staging: %w", err)
	}
	if validation.Size != plan.Spec.Size {
		return StageResult{}, fmt.Errorf("APK size mismatch: plan records %d bytes, source is %d bytes", plan.Spec.Size, validation.Size)
	}

	if err := validateRelativePath(plan.Spec.ReleaseDirectory); err != nil {
		return StageResult{}, fmt.Errorf("release directory: %w", err)
	}
	finalDir := filepath.Join(repository, filepath.FromSlash(plan.Spec.ReleaseDirectory))
	if _, err := os.Lstat(finalDir); err == nil {
		return StageResult{}, fmt.Errorf("immutable release directory already exists: %s", finalDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return StageResult{}, fmt.Errorf("inspect release directory: %w", err)
	}

	parentRel := filepath.ToSlash(filepath.Dir(plan.Spec.ReleaseDirectory))
	parentDir, err := ensureSafeDirectoryTree(repository, parentRel)
	if err != nil {
		return StageResult{}, err
	}
	tempDir, err := os.MkdirTemp(parentDir, ".hps-stage-")
	if err != nil {
		return StageResult{}, fmt.Errorf("create staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(tempDir)
		}
	}()
	if err := os.Chmod(tempDir, 0o755); err != nil {
		return StageResult{}, fmt.Errorf("set staging directory permissions: %w", err)
	}

	stagedAPK := filepath.Join(tempDir, plan.Spec.FileName)
	copiedSize, copiedDigest, err := copyRegularFile(plan.Spec.SourceAPK, stagedAPK, 0o644)
	if err != nil {
		return StageResult{}, err
	}
	if copiedSize != plan.Spec.Size || copiedDigest != plan.Spec.SHA256 {
		return StageResult{}, errors.New("copied APK does not match the approved plan")
	}

	release := Release{
		APIVersion: APIVersion,
		Kind:       KindRelease,
		Metadata: ReleaseMetadata{
			Name:      plan.Spec.Package,
			Version:   plan.Spec.Version,
			CreatedAt: now.UTC().Format(time.RFC3339),
		},
		Spec: ReleaseSpec{
			Channel:         plan.Spec.Channel,
			File:            plan.Spec.FileName,
			Package:         plan.Spec.Package,
			VersionCode:     plan.Spec.VersionCode,
			VersionName:     plan.Spec.VersionName,
			SHA256:          plan.Spec.SHA256,
			Size:            plan.Spec.Size,
			SignerSHA256:    plan.Spec.SignerSHA256,
			Provenance:      plan.Spec.Provenance,
			StagePlanSHA256: planDigest,
			ApprovalSHA256:  approvalDigest,
		},
	}
	releaseBytes, err := marshalJSON(release)
	if err != nil {
		return StageResult{}, err
	}
	releasePath := filepath.Join(tempDir, "release.json")
	if err := writeExclusiveFile(releasePath, releaseBytes, 0o644); err != nil {
		return StageResult{}, err
	}
	releaseDigest := digestBytes(releaseBytes)

	checksums := fmt.Sprintf("%s  %s\n%s  release.json\n", plan.Spec.SHA256, plan.Spec.FileName, releaseDigest)
	checksumsPath := filepath.Join(tempDir, "SHA256SUMS")
	if err := writeExclusiveFile(checksumsPath, []byte(checksums), 0o644); err != nil {
		return StageResult{}, err
	}
	if err := syncDirectory(tempDir); err != nil {
		return StageResult{}, err
	}
	if err := os.Mkdir(finalDir, 0o755); err != nil {
		return StageResult{}, fmt.Errorf("reserve immutable release directory: %w", err)
	}
	reserved := true
	defer func() {
		if reserved && !committed {
			_ = os.RemoveAll(finalDir)
		}
	}()
	for _, name := range []string{plan.Spec.FileName, "SHA256SUMS", "release.json"} {
		if err := os.Rename(filepath.Join(tempDir, name), filepath.Join(finalDir, name)); err != nil {
			return StageResult{}, fmt.Errorf("commit release file %s: %w", name, err)
		}
	}
	if err := syncDirectory(finalDir); err != nil {
		return StageResult{}, err
	}
	committed = true
	reserved = false
	if err := os.Remove(tempDir); err != nil {
		return StageResult{}, fmt.Errorf("remove empty staging directory: %w", err)
	}
	if err := syncDirectory(parentDir); err != nil {
		return StageResult{}, err
	}

	return StageResult{
		ReleaseDirectory: finalDir,
		APKPath:          filepath.Join(finalDir, plan.Spec.FileName),
		ReleasePath:      filepath.Join(finalDir, "release.json"),
		ChecksumsPath:    filepath.Join(finalDir, "SHA256SUMS"),
		PlanSHA256:       planDigest,
		ApprovalSHA256:   approvalDigest,
	}, nil
}
