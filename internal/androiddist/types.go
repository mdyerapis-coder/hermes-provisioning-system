package androiddist

import "regexp"

const (
	APIVersion        = "hps.hermes/v1"
	KindStageManifest = "AndroidStageManifest"
	KindStagePlan     = "AndroidStagePlan"
	KindChannelPlan   = "AndroidChannelPlan"
	KindApproval      = "HPSApproval"
	KindRelease       = "AndroidRelease"
	KindChannel       = "AndroidChannel"
	ActionStage       = "android-stage"
	ActionPromote     = "android-channel-promote"
)

const (
	maxManifestBytes = 1 << 20
	maxPlanBytes     = 1 << 20
	maxApprovalBytes = 64 << 10
	maxReleaseBytes  = 1 << 20
)

var (
	channelPattern     = regexp.MustCompile(`^(debug|beta|stable)$`)
	segmentPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+~-]{0,127}$`)
	packageNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
)

// StageManifest describes a locally available APK and its independently
// established identity. It is an input document and is never published.
type StageManifest struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   StageMetadata     `json:"metadata"`
	Spec       StageManifestSpec `json:"spec"`
}

type StageMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type StageManifestSpec struct {
	Channel      string     `json:"channel"`
	APK          string     `json:"apk"`
	FileName     string     `json:"fileName"`
	Package      string     `json:"package"`
	VersionCode  string     `json:"versionCode"`
	VersionName  string     `json:"versionName"`
	SHA256       string     `json:"sha256"`
	SignerSHA256 string     `json:"signerSha256"`
	Provenance   Provenance `json:"provenance"`
}

type Provenance struct {
	SourceRepository string `json:"sourceRepository"`
	Commit           string `json:"commit"`
	BuildID          string `json:"buildId,omitempty"`
	Builder          string `json:"builder,omitempty"`
}

// ToolPaths optionally pins the Android SDK tools used during validation.
type ToolPaths struct {
	AAPT2     string
	APKSigner string
}

// StagePlan is the exact, reviewable staging operation. Apply requires a
// machine-readable approval bound to the SHA-256 of the serialized plan file.
type StagePlan struct {
	APIVersion string        `json:"apiVersion"`
	Kind       string        `json:"kind"`
	CreatedAt  string        `json:"createdAt"`
	Spec       StagePlanSpec `json:"spec"`
}

type StagePlanSpec struct {
	Repository       string     `json:"repository"`
	SourceAPK        string     `json:"sourceApk"`
	Channel          string     `json:"channel"`
	Version          string     `json:"version"`
	FileName         string     `json:"fileName"`
	ReleaseDirectory string     `json:"releaseDirectory"`
	APKDestination   string     `json:"apkDestination"`
	ReleaseManifest  string     `json:"releaseManifest"`
	Checksums        string     `json:"checksums"`
	Package          string     `json:"package"`
	VersionCode      string     `json:"versionCode"`
	VersionName      string     `json:"versionName"`
	SHA256           string     `json:"sha256"`
	Size             int64      `json:"size"`
	SignerSHA256     string     `json:"signerSha256"`
	Provenance       Provenance `json:"provenance"`
}

// Approval authorizes exactly one serialized plan digest.
type Approval struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Action     string `json:"action"`
	PlanSHA256 string `json:"planSha256"`
	Approved   bool   `json:"approved"`
	ApprovedBy string `json:"approvedBy"`
	ApprovedAt string `json:"approvedAt"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
}

// Release is the public immutable release record written beside the APK.
type Release struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   ReleaseMetadata `json:"metadata"`
	Spec       ReleaseSpec     `json:"spec"`
}

type ReleaseMetadata struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	CreatedAt string `json:"createdAt"`
}

type ReleaseSpec struct {
	Channel         string     `json:"channel"`
	File            string     `json:"file"`
	Package         string     `json:"package"`
	VersionCode     string     `json:"versionCode"`
	VersionName     string     `json:"versionName"`
	SHA256          string     `json:"sha256"`
	Size            int64      `json:"size"`
	SignerSHA256    string     `json:"signerSha256"`
	Provenance      Provenance `json:"provenance"`
	StagePlanSHA256 string     `json:"stagePlanSha256"`
	ApprovalSHA256  string     `json:"approvalSha256"`
}

// StageResult identifies the immutable directory created by ApplyStage.
type StageResult struct {
	ReleaseDirectory string
	APKPath          string
	ReleasePath      string
	ChecksumsPath    string
	PlanSHA256       string
	ApprovalSHA256   string
}

// ChannelPlan is an exact proposed channel-pointer update.
type ChannelPlan struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	CreatedAt  string          `json:"createdAt"`
	Spec       ChannelPlanSpec `json:"spec"`
}

type ChannelPlanSpec struct {
	Repository      string `json:"repository"`
	Channel         string `json:"channel"`
	Version         string `json:"version"`
	ReleaseManifest string `json:"releaseManifest"`
	ReleaseSHA256   string `json:"releaseSha256"`
	ChannelManifest string `json:"channelManifest"`
	Package         string `json:"package"`
	VersionCode     string `json:"versionCode"`
	VersionName     string `json:"versionName"`
	APK             string `json:"apk"`
	APKSHA256       string `json:"apkSha256"`
	SignerSHA256    string `json:"signerSha256"`
}

// Channel is the public mutable pointer. Historical release directories remain
// immutable; promotion atomically replaces only this document.
type Channel struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   ChannelMetadata `json:"metadata"`
	Spec       ChannelSpec     `json:"spec"`
}

type ChannelMetadata struct {
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
}

type ChannelSpec struct {
	Version             string `json:"version"`
	ReleaseManifest     string `json:"releaseManifest"`
	ReleaseSHA256       string `json:"releaseSha256"`
	Package             string `json:"package"`
	VersionCode         string `json:"versionCode"`
	VersionName         string `json:"versionName"`
	APK                 string `json:"apk"`
	APKSHA256           string `json:"apkSha256"`
	SignerSHA256        string `json:"signerSha256"`
	PromotionPlanSHA256 string `json:"promotionPlanSha256"`
	ApprovalSHA256      string `json:"approvalSha256"`
}
