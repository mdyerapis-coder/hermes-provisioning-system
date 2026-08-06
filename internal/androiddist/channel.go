package androiddist

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func PlanChannel(repository, channel, version string, now time.Time) (ChannelPlan, error) {
	repository, err := canonicalDirectory(repository)
	if err != nil {
		return ChannelPlan{}, fmt.Errorf("repository: %w", err)
	}
	if err := validateChannel(channel); err != nil {
		return ChannelPlan{}, err
	}
	if err := validateSegment("version", version); err != nil {
		return ChannelPlan{}, err
	}

	releaseRel := filepath.ToSlash(filepath.Join("android", channel, version, "release.json"))
	if err := validateRelativePath(releaseRel); err != nil {
		return ChannelPlan{}, err
	}
	releasePath := filepath.Join(repository, filepath.FromSlash(releaseRel))
	var release Release
	releaseBytes, err := decodeStrictFileBytes(releasePath, maxReleaseBytes, &release)
	if err != nil {
		return ChannelPlan{}, fmt.Errorf("load release manifest: %w", err)
	}
	if err := validateRelease(release, channel, version); err != nil {
		return ChannelPlan{}, err
	}
	apkRel := filepath.ToSlash(filepath.Join("android", channel, version, release.Spec.File))
	apkPath := filepath.Join(repository, filepath.FromSlash(apkRel))
	apkDigest, apkSize, err := digestRegularFile(apkPath)
	if err != nil {
		return ChannelPlan{}, fmt.Errorf("verify staged APK: %w", err)
	}
	if apkDigest != release.Spec.SHA256 || apkSize != release.Spec.Size {
		return ChannelPlan{}, errors.New("staged APK does not match release.json")
	}

	return ChannelPlan{
		APIVersion: APIVersion,
		Kind:       KindChannelPlan,
		CreatedAt:  now.UTC().Format(time.RFC3339),
		Spec: ChannelPlanSpec{
			Repository:      repository,
			Channel:         channel,
			Version:         version,
			ReleaseManifest: releaseRel,
			ReleaseSHA256:   digestBytes(releaseBytes),
			ChannelManifest: filepath.ToSlash(filepath.Join("android", channel, "current.json")),
			Package:         release.Spec.Package,
			VersionCode:     release.Spec.VersionCode,
			VersionName:     release.Spec.VersionName,
			APK:             apkRel,
			APKSHA256:       release.Spec.SHA256,
			SignerSHA256:    release.Spec.SignerSHA256,
		},
	}, nil
}

