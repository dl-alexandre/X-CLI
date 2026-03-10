package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/dl-alexandre/X-CLI/internal/cli"
	kongcompletion "github.com/jotaen/kong-completion"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	cli.Version = version
	cli.BinaryName = "x"
	cli.GitHubRepo = "X-CLI"
	cli.GitCommit = gitCommit
	cli.BuildTime = buildTime

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
		fmt.Printf("x %s (commit: %s) built %s\n", version, gitCommit, buildTime)
		os.Exit(0)
	}

	if err := ctx.Run(&c.Globals); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
