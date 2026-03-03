package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/mattn/go-isatty"
	"github.com/dl-alexandre/cli-template/internal/api"
	"github.com/dl-alexandre/cli-template/internal/cache"
	"github.com/dl-alexandre/cli-template/internal/config"
	"github.com/dl-alexandre/cli-template/internal/output"
)

// CLI is the main command-line interface structure using Kong
type CLI struct {
	Globals

	List       ListCmd       `cmd:"" help:"List all resources"`
	Get        GetCmd        `cmd:"" help:"Get a resource by ID"`
	Search     SearchCmd     `cmd:"" help:"Search for resources"`
	Version    VersionCmd    `cmd:"" help:"Show version information"`
	Completion CompletionCmd `cmd:"" help:"Generate shell completion script"`
}

// Globals contains global flags available to all commands
type Globals struct {
	ConfigFile string `help:"Config file path" short:"c" env:"{{APPNAME}}_CONFIG"`
	APIURL     string `help:"API base URL" env:"{{APPNAME}}_API_URL"`
	Timeout    int    `help:"Request timeout in seconds" default:"30" env:"{{APPNAME}}_TIMEOUT"`
	NoCache    bool   `help:"Disable caching" env:"{{APPNAME}}_NO_CACHE"`
	CacheDir   string `help:"Cache directory" env:"{{APPNAME}}_CACHE_DIR"`
	CacheTTL   int    `help:"Cache TTL in minutes" default:"60" env:"{{APPNAME}}_CACHE_TTL"`
	Verbose    bool   `help:"Enable verbose output" short:"v" env:"{{APPNAME}}_VERBOSE"`
	Debug      bool   `help:"Enable debug output" env:"{{APPNAME}}_DEBUG"`
	Format     string `help:"Output format: table, json, markdown" default:"table" enum:"table,json,markdown" env:"{{APPNAME}}_FORMAT"`

	// Runtime dependencies (initialized by AfterApply)
	Config *config.Config `kong:"-"`
	Cache  *cache.Cache   `kong:"-"`
	Client *api.Client    `kong:"-"`
}

// AfterApply initializes runtime dependencies after flag parsing
func (g *Globals) AfterApply() error {
	// Skip initialization for help and version commands
	if g.ConfigFile == "" && g.APIURL == "" {
		return nil
	}

	// Load configuration
	flags := config.Flags{
		ConfigFile: g.ConfigFile,
		APIURL:     g.APIURL,
		Timeout:    g.Timeout,
		NoCache:    g.NoCache,
		CacheDir:   g.CacheDir,
		CacheTTL:   g.CacheTTL,
		Verbose:    g.Verbose,
		Debug:      g.Debug,
		Format:     g.Format,
	}

	cfg, err := config.Load(flags)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	g.Config = cfg

	// Initialize cache if enabled
	if !g.NoCache && cfg.Cache.Enabled {
		g.Cache = cache.New(cfg.Cache.Dir, cfg.Cache.TTL)
	}

	// Initialize API client
	g.Client = api.NewClient(api.ClientOptions{
		BaseURL: cfg.API.URL,
		Timeout: cfg.API.Timeout,
		Verbose: g.Verbose,
		Debug:   g.Debug,
	})

	return nil
}

// ShouldUseColor determines if color output should be used
func (g *Globals) ShouldUseColor() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// GetPrinter returns an output printer based on format
func (g *Globals) GetPrinter() output.Printer {
	return output.NewPrinter(g.Format, g.ShouldUseColor())
}

// ListCmd handles the list command
type ListCmd struct {
	Limit  int    `help:"Maximum number of results" default:"20"`
	Offset int    `help:"Offset for pagination" default:"0"`
	Format string `help:"Output format (overrides global)" enum:"table,json,markdown"`
}

func (c *ListCmd) Run(globals *Globals) error {
	format := c.Format
	if format == "" {
		format = globals.Format
	}

	ctx := context.Background()
	items, err := globals.Client.List(ctx, c.Limit, c.Offset)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintItems(items)
}

// GetCmd handles the get command
type GetCmd struct {
	ID     string `arg:"" help:"Resource ID to retrieve"`
	Format string `help:"Output format (overrides global)" enum:"table,json,markdown"`
}

func (c *GetCmd) Run(globals *Globals) error {
	format := c.Format
	if format == "" {
		format = globals.Format
	}

	ctx := context.Background()
	item, err := globals.Client.Get(ctx, c.ID)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintItem(item)
}

// SearchCmd handles the search command
type SearchCmd struct {
	Query  string `arg:"" help:"Search query"`
	Limit  int    `help:"Maximum number of results" default:"10"`
	Format string `help:"Output format (overrides global)" enum:"table,json,markdown"`
}

func (c *SearchCmd) Run(globals *Globals) error {
	format := c.Format
	if format == "" {
		format = globals.Format
	}

	ctx := context.Background()
	items, err := globals.Client.Search(ctx, c.Query, c.Limit)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintItems(items)
}

// VersionCmd handles the version command
type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	// Version is handled in main.go
	return nil
}

// CompletionCmd handles shell completion generation
type CompletionCmd struct {
	Shell string `arg:"" help:"Shell: bash, zsh, fish, powershell" enum:"bash,zsh,fish,powershell"`
}

func (c *CompletionCmd) Run() error {
	var shell kong.Completer

	switch c.Shell {
	case "bash":
		shell = kong.BashCompletion
	case "zsh":
		shell = kong.ZshCompletion
	case "fish":
		shell = kong.FishCompletion
	case "powershell":
		shell = kong.PowerShellCompletion
	}

	fmt.Println(shell)
	return nil
}
