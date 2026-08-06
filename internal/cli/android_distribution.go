package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mdyerapis-coder/hermes-provisioning-system/internal/androiddist"
)

func runAndroidStage(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printAndroidStageHelp(stdout)
		return 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		printAndroidStageHelp(stdout)
		return 0
	case "plan":
		return runAndroidStagePlan(args[1:], stdout, stderr)
	case "apply":
		return runAndroidStageApply(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown android stage command %q\n\n", args[0])
		printAndroidStageHelp(stderr)
		return 2
	}
}

func runAndroidStagePlan(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("android stage plan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	manifestPath := fs.String("manifest", "", "AndroidStageManifest JSON file")
	repository := fs.String("repo", envOr("HPS_REPOSITORY", "/srv/hermes"), "HPS repository root")
	out := fs.String("out", "", "new path for the generated stage plan")
	aapt2 := fs.String("aapt2", os.Getenv("HPS_AAPT2"), "optional aapt2 executable path")
	apksigner := fs.String("apksigner", os.Getenv("HPS_APKSIGNER"), "optional apksigner executable path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if missingRequired(stderr, map[string]string{"--manifest": *manifestPath, "--out": *out}) {
		return 2
	}

	manifest, err := androiddist.LoadStageManifest(*manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "stage planning failed: %v\n", err)
		return 1
	}
	plan, err := androiddist.PlanStage(context.Background(), manifest, *repository, androiddist.ToolPaths{
		AAPT2:     *aapt2,
		APKSigner: *apksigner,
	}, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "stage planning failed: %v\n", err)
		return 1
	}
	digest, err := androiddist.WriteJSON(*out, plan)
	if err != nil {
		fmt.Fprintf(stderr, "write stage plan: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "stage plan: %s\n", *out)
	fmt.Fprintf(stdout, "plan-sha256: %s\n", digest)
	fmt.Fprintf(stdout, "source APK: %s\n", plan.Spec.SourceAPK)
	fmt.Fprintf(stdout, "destination: %s/%s\n", plan.Spec.Repository, plan.Spec.ReleaseDirectory)
	fmt.Fprintln(stdout, "stage plan created; no repository files were changed")
	return 0
}

func runAndroidStageApply(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("android stage apply", flag.ContinueOnError)
	fs.SetOutput(stderr)
	planPath := fs.String("plan", "", "approved AndroidStagePlan JSON file")
	approvalPath := fs.String("approval", "", "HPSApproval JSON file bound to the plan digest")
	repository := fs.String("repo", envOr("HPS_REPOSITORY", "/srv/hermes"), "HPS repository root")
	aapt2 := fs.String("aapt2", os.Getenv("HPS_AAPT2"), "optional aapt2 executable path")
	apksigner := fs.String("apksigner", os.Getenv("HPS_APKSIGNER"), "optional apksigner executable path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if missingRequired(stderr, map[string]string{"--plan": *planPath, "--approval": *approvalPath}) {
		return 2
	}

	result, err := androiddist.ApplyStage(context.Background(), *planPath, *approvalPath, *repository, androiddist.ToolPaths{
		AAPT2:     *aapt2,
		APKSigner: *apksigner,
	}, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "APK staging failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "release directory: %s\n", result.ReleaseDirectory)
	fmt.Fprintf(stdout, "APK: %s\n", result.APKPath)
	fmt.Fprintf(stdout, "release manifest: %s\n", result.ReleasePath)
	fmt.Fprintf(stdout, "checksums: %s\n", result.ChecksumsPath)
	fmt.Fprintf(stdout, "stage-plan-sha256: %s\n", result.PlanSHA256)
	fmt.Fprintf(stdout, "approval-sha256: %s\n", result.ApprovalSHA256)
	fmt.Fprintln(stdout, "APK staged as an immutable release; no channel was promoted")
	return 0
}

func runAndroidChannel(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printAndroidChannelHelp(stdout)
		return 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		printAndroidChannelHelp(stdout)
		return 0
	case "plan":
		return runAndroidChannelPlan(args[1:], stdout, stderr)
	case "promote":
		return runAndroidChannelPromote(args[1:], stdout, stderr)
	case "show":
		return runAndroidChannelShow(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown android channel command %q\n\n", args[0])
		printAndroidChannelHelp(stderr)
		return 2
	}
}

func runAndroidChannelPlan(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("android channel plan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repository := fs.String("repo", envOr("HPS_REPOSITORY", "/srv/hermes"), "HPS repository root")
	channel := fs.String("channel", "", "release channel: debug, beta, or stable")
	version := fs.String("version", "", "immutable staged release version")
	out := fs.String("out", "", "new path for the generated promotion plan")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if missingRequired(stderr, map[string]string{"--channel": *channel, "--version": *version, "--out": *out}) {
		return 2
	}

	plan, err := androiddist.PlanChannel(*repository, *channel, *version, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "channel planning failed: %v\n", err)
		return 1
	}
	digest, err := androiddist.WriteJSON(*out, plan)
	if err != nil {
		fmt.Fprintf(stderr, "write channel plan: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "channel plan: %s\n", *out)
	fmt.Fprintf(stdout, "plan-sha256: %s\n", digest)
	fmt.Fprintf(stdout, "channel: %s\n", plan.Spec.Channel)
	fmt.Fprintf(stdout, "version: %s\n", plan.Spec.Version)
	fmt.Fprintln(stdout, "channel plan created; current.json was not changed")
	return 0
}

func runAndroidChannelPromote(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("android channel promote", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repository := fs.String("repo", envOr("HPS_REPOSITORY", "/srv/hermes"), "HPS repository root")
	planPath := fs.String("plan", "", "approved AndroidChannelPlan JSON file")
	approvalPath := fs.String("approval", "", "HPSApproval JSON file bound to the plan digest")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if missingRequired(stderr, map[string]string{"--plan": *planPath, "--approval": *approvalPath}) {
		return 2
	}

	channel, err := androiddist.PromoteChannel(*planPath, *approvalPath, *repository, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "channel promotion failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "channel: %s\n", channel.Metadata.Name)
	fmt.Fprintf(stdout, "version: %s\n", channel.Spec.Version)
	fmt.Fprintf(stdout, "release: %s\n", channel.Spec.ReleaseManifest)
	fmt.Fprintf(stdout, "APK: %s\n", channel.Spec.APK)
	fmt.Fprintln(stdout, "channel pointer promoted atomically")
	return 0
}

func runAndroidChannelShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("android channel show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repository := fs.String("repo", envOr("HPS_REPOSITORY", "/srv/hermes"), "HPS repository root")
	channelName := fs.String("channel", "", "release channel: debug, beta, or stable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if missingRequired(stderr, map[string]string{"--channel": *channelName}) {
		return 2
	}
	channel, err := androiddist.ShowChannel(*repository, *channelName)
	if err != nil {
		fmt.Fprintf(stderr, "show channel failed: %v\n", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(channel); err != nil {
		fmt.Fprintf(stderr, "encode channel: %v\n", err)
		return 1
	}
	return 0
}

func missingRequired(stderr io.Writer, values map[string]string) bool {
	missing := false
	for name, value := range values {
		if strings.TrimSpace(value) == "" {
			fmt.Fprintf(stderr, "%s is required\n", name)
			missing = true
		}
	}
	return missing
}

func printAndroidStageHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  hps android stage plan --manifest RELEASE.json --repo /srv/hermes --out STAGE-PLAN.json
  hps android stage apply --plan STAGE-PLAN.json --approval APPROVAL.json --repo /srv/hermes

Commands:
  plan    Validate the APK and create an exact, reviewable plan without changing the repository
  apply   Revalidate and atomically create an immutable release directory after exact-plan approval

Safety:
  apply requires an HPSApproval document whose planSha256 matches the exact plan file.
  Staging never promotes debug, beta, or stable current.json.
`)
}

func printAndroidChannelHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  hps android channel plan --repo /srv/hermes --channel CHANNEL --version VERSION --out PLAN.json
  hps android channel promote --repo /srv/hermes --plan PLAN.json --approval APPROVAL.json
  hps android channel show --repo /srv/hermes --channel CHANNEL

Commands:
  plan      Verify a staged release and create an exact promotion plan
  promote   Atomically update only CHANNEL/current.json after exact-plan approval
  show      Display and verify the current channel pointer

Safety:
  Historical release directories are immutable. Promotion changes only current.json.
`)
}