// PromoteChannel atomically updates current.json after exact-plan approval.
func PromoteChannel(planPath, approvalPath, repository string, now time.Time) (Channel, error) {
	var plan ChannelPlan
	planBytes, err := decodeStrictFileBytes(planPath, maxPlanBytes, &plan)
	if err != nil {
		return Channel{}, fmt.Errorf("load channel plan: %w", err)
	}
	if err := validateChannelPlan(plan); err != nil {
		return Channel{}, err
	}
	planDigest := digestBytes(planBytes)

	var approval Approval
	approvalBytes, err := decodeStrictFileBytes(approvalPath, maxApprovalBytes, &approval)
	if err != nil {
		return Channel{}, fmt.Errorf("load approval: %w", err)
	}
	if err := validateApproval(approval, ActionPromote, planDigest, now); err != nil {
		return Channel{}, err
	}
	approvalDigest := digestBytes(approvalBytes)

	repository, err = canonicalDirectory(repository)
	if err != nil {
		return Channel{}, fmt.Errorf("repository: %w", err)
	}
	if repository != plan.Spec.Repository {
		return Channel{}, fmt.Errorf("repository mismatch: plan requires %q, command selected %q", plan.Spec.Repository, repository)
	}

	releasePath := filepath.Join(repository, filepath.FromSlash(plan.Spec.ReleaseManifest))
	releaseDigest, _, err := digestRegularFile(releasePath)
	if err != nil {
		return Channel{}, fmt.Errorf("verify release manifest: %w", err)
	}
	if releaseDigest != plan.Spec.ReleaseSHA256 {
		return Channel{}, errors.New("release manifest changed after channel planning")
	}
	apkPath := filepath.Join(repository, filepath.FromSlash(plan.Spec.APK))
	apkDigest, _, err := digestRegularFile(apkPath)
	if err != nil {
		return Channel{}, fmt.Errorf("verify staged APK before promotion: %w", err)
	}
	if apkDigest != plan.Spec.APKSHA256 {
		return Channel{}, errors.New("staged APK changed after channel planning")
	}

	channelDoc := Channel{
		APIVersion: APIVersion,
		Kind:       KindChannel,
		Metadata: ChannelMetadata{
			Name:      plan.Spec.Channel,
			UpdatedAt: now.UTC().Format(time.RFC3339),
		},
		Spec: ChannelSpec{
			Version:             plan.Spec.Version,
			ReleaseManifest:     plan.Spec.ReleaseManifest,
			ReleaseSHA256:       plan.Spec.ReleaseSHA256,
			Package:             plan.Spec.Package,
			VersionCode:         plan.Spec.VersionCode,
			VersionName:         plan.Spec.VersionName,
			APK:                 plan.Spec.APK,
			APKSHA256:           plan.Spec.APKSHA256,
			SignerSHA256:        plan.Spec.SignerSHA256,
			PromotionPlanSHA256: planDigest,
			ApprovalSHA256:      approvalDigest,
		},
	}
	data, err := marshalJSON(channelDoc)
	if err != nil {
		return Channel{}, err
	}
	channelDir, err := ensureSafeDirectoryTree(repository, filepath.ToSlash(filepath.Join("android", plan.Spec.Channel)))
	if err != nil {
		return Channel{}, err
	}
	finalPath := filepath.Join(repository, filepath.FromSlash(plan.Spec.ChannelManifest))
	temp, err := os.CreateTemp(channelDir, ".current-*.json")
	if err != nil {
		return Channel{}, fmt.Errorf("create channel manifest temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return Channel{}, err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return Channel{}, err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return Channel{}, err
	}
	if err := temp.Close(); err != nil {
		return Channel{}, err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return Channel{}, fmt.Errorf("promote channel pointer: %w", err)
	}
	if err := syncDirectory(channelDir); err != nil {
		return Channel{}, err
	}
	return channelDoc, nil
}

// ShowChannel loads current.json and verifies the referenced release manifest
// and APK against the pointer's pinned digests.
func ShowChannel(repository, channel string) (Channel, error) {
	repository, err := canonicalDirectory(repository)
	if err != nil {
		return Channel{}, fmt.Errorf("repository: %w", err)
	}
	if err := validateChannel(channel); err != nil {
		return Channel{}, err
	}
	path := filepath.Join(repository, "android", channel, "current.json")
	var current Channel
	if err := decodeStrictFile(path, maxReleaseBytes, &current); err != nil {
		return Channel{}, fmt.Errorf("load channel manifest: %w", err)
	}
	if err := validateChannelDocument(current, channel); err != nil {
		return Channel{}, err
	}
	releasePath := filepath.Join(repository, filepath.FromSlash(current.Spec.ReleaseManifest))
	releaseDigest, _, err := digestRegularFile(releasePath)
	if err != nil {
		return Channel{}, fmt.Errorf("verify referenced release manifest: %w", err)
	}
	if releaseDigest != current.Spec.ReleaseSHA256 {
		return Channel{}, errors.New("current channel pointer references a changed release manifest")
	}
	apkPath := filepath.Join(repository, filepath.FromSlash(current.Spec.APK))
	apkDigest, _, err := digestRegularFile(apkPath)
	if err != nil {
		return Channel{}, fmt.Errorf("verify referenced APK: %w", err)
	}
	if apkDigest != current.Spec.APKSHA256 {
		return Channel{}, errors.New("current channel pointer references a changed APK")
	}
	return current, nil
}
