package main

import (
	"os"

	"github.com/mdyerapis-coder/hermes-provisioning-system/internal/cli"
)

var (
	version = "0.1.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}, os.Stdout, os.Stderr))
}
