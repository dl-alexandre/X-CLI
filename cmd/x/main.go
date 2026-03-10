package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/dl-alexandre/X-CLI/internal/cli"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	// Set version info in cli package
	cli.Version = version
	cli.BinaryName = "x"
	cli.GitHubRepo = "X-CLI"
	cli.GitCommit = gitCommit
	cli.BuildTime = buildTime

	var c cli.CLI
	ctx := kong.Parse(&c,
		kong.Name("x"),
		kong.Description("A terminal-first CLI for X/Twitter: timelines, search, profiles, and bookmarks without API keys"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": version,
		},
	)

	if ctx.Command() == "version" {
		fmt.Printf("x %s (commit: %s) built %s\n", version, gitCommit, buildTime)
		os.Exit(0)
	}

	if err := ctx.Run(&c.Globals); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
