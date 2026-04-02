package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/dl-alexandre/X-CLI/internal/cli"
	cliver "github.com/dl-alexandre/cli-tools/version"
	kongcompletion "github.com/jotaen/kong-completion"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	// Set version info in cli-tools
	cliver.Version = version
	cliver.GitCommit = gitCommit
	cliver.BuildTime = buildTime
	cliver.BinaryName = "x"

	var c cli.CLI
	parser := kong.Must(&c,
		kong.Name("x"),
		kong.Description("A terminal-first CLI for X: timelines, search, profiles, and bookmarks without API keys"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": version,
		},
	)

	kongcompletion.Register(parser)

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		parser.Errorf("%s", err)
		os.Exit(1)
	}

	if ctx.Command() == "version" {
		fmt.Printf("x %s (commit: %s) built %s\n", cliver.Version, cliver.GitCommit, cliver.BuildTime)
		os.Exit(0)
	}

	// Run auto-update check in background (after initialization)
	// This runs asynchronously and won't block the main command
	go func() {
		// Small delay to not interfere with command output
		time.Sleep(100 * time.Millisecond)
		cli.AutoUpdateCheck()
	}()

	if err := ctx.Run(&c.Globals); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
