package main

import (
	"cowpoke/internal/cli"
)

// Build information. These will be set during build time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	// Set version information that can be used by the CLI
	cli.SetVersionInfo(version, commit, date, builtBy)
	cli.Execute()
}
