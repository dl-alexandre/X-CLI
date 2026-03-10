package cli

import (
	"fmt"
	"os"

	"github.com/dl-alexandre/X-CLI/internal/config"
	"github.com/dl-alexandre/X-CLI/internal/output"
	"github.com/dl-alexandre/X-CLI/internal/xapi"
	"github.com/mattn/go-isatty"
)

type CLI struct {
	Globals

	Status      StatusCmd      `cmd:"" help:"Show scaffold status and planned capabilities"`
	Doctor      DoctorCmd      `cmd:"" help:"Check auth, browser, and native transport readiness"`
	AnalyzeTXID AnalyzeTXIDCmd `cmd:"" name:"analyze-txid" help:"Analyze a captured txid JSONL corpus"`
	HarvestTXID HarvestTXIDCmd `cmd:"" name:"harvest-txid" help:"Extract salt samples from txid trace"`
	CompareTXID CompareTXIDCmd `cmd:"" name:"compare-txid" help:"Compare salts across operations to detect patterns"`
	Feed        FeedCmd        `cmd:"" help:"Fetch the home timeline"`
	Favorites   FavoritesCmd   `cmd:"" help:"Fetch bookmarked tweets"`
	Search      SearchCmd      `cmd:"" help:"Search tweets"`
	Tweet       TweetCmd       `cmd:"" help:"Show a tweet and replies"`
	List        ListCmd        `cmd:"" name:"list" help:"Fetch tweets from a list"`
	User        UserCmd        `cmd:"" help:"Show a user profile"`
	UserPosts   UserPostsCmd   `cmd:"" name:"user-posts" help:"Fetch tweets from a user"`
	Likes       LikesCmd       `cmd:"" help:"Fetch tweets liked by a user"`
	Followers   FollowersCmd   `cmd:"" help:"Fetch followers for a user"`
	Following   FollowingCmd   `cmd:"" help:"Fetch accounts followed by a user"`
	Post        PostCmd        `cmd:"" help:"Create a new post"`
	Delete      DeleteCmd      `cmd:"" help:"Delete one of your posts"`
	Like        LikeCmd        `cmd:"" help:"Like a post"`
	Unlike      UnlikeCmd      `cmd:"" help:"Unlike a post"`
	Retweet     RetweetCmd     `cmd:"" help:"Retweet a post"`
	Unretweet   UnretweetCmd   `cmd:"" help:"Undo a retweet"`
	Bookmark    BookmarkCmd    `cmd:"" help:"Bookmark a post"`
	Unbookmark  UnbookmarkCmd  `cmd:"" help:"Remove a bookmark"`
	Version     VersionCmd     `cmd:"" help:"Show version information"`
	Completion  CompletionCmd  `cmd:"" help:"Generate shell completion guidance"`
}

type Globals struct {
	ConfigFile    string `help:"Config file path" short:"c" env:"X_CONFIG"`
	Format        string `help:"Output format" default:"table" enum:"table,json,markdown" env:"X_FORMAT"`
	AuthToken     string `help:"Twitter auth token" env:"TWITTER_AUTH_TOKEN"`
	CT0           string `help:"Twitter ct0 token" env:"TWITTER_CT0"`
	Proxy         string `help:"HTTP or SOCKS proxy" env:"TWITTER_PROXY"`
	TraceTxIDFile string `help:"Write x-client-transaction-id traces to JSONL" env:"X_BROWSER_TRACE_TXID_FILE"`
	TraceTxIDOps  string `help:"Comma-separated txid operations to trace" env:"X_BROWSER_TRACE_TXID_OPS"`
	Verbose       bool   `help:"Enable verbose logging" short:"v" env:"X_VERBOSE"`
	Debug         bool   `help:"Enable debug logging" env:"X_DEBUG"`
	NoColor       bool   `help:"Disable ANSI color in table output" env:"NO_COLOR"`

	Config *config.Config `kong:"-"`
	Client *xapi.Client   `kong:"-"`
}

