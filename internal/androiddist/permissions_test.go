package androiddist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestPublishedModesIgnoreRestrictiveUmask(t *testing.T) {
	if os.Getenv("HPS_TEST_RESTRICTIVE_UMASK_HELPER") == "1" {
		runPublishedModesRestrictiveUmaskTest(t)
		return
	}

	command := exec.Command(os.Args[0], "-test.run=^TestPublishedModesIgnoreRestrictiveUmask$")
	command.Env = append(os.Environ(), "HPS_TEST_RESTRICTIVE_UMASK_HELPER=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("restrictive-umask subprocess failed: %v\n%s", err, output)
	}
}

func runPublishedModesRestrictiveUmaskTest(t *testing.T) {
	oldUmask := syscall.Umask(0o077)
	defer syscall.Umask(oldUmask)

	root := t.TempDir()
	apk := filepath.Join(root, "app-debug.apk")
	apkContent := []byte("fixture APK\n")
	if err := os.WriteFile(apk, apkContent, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(apkContent)
	apkDigest := hex.EncodeToString(digest[:])

	tools := createFakeTools(t, root)
	for _, path := range []string{
		filepath.Join(root, "android"),
		filepath.Join(root, "android", "debug"),
	} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
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
				BuildID:          "restrictive-umask-test",
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
		ApprovedBy: "test",
		ApprovedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("write stage approval: %v", err)
	}

	result, err := ApplyStage(context.Background(), planPath, approvalPath, root, tools, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ApplyStage: %v", err)
	}

	for _, path := range []string{
		filepath.Join(root, "android"),
		filepath.Join(root, "android", "debug"),
		result.ReleaseDirectory,
	} {
		requirePublishedMode(t, path, 0o755)
	}
	for _, path := range []string{result.APKPath, result.ReleasePath, result.ChecksumsPath} {
		requirePublishedMode(t, path, 0o644)
	}

	channelPlan, err := PlanChannel(root, "debug", "0.1.0-phase1", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("PlanChannel: %v", err)
	}
	channelPlanPath := filepath.Join(root, "channel-plan.json")
	channelPlanDigest, err := WriteJSON(channelPlanPath, channelPlan)
	if err != nil {
		t.Fatalf("write channel plan: %v", err)
	}
	channelApprovalPath := filepath.Join(root, "channel-approval.json")
	if _, err := WriteJSON(channelApprovalPath, Approval{
		APIVersion: APIVersion,
		Kind:       KindApproval,
		Action:     ActionPromote,
		PlanSHA256: channelPlanDigest,
		Approved:   true,
		ApprovedBy: "test",
		ApprovedAt: now.Add(2 * time.Minute).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("write channel approval: %v", err)
	}
	if _, err := PromoteChannel(channelPlanPath, channelApprovalPath, root, now.Add(3*time.Minute)); err != nil {
		t.Fatalf("PromoteChannel: %v", err)
	}
	requirePublishedMode(t, filepath.Join(root, "android", "debug", "current.json"), 0o644)
}

func requirePublishedMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if actual := info.Mode().Perm(); actual != expected {
		t.Fatalf("mode for %s: got %04o, want %04o", path, actual, expected)
	}
}
