package androiddist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testSigner = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestStageAndPromoteLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	apk := filepath.Join(root, "app-debug.apk")
	apkContent := []byte("fixture APK\n")
	if err := os.WriteFile(apk, apkContent, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(apkContent)
	apkDigest := hex.EncodeToString(digest[:])

	tools := createFakeTools(t, root)
	now := time.Date(2026, 8, 6, 6, 0, 0, 0, time.UTC)
	manifest := StageManifest{
		APIVersion: APIVersion,
		Kind:       KindStageManifest,
		Metadata: StageMetadata{
			Name:    "hermes-android-debug",
			Version: "0.1.0-phase1",
		},
		Spec: StageManifestSpec{
			Channel:      "debug",
			APK:          apk,
			FileName:     "hermes-android-debug.apk",
			Package:      "com.hermesandroid.app.debug",
			VersionCode:  "1",
			VersionName:  "0.1.0-phase1",
			SHA256:       apkDigest,
			SignerSHA256: testSigner,
			Provenance: Provenance{
				SourceRepository: "https://example.invalid/hermes-android",
				Commit:           "abc123",
				BuildID:          "test-build",
			},
		},
	}

	plan, err := PlanStage(context.Background(), manifest, root, tools, now)
	if err != nil {
		t.Fatalf("PlanStage: %v", err)
	}
	planPath := filepath.Join(root, "stage-plan.json")
	planDigest, err := WriteJSON(planPath, plan)
	if err != nil {
		t.Fatalf("write stage plan: %v", err)
	}
	approvalPath := filepath.Join(root, "stage-approval.json")
	if _, err := WriteJSON(approvalPath, Approval{
		APIVersion: APIVersion,
		Kind:       KindApproval,
		Action:     ActionStage,
		PlanSHA256: planDigest,
		Approved:   true,
		ApprovedBy: "Mason Dyer",
		ApprovedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("write stage approval: %v", err)
	}

	result, err := ApplyStage(context.Background(), planPath, approvalPath, root, tools, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ApplyStage: %v", err)
	}
	for _, path := range []string{result.APKPath, result.ReleasePath, result.ChecksumsPath} {
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("expected regular file %s: info=%v err=%v", path, info, err)
		}
	}
	checksums, err := os.ReadFile(result.ChecksumsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(checksums), apkDigest+"  hermes-android-debug.apk") {
		t.Fatalf("unexpected checksums: %s", checksums)
	}
	if _, err := ApplyStage(context.Background(), planPath, approvalPath, root, tools, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expected immutable release re-stage to fail")
	}

	channelPlan, err := PlanChannel(root, "debug", "0.1.0-phase1", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("PlanChannel: %v", err)
	}
	channelPlanPath := filepath.Join(root, "channel-plan.json")
	channelPlanDigest, err := WriteJSON(channelPlanPath, channelPlan)
	if err != nil {
		t.Fatal(err)
	}
	channelApprovalPath := filepath.Join(root, "channel-approval.json")
	if _, err := WriteJSON(channelApprovalPath, Approval{
		APIVersion: APIVersion,
		Kind:       KindApproval,
		Action:     ActionPromote,
		PlanSHA256: channelPlanDigest,
		Approved:   true,
		ApprovedBy: "Mason Dyer",
		ApprovedAt: now.Add(3 * time.Minute).Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	promoted, err := PromoteChannel(channelPlanPath, channelApprovalPath, root, now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("PromoteChannel: %v", err)
	}
	if promoted.Spec.Version != "0.1.0-phase1" {
		t.Fatalf("unexpected promoted version: %s", promoted.Spec.Version)
	}
	current, err := ShowChannel(root, "debug")
	if err != nil {
		t.Fatalf("ShowChannel: %v", err)
	}
	if current.Spec.APK_SHA256 != apkDigest || current.Spec.SignerSHA256 != testSigner {
		t.Fatalf("unexpected current channel: %+v", current)
	}
}

func TestApprovalMustMatchExactPlan(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := ChannelPlan{
		APIVersion: APIVersion,
		Kind:       KindChannelPlan,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Spec: ChannelPlanSpec{
			Repository:      root,
			Channel:         "debug",
			Version:         "1.0.0",
			ReleaseManifest: "android/debug/1.0.0/release.json",
			ReleaseSHA256:   strings.Repeat("a", 64),
			ChannelManifest: "android/debug/current.json",
			Package:         "com.hermesandroid.app.debug",
			VersionCode:     "1",
			VersionName:     "1.0.0",
			APK:             "android/debug/1.0.0/app.apk",
			APK_SHA256:      strings.Repeat("b", 64),
			SignerSHA256:    strings.Repeat("c", 64),
		},
	}
	planPath := filepath.Join(root, "plan.json")
	if _, err := WriteJSON(planPath, plan); err != nil {
		t.Fatal(err)
	}
	approvalPath := filepath.Join(root, "approval.json")
	if _, err := WriteJSON(approvalPath, Approval{
		APIVersion: APIVersion,
		Kind:       KindApproval,
		Action:     ActionPromote,
		PlanSHA256: strings.Repeat("0", 64),
		Approved:   true,
		ApprovedBy: "reviewer",
		ApprovedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := PromoteChannel(planPath, approvalPath, root, time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "exact plan") {
		t.Fatalf("expected exact-plan approval failure, got %v", err)
	}
}

func TestStableRejectsDebugPackage(t *testing.T) {
	t.Parallel()

	manifest := StageManifest{
		APIVersion: APIVersion,
		Kind:       KindStageManifest,
		Metadata:   StageMetadata{Name: "hermes", Version: "1.0.0"},
		Spec: StageManifestSpec{
			Channel:      "stable",
			APK:          "/tmp/app.apk",
			FileName:     "app.apk",
			Package:      "com.hermesandroid.app.debug",
			VersionCode:  "1",
			VersionName:  "1.0.0",
			SHA256:       strings.Repeat("a", 64),
			SignerSHA256: strings.Repeat("b", 64),
			Provenance:   Provenance{SourceRepository: "repo", Commit: "commit"},
		},
	}
	if err := validateStageManifest(manifest); err == nil || !strings.Contains(err.Error(), "stable") {
		t.Fatalf("expected stable/debug rejection, got %v", err)
	}
}

func TestStrictStageManifestRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "manifest.json")
	content := `{
  "apiVersion":"hps.hermes/v1",
  "kind":"AndroidStageManifest",
  "metadata":{"name":"hermes","version":"1.0.0"},
  "spec":{
    "channel":"debug",
    "apk":"/tmp/app.apk",
    "fileName":"app.apk",
    "package":"com.hermesandroid.app.debug",
    "versionCode":"1",
    "versionName":"1.0.0",
    "sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "signerSha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "provenance":{"sourceRepository":"repo","commit":"commit"},
    "unexpected":true
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadStageManifest(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}
}

func createFakeTools(t *testing.T, root string) ToolPaths {
	t.Helper()
	toolsDir := filepath.Join(root, "tools")
	if err := os.Mkdir(toolsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	aapt2 := filepath.Join(toolsDir, "aapt2")
	apksigner := filepath.Join(toolsDir, "apksigner")
	if err := os.WriteFile(aapt2, []byte("#!/bin/sh\nprintf \"%s\\n\" \"package: name='com.hermesandroid.app.debug' versionCode='1' versionName='0.1.0-phase1'\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apksigner, []byte("#!/bin/sh\nprintf \"%s\\n\" \"Signer #1 certificate SHA-256 digest: "+testSigner+"\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return ToolPaths{AAPT2: aapt2, APKSigner: apksigner}
}

func TestReleaseJSONIsStrictlyReadable(t *testing.T) {
	t.Parallel()

	release := Release{APIVersion: APIVersion, Kind: KindRelease}
	data, err := marshalJSON(release)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Release
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
}