func (g *Globals) AfterApply() error {
	cfg, err := config.Load(config.Flags{
		ConfigFile:    g.ConfigFile,
		Format:        g.Format,
		Verbose:       g.Verbose,
		Debug:         g.Debug,
		AuthToken:     g.AuthToken,
		CT0:           g.CT0,
		Proxy:         g.Proxy,
		TraceTxIDFile: g.TraceTxIDFile,
		TraceTxIDOps:  g.TraceTxIDOps,
	})
	if err != nil {
		return err
	}

	g.Config = cfg
	g.Client = xapi.NewClient(xapi.Options{
		Config:  cfg,
		Verbose: g.Verbose,
		Debug:   g.Debug,
	})

	if g.Format == "" {
		g.Format = cfg.Output.Format
	}

	return nil
}

func (g *Globals) ShouldUseColor() bool {
	if g.NoColor {
		return false
	}
	return isatty.IsTerminal(os.Stdout.Fd())
}

func (g *Globals) Printer(format string) *output.Printer {
	if format == "" {
		format = g.Format
	}
	return output.NewPrinter(format, g.ShouldUseColor())
}

type StatusCmd struct {
}

func (c *StatusCmd) Run(globals *Globals) error {
	return globals.Printer("").PrintStatus(globals.Client.Status(Version))
}

type HarvestTXIDCmd struct {
	File string `arg:"" help:"Path to txid JSONL trace file"`
}

func (c *HarvestTXIDCmd) Run(globals *Globals) error {
	salts, err := xapi.ExtractSalts(c.File)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintSalts(salts)
}

type CompareTXIDCmd struct {
	File string `arg:"" help:"Path to txid JSONL trace file"`
}

func (c *CompareTXIDCmd) Run(globals *Globals) error {
	comps, err := xapi.CompareSalts(c.File)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintSaltComparisons(comps)
}

type DoctorCmd struct{}

func (c *DoctorCmd) Run(globals *Globals) error {
	return globals.Printer("").PrintDoctor(globals.Client.Doctor())
}

type AnalyzeTXIDCmd struct {
	File      string `arg:"" help:"Path to txid JSONL trace file"`
	Operation string `help:"Analyze only a specific operation"`
	FullBits  bool   `help:"Include the full 560-bit probability map"`
}

func (c *AnalyzeTXIDCmd) Run(globals *Globals) error {
	report, err := xapi.BuildTXIDAnalysisReport(c.File, c.Operation, c.FullBits)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintDoctor(report)
}

type FeedCmd struct {
	Type string `help:"Feed type" default:"for-you" enum:"for-you,following"`
	Max  int    `help:"Maximum tweets to fetch" default:"20"`
}

func (c *FeedCmd) Run(globals *Globals) error {
	result, err := globals.Client.Feed(c.Type, c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type FavoritesCmd struct {
	Max int `help:"Maximum tweets to fetch" default:"20"`
}

func (c *FavoritesCmd) Run(globals *Globals) error {
	result, err := globals.Client.Favorites(c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type SearchCmd struct {
	Query string `arg:"" help:"Search query"`
	Type  string `help:"Search tab" default:"Top" enum:"Top,Latest,Photos,Videos"`
	Max   int    `help:"Maximum tweets to fetch" default:"20"`
}

func (c *SearchCmd) Run(globals *Globals) error {
	result, err := globals.Client.Search(c.Query, c.Type, c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type TweetCmd struct {
	ID         string `arg:"" help:"Tweet ID or full status URL"`
	MaxReplies int    `help:"Maximum replies to fetch" default:"20"`
}

func (c *TweetCmd) Run(globals *Globals) error {
	thread, err := globals.Client.Tweet(xapi.NormalizeTweetID(c.ID), c.MaxReplies)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTweetThread(thread)
}

type ListCmd struct {
	ListID string `arg:"" help:"Twitter list ID"`
	Max    int    `help:"Maximum tweets to fetch" default:"20"`
}

func (c *ListCmd) Run(globals *Globals) error {
	result, err := globals.Client.List(c.ListID, c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type UserCmd struct {
	ScreenName string `arg:"" help:"Screen name without the @ prefix"`
}

func (c *UserCmd) Run(globals *Globals) error {
	user, err := globals.Client.User(xapi.NormalizeScreenName(c.ScreenName))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintUserProfile(user)
}

type UserPostsCmd struct {
	ScreenName string `arg:"" help:"Screen name without the @ prefix"`
	Max        int    `help:"Maximum tweets to fetch" default:"20"`
}

func (c *UserPostsCmd) Run(globals *Globals) error {
	result, err := globals.Client.UserPosts(xapi.NormalizeScreenName(c.ScreenName), c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type LikesCmd struct {
	ScreenName string `arg:"" help:"Screen name without the @ prefix"`
	Max        int    `help:"Maximum tweets to fetch" default:"20"`
}

func (c *LikesCmd) Run(globals *Globals) error {
	result, err := globals.Client.Likes(xapi.NormalizeScreenName(c.ScreenName), c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type FollowersCmd struct {
	ScreenName string `arg:"" help:"Screen name without the @ prefix"`
	Max        int    `help:"Maximum users to fetch" default:"20"`
}

func (c *FollowersCmd) Run(globals *Globals) error {
	users, err := globals.Client.Followers(xapi.NormalizeScreenName(c.ScreenName), c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintUsers(users)
}

type FollowingCmd struct {
	ScreenName string `arg:"" help:"Screen name without the @ prefix"`
	Max        int    `help:"Maximum users to fetch" default:"20"`
}

func (c *FollowingCmd) Run(globals *Globals) error {
	users, err := globals.Client.Following(xapi.NormalizeScreenName(c.ScreenName), c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintUsers(users)
}

type PostCmd struct {
	Text string `arg:"" help:"Post text"`
}

func (c *PostCmd) Run(globals *Globals) error {
	result, err := globals.Client.CreatePost(c.Text)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type DeleteCmd struct {
	ID string `arg:"" help:"Tweet ID or full status URL"`
}

func (c *DeleteCmd) Run(globals *Globals) error {
	result, err := globals.Client.DeletePost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type LikeCmd struct {
	ID string `arg:"" help:"Tweet ID or full status URL"`
}

func (c *LikeCmd) Run(globals *Globals) error {
	result, err := globals.Client.LikePost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type UnlikeCmd struct {
	ID string `arg:"" help:"Tweet ID or full status URL"`
}

func (c *UnlikeCmd) Run(globals *Globals) error {
	result, err := globals.Client.UnlikePost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type RetweetCmd struct {
	ID string `arg:"" help:"Tweet ID or full status URL"`
}

func (c *RetweetCmd) Run(globals *Globals) error {
	result, err := globals.Client.RetweetPost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type UnretweetCmd struct {
	ID string `arg:"" help:"Tweet ID or full status URL"`
}

func (c *UnretweetCmd) Run(globals *Globals) error {
	result, err := globals.Client.UnretweetPost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type BookmarkCmd struct {
	ID string `arg:"" help:"Tweet ID or full status URL"`
}

func (c *BookmarkCmd) Run(globals *Globals) error {
	result, err := globals.Client.BookmarkPost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type UnbookmarkCmd struct {
	ID string `arg:"" help:"Tweet ID or full status URL"`
}

func (c *UnbookmarkCmd) Run(globals *Globals) error {
	result, err := globals.Client.UnbookmarkPost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	return nil
}

type CompletionCmd struct {
	Shell string `arg:"" help:"Shell name" enum:"bash,zsh,fish,powershell"`
}

func (c *CompletionCmd) Run() error {
	_, _ = fmt.Fprintln(os.Stdout, "Shell completions are not wired yet.")
	_, _ = fmt.Fprintf(os.Stdout, "Use `x --help` for now while the generated completion command is being added for %s.\n", c.Shell)
	return nil
}
