package cli

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/auth"
	"github.com/dl-alexandre/X-CLI/internal/config"
	"github.com/dl-alexandre/X-CLI/internal/draft"
	"github.com/dl-alexandre/X-CLI/internal/model"
	"github.com/dl-alexandre/X-CLI/internal/output"
	"github.com/dl-alexandre/X-CLI/internal/profile"
	"github.com/dl-alexandre/X-CLI/internal/search"
	"github.com/dl-alexandre/X-CLI/internal/template"
	"github.com/dl-alexandre/X-CLI/internal/xapi"
	"github.com/mattn/go-isatty"
)

type CLI struct {
	Globals

	Status      StatusCmd      `cmd:"" help:"Show scaffold status and planned capabilities"`
	Doctor      DoctorCmd      `cmd:"" help:"Check auth, browser, and native transport readiness"`
	Login       LoginCmd       `cmd:"" help:"Authenticate with X using OAuth 2.0"`
	Logout      LogoutCmd      `cmd:"" help:"Remove stored authentication credentials"`
	Profiles    ProfilesCmd    `cmd:"" help:"List all configured profiles"`
	AnalyzeTXID AnalyzeTXIDCmd `cmd:"" name:"analyze-txid" help:"Analyze a captured txid JSONL corpus"`
	HarvestTXID HarvestTXIDCmd `cmd:"" name:"harvest-txid" help:"Extract salt samples from txid trace"`
	CompareTXID CompareTXIDCmd `cmd:"" name:"compare-txid" help:"Compare salts across operations to detect patterns"`
	Analytics   AnalyticsCmd   `cmd:"" help:"Analytics dashboard for engagement metrics"`
	Feed        FeedCmd        `cmd:"" help:"Fetch the home timeline"`
	Favorites   FavoritesCmd   `cmd:"" help:"Fetch bookmarked posts"`
	Search      SearchCmd      `cmd:"" help:"Search posts"`
	Tweet       TweetCmd       `cmd:"" help:"Show a post and replies"`
	List        ListCmd        `cmd:"" name:"list" help:"Fetch posts from a list"`
	User        UserCmd        `cmd:"" help:"Show a user profile"`
	UserPosts   UserPostsCmd   `cmd:"" name:"user-posts" help:"Fetch posts from a user"`
	UserSummary UserSummaryCmd `cmd:"" name:"user-summary" help:"Generate LLM-readable summary of a user's recent posts"`
	Likes       LikesCmd       `cmd:"" help:"Fetch posts liked by a user"`
	Followers   FollowersCmd   `cmd:"" help:"Fetch followers for a user"`
	Following   FollowingCmd   `cmd:"" help:"Fetch accounts followed by a user"`
	Mentions    MentionsCmd    `cmd:"" help:"Monitor mentions with optional filtering and notifications"`
	Post        PostCmd        `cmd:"" help:"Create a new post"`
	Draft       DraftCmd       `cmd:"" help:"Manage draft posts"`
	Schedule    ScheduleCmd    `cmd:"" help:"Manage scheduled posts"`
	Template    TemplateCmd    `cmd:"" help:"Manage post templates"`
	Delete      DeleteCmd      `cmd:"" help:"Delete one of your posts"`
	Like        LikeCmd        `cmd:"" help:"Like a post"`
	Unlike      UnlikeCmd      `cmd:"" help:"Unlike a post"`
	Retweet     RetweetCmd     `cmd:"" help:"Repost a post"`
	Unretweet   UnretweetCmd   `cmd:"" help:"Undo a repost"`
	Bookmark    BookmarkCmd    `cmd:"" help:"Bookmark a post"`
	Unbookmark  UnbookmarkCmd  `cmd:"" help:"Remove a bookmark"`
	Version     VersionCmd     `cmd:"" help:"Show version information"`
	Completion  CompletionCmd  `cmd:"" help:"Generate shell completion guidance"`
}

type Globals struct {
	ConfigFile    string `help:"Config file path" short:"c" env:"X_CONFIG"`
	Format        string `help:"Output format" default:"table" enum:"table,json,markdown" env:"X_FORMAT"`
	JSON          bool   `help:"Output as JSON (shorthand for --format json)" short:"j"`
	Markdown      bool   `help:"Output as Markdown (shorthand for --format markdown)" short:"m"`
	Profile       string `help:"Account profile to use" short:"p" env:"X_PROFILE"`
	AuthToken     string `help:"X auth token" env:"X_AUTH_TOKEN"`
	CT0           string `help:"X ct0 token" env:"X_CT0"`
	Proxy         string `help:"HTTP or SOCKS proxy" env:"X_PROXY"`
	TraceTxIDFile string `help:"Write x-client-transaction-id traces to JSONL" env:"X_BROWSER_TRACE_TXID_FILE"`
	TraceTxIDOps  string `help:"Comma-separated txid operations to trace" env:"X_BROWSER_TRACE_TXID_OPS"`
	Verbose       bool   `help:"Enable verbose logging" short:"v" env:"X_VERBOSE"`
	Debug         bool   `help:"Enable debug logging" env:"X_DEBUG"`
	NoColor       bool   `help:"Disable ANSI color in table output" env:"NO_COLOR"`

	Config *config.Config `kong:"-"`
	Client *xapi.Client   `kong:"-"`
}

func (g *Globals) AfterApply() error {
	if g.JSON {
		g.Format = "json"
	} else if g.Markdown {
		g.Format = "markdown"
	}

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
		Profile: g.Profile,
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

func (g *Globals) HasTemplate() bool {
	return false
}

func (g *Globals) PrintWithTemplate(data output.TemplateData) error {
	return nil
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

type LoginCmd struct {
	Force bool   `help:"Force re-authentication even if already logged in"`
	Name  string `help:"Profile name for multiple accounts"`
}

func (c *LoginCmd) Run(globals *Globals) error {
	profile := c.Name
	if profile == "" {
		profile = auth.DefaultProfile
	}

	storage := auth.NewTokenStorageWithProfile(profile)

	if !c.Force {
		if token, err := storage.Load(); err == nil && token != nil && !token.IsExpired() {
			fmt.Printf("Already authenticated (profile: %s).\n", profile)
			fmt.Printf("Token status: %s\n", auth.GetTokenStatus(token))
			fmt.Println("Use --force to re-authenticate.")
			return nil
		}
	}

	fmt.Printf("Starting OAuth 2.0 authentication flow (profile: %s)...\n", profile)
	fmt.Println()

	flow := auth.NewOAuthFlow()
	token, err := flow.Start()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := storage.Save(token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	fmt.Println("\n✓ Authentication successful!")
	fmt.Printf("Profile: %s\n", profile)
	fmt.Printf("Token stored securely in %s\n", storageLocation(storage))
	fmt.Printf("Token expires at: %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05"))

	return nil
}

type LogoutCmd struct {
	Name string `help:"Profile name to logout from"`
}

func (c *LogoutCmd) Run(globals *Globals) error {
	profile := c.Name
	if profile == "" {
		profile = auth.DefaultProfile
	}

	storage := auth.NewTokenStorageWithProfile(profile)

	if err := storage.Delete(); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}

	fmt.Printf("✓ Logged out successfully (profile: %s).\n", profile)
	fmt.Println("Stored credentials have been removed.")
	return nil
}

type ProfilesCmd struct {
	Export  ProfilesExportCmd  `cmd:"" help:"Export a profile to JSON file"`
	Import  ProfilesImportCmd  `cmd:"" help:"Import a profile from JSON file"`
	Backup  ProfilesBackupCmd  `cmd:"" help:"Backup all profiles to JSON file"`
	Restore ProfilesRestoreCmd `cmd:"" help:"Restore profiles from backup file"`
}

func (c *ProfilesCmd) Run(globals *Globals) error {
	profiles := auth.ListProfiles()

	fmt.Println("Available profiles:")
	for _, profile := range profiles {
		storage := auth.NewTokenStorageWithProfile(profile)
		token, err := storage.Load()

		status := "not authenticated"
		if err == nil && token != nil {
			status = auth.GetTokenStatus(token)
		}

		fmt.Printf("  - %s: %s\n", profile, status)
	}

	return nil
}

func storageLocation(storage *auth.TokenStorage) string {
	if storage.IsKeyringAvailable() {
		return "system keychain"
	}
	return "encrypted file (~/.config/x-cli/tokens.json.enc)"
}

type ProfilesExportCmd struct {
	Name   string `arg:"" help:"Profile name to export"`
	Output string `help:"Output file path" short:"o"`
}

func (c *ProfilesExportCmd) Run(globals *Globals) error {
	outputPath := c.Output
	if outputPath == "" {
		outputPath = fmt.Sprintf("%s-profile.json", c.Name)
	}

	exported, err := profile.ExportProfile(c.Name, profile.ExportOptions{OutputPath: outputPath})
	if err != nil {
		return fmt.Errorf("export profile: %w", err)
	}

	fmt.Printf("✓ Profile exported successfully!\n")
	fmt.Printf("  Profile: %s\n", exported.ProfileName)
	fmt.Printf("  Output: %s\n", outputPath)
	fmt.Printf("  Token status: %s\n", exported.Metadata.TokenStatus)
	return nil
}

type ProfilesImportCmd struct {
	Input     string `arg:"" help:"JSON file to import"`
	Overwrite bool   `help:"Overwrite existing profile"`
	Rename    bool   `help:"Rename profile if it exists"`
	NewName   string `help:"New profile name"`
}

func (c *ProfilesImportCmd) Run(globals *Globals) error {
	resolution := profile.ConflictSkip
	if c.Overwrite {
		resolution = profile.ConflictOverwrite
	} else if c.Rename {
		resolution = profile.ConflictRename
	}

	token, err := profile.ImportProfile(profile.ImportOptions{
		InputPath:          c.Input,
		ConflictResolution: resolution,
		NewName:            c.NewName,
	})
	if err != nil {
		return fmt.Errorf("import profile: %w", err)
	}

	fmt.Printf("✓ Profile imported successfully!\n")
	fmt.Printf("  Token expires at: %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05"))
	return nil
}

type ProfilesBackupCmd struct {
	Output string `arg:"" help:"Output file path"`
	All    bool   `help:"Backup all profiles"`
}

func (c *ProfilesBackupCmd) Run(globals *Globals) error {
	outputPath := c.Output
	if outputPath == "" {
		timestamp := time.Now().Format("2006-01-02-150405")
		outputPath = fmt.Sprintf("x-cli-backup-%s.json", timestamp)
	}

	backup, err := profile.BackupAllProfiles(outputPath)
	if err != nil {
		return fmt.Errorf("backup profiles: %w", err)
	}

	fmt.Printf("✓ Backup created successfully!\n")
	fmt.Printf("  Profiles: %d\n", backup.Count)
	fmt.Printf("  Output: %s\n", outputPath)
	return nil
}

type ProfilesRestoreCmd struct {
	Input     string `arg:"" help:"Backup file to restore"`
	Overwrite bool   `help:"Overwrite existing profiles"`
	Skip      bool   `help:"Skip existing profiles"`
}

func (c *ProfilesRestoreCmd) Run(globals *Globals) error {
	resolution := profile.ConflictRename
	if c.Overwrite {
		resolution = profile.ConflictOverwrite
	} else if c.Skip {
		resolution = profile.ConflictSkip
	}

	imported, err := profile.RestoreBackup(c.Input, resolution)
	if err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}

	fmt.Printf("✓ Backup restored successfully!\n")
	fmt.Printf("  Profiles imported: %d\n", len(imported))
	for _, name := range imported {
		fmt.Printf("  - %s\n", name)
	}
	return nil
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
	Max  int    `help:"Maximum posts to fetch" default:"20"`
}

func (c *FeedCmd) Run(globals *Globals) error {
	result, err := globals.Client.Feed(c.Type, c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type FavoritesCmd struct {
	Max int `help:"Maximum posts to fetch" default:"20"`
}

func (c *FavoritesCmd) Run(globals *Globals) error {
	result, err := globals.Client.Favorites(c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type SearchCmd struct {
	Query       string `arg:"" help:"Search query"`
	Type        string `help:"Search tab" default:"Top" enum:"Top,Latest,Photos,Videos"`
	Max         int    `help:"Maximum posts to fetch" default:"20"`
	FromUser    string `help:"Filter by author username (without @)" name:"from-user"`
	MinLikes    int    `help:"Minimum likes threshold" name:"min-likes"`
	MinRetweets int    `help:"Minimum reposts threshold" name:"min-retweets"`
	DateRange   string `help:"Date range filter (format: 'start,end' e.g., '2024-01-01,2024-01-31')" name:"date-range"`
	HasMedia    bool   `help:"Only show posts with media" name:"has-media"`
	IsReply     bool   `help:"Only show replies" name:"is-reply"`
	IsRetweet   bool   `help:"Only show reposts" name:"is-retweet"`
}

func (c *SearchCmd) Run(globals *Globals) error {
	filter, err := c.buildFilter()
	if err != nil {
		return err
	}

	modifiedQuery := c.Query
	if filter != nil && filter.HasFilters() {
		modifiedQuery = search.BuildSearchQuery(c.Query, filter)
	}

	result, err := globals.Client.Search(modifiedQuery, c.Type, c.Max)
	if err != nil {
		return err
	}

	if filter != nil && filter.HasFilters() {
		result.Tweets = search.Apply(result.Tweets, filter)
	}

	return globals.Printer("").PrintTimeline(result)
}

func (c *SearchCmd) buildFilter() (*search.Filter, error) {
	if c.FromUser == "" && c.MinLikes == 0 && c.MinRetweets == 0 &&
		c.DateRange == "" && !c.HasMedia && !c.IsReply && !c.IsRetweet {
		return nil, nil
	}

	filter := &search.Filter{
		FromUser:    search.NormalizeUsername(c.FromUser),
		MinLikes:    c.MinLikes,
		MinRetweets: c.MinRetweets,
		HasMedia:    c.HasMedia,
		IsReply:     c.IsReply,
		IsRetweet:   c.IsRetweet,
	}

	if c.DateRange != "" {
		start, end, err := search.ParseDateRange(c.DateRange)
		if err != nil {
			return nil, fmt.Errorf("invalid date range: %w", err)
		}
		filter.DateStart = start
		filter.DateEnd = end
	}

	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	return filter, nil
}

type TweetCmd struct {
	ID         string `arg:"" help:"Post ID or full status URL"`
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
	ListID string `arg:"" help:"X list ID"`
	Max    int    `help:"Maximum posts to fetch" default:"20"`
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
	Max        int    `help:"Maximum posts to fetch" default:"20"`
}

func (c *UserPostsCmd) Run(globals *Globals) error {
	result, err := globals.Client.UserPosts(xapi.NormalizeScreenName(c.ScreenName), c.Max)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintTimeline(result)
}

type UserSummaryCmd struct {
	ScreenName string `arg:"" help:"Screen name without the @ prefix"`
	Max        int    `help:"Maximum posts to analyze" default:"20"`
}

func (c *UserSummaryCmd) Run(globals *Globals) error {
	user, err := globals.Client.User(xapi.NormalizeScreenName(c.ScreenName))
	if err != nil {
		return err
	}

	result, err := globals.Client.UserPosts(xapi.NormalizeScreenName(c.ScreenName), c.Max)
	if err != nil {
		return err
	}

	return globals.Printer("").PrintUserSummary(user, result.Tweets)
}

type LikesCmd struct {
	ScreenName string `arg:"" help:"Screen name without the @ prefix"`
	Max        int    `help:"Maximum posts to fetch" default:"20"`
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
	Text     string            `arg:"" help:"Post text" optional:""`
	Thread   bool              `help:"Split long text into a numbered thread"`
	File     string            `help:"Image/video file to attach"`
	TextFile string            `help:"Read post text from file"`
	Template string            `help:"Use a saved template"`
	Vars     map[string]string `help:"Template variables (key=value)"`
}

func (c *PostCmd) Run(globals *Globals) error {
	text := c.Text

	if c.Template != "" {
		store, err := template.NewTemplateStore()
		if err != nil {
			return fmt.Errorf("template store: %w", err)
		}

		tmpl, err := store.Get(c.Template)
		if err != nil {
			return fmt.Errorf("get template: %w", err)
		}

		rendered, err := tmpl.Render(c.Vars)
		if err != nil {
			return fmt.Errorf("render template: %w", err)
		}

		text = rendered
	}

	if c.TextFile != "" {
		data, err := os.ReadFile(c.TextFile)
		if err != nil {
			return fmt.Errorf("read text file: %w", err)
		}
		text = string(data)
	}

	text = strings.TrimSpace(text)

	if c.File != "" {
		if text == "" {
			return fmt.Errorf("post text cannot be empty when attaching media")
		}

		mediaResult, err := globals.Client.UploadMedia(c.File)
		if err != nil {
			return fmt.Errorf("upload media: %w", err)
		}

		result, err := globals.Client.CreatePostWithMedia(text, mediaResult.MediaIDString)
		if err != nil {
			return err
		}
		return globals.Printer("").PrintActionResult(result)
	}

	if text == "" {
		return fmt.Errorf("post text cannot be empty")
	}

	if c.Thread {
		results, err := globals.Client.CreateThread(text)
		if err != nil {
			return err
		}
		for _, result := range results {
			if err := globals.Printer("").PrintActionResult(result); err != nil {
				return err
			}
		}
		return nil
	}

	result, err := globals.Client.CreatePost(text)
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type DeleteCmd struct {
	ID string `arg:"" help:"Post ID or full status URL"`
}

func (c *DeleteCmd) Run(globals *Globals) error {
	result, err := globals.Client.DeletePost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type LikeCmd struct {
	ID    string `arg:"" help:"Post ID or full status URL" optional:""`
	Batch string `help:"File containing post IDs (one per line)"`
}

func (c *LikeCmd) Run(globals *Globals) error {
	if c.Batch != "" {
		return processBatch(globals, c.Batch, globals.Client.LikePost, "like")
	}

	result, err := globals.Client.LikePost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type UnlikeCmd struct {
	ID    string `arg:"" help:"Post ID or full status URL" optional:""`
	Batch string `help:"File containing post IDs (one per line)"`
}

func (c *UnlikeCmd) Run(globals *Globals) error {
	if c.Batch != "" {
		return processBatch(globals, c.Batch, globals.Client.UnlikePost, "unlike")
	}

	result, err := globals.Client.UnlikePost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type RetweetCmd struct {
	ID    string `arg:"" help:"Post ID or full status URL" optional:""`
	Batch string `help:"File containing post IDs (one per line)"`
}

func (c *RetweetCmd) Run(globals *Globals) error {
	if c.Batch != "" {
		return processBatch(globals, c.Batch, globals.Client.RetweetPost, "retweet")
	}

	result, err := globals.Client.RetweetPost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type UnretweetCmd struct {
	ID    string `arg:"" help:"Post ID or full status URL" optional:""`
	Batch string `help:"File containing post IDs (one per line)"`
}

func (c *UnretweetCmd) Run(globals *Globals) error {
	if c.Batch != "" {
		return processBatch(globals, c.Batch, globals.Client.UnretweetPost, "unretweet")
	}

	result, err := globals.Client.UnretweetPost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type BookmarkCmd struct {
	ID    string `arg:"" help:"Post ID or full status URL" optional:""`
	Batch string `help:"File containing post IDs (one per line)"`
}

func (c *BookmarkCmd) Run(globals *Globals) error {
	if c.Batch != "" {
		return processBatch(globals, c.Batch, globals.Client.BookmarkPost, "bookmark")
	}

	result, err := globals.Client.BookmarkPost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

type UnbookmarkCmd struct {
	ID    string `arg:"" help:"Post ID or full status URL" optional:""`
	Batch string `help:"File containing post IDs (one per line)"`
}

func (c *UnbookmarkCmd) Run(globals *Globals) error {
	if c.Batch != "" {
		return processBatch(globals, c.Batch, globals.Client.UnbookmarkPost, "unbookmark")
	}

	result, err := globals.Client.UnbookmarkPost(xapi.NormalizeTweetID(c.ID))
	if err != nil {
		return err
	}
	return globals.Printer("").PrintActionResult(result)
}

func processBatch(globals *Globals, filePath string, action func(string) (model.ActionResult, error), actionName string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read batch file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	ids := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			ids = append(ids, xapi.NormalizeTweetID(line))
		}
	}

	if len(ids) == 0 {
		return fmt.Errorf("no post IDs found in batch file")
	}

	fmt.Printf("Processing %d posts...\n", len(ids))

	successCount := 0
	failCount := 0

	for i, id := range ids {
		fmt.Printf("[%d/%d] %s %s... ", i+1, len(ids), actionName, id)

		_, err := action(id)
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
			failCount++
		} else {
			fmt.Println("OK")
			successCount++
		}

		if i < len(ids)-1 {
			delay := time.Duration(1500+rand.Intn(1000)) * time.Millisecond
			time.Sleep(delay)
		}
	}

	fmt.Printf("\nCompleted: %d success, %d failed\n", successCount, failCount)
	return nil
}

type DraftCmd struct {
	Save   DraftSaveCmd   `cmd:"" help:"Save a draft post"`
	List   DraftListCmd   `cmd:"" help:"List all drafts"`
	Post   DraftPostCmd   `cmd:"" help:"Post a draft"`
	Edit   DraftEditCmd   `cmd:"" help:"Edit a draft"`
	Delete DraftDeleteCmd `cmd:"" help:"Delete a draft"`
}

type DraftSaveCmd struct {
	Text string `arg:"" help:"Draft text"`
}

func (c *DraftSaveCmd) Run(globals *Globals) error {
	store, err := draft.NewDraftStore()
	if err != nil {
		return fmt.Errorf("draft store: %w", err)
	}

	d, err := store.Save(c.Text)
	if err != nil {
		return fmt.Errorf("save draft: %w", err)
	}

	fmt.Printf("Draft saved: %s\n", d.ID)
	return nil
}

type DraftListCmd struct{}

func (c *DraftListCmd) Run(globals *Globals) error {
	store, err := draft.NewDraftStore()
	if err != nil {
		return fmt.Errorf("draft store: %w", err)
	}

	drafts := store.List()
	if len(drafts) == 0 {
		fmt.Println("No drafts saved.")
		return nil
	}

	fmt.Printf("Drafts (%d):\n\n", len(drafts))
	for i, d := range drafts {
		preview := d.Text
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}
		fmt.Printf("%d. [%s] %s\n", i+1, d.ID, preview)
		fmt.Printf("   Updated: %s\n\n", d.UpdatedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

type DraftPostCmd struct {
	ID string `arg:"" help:"Draft ID to post"`
}

func (c *DraftPostCmd) Run(globals *Globals) error {
	store, err := draft.NewDraftStore()
	if err != nil {
		return fmt.Errorf("draft store: %w", err)
	}

	d, err := store.Get(c.ID)
	if err != nil {
		return fmt.Errorf("get draft: %w", err)
	}

	result, err := globals.Client.CreatePost(d.Text)
	if err != nil {
		return fmt.Errorf("post draft: %w", err)
	}

	if err := store.Delete(c.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to delete draft: %v\n", err)
	}

	return globals.Printer("").PrintActionResult(result)
}

type DraftEditCmd struct {
	ID   string `arg:"" help:"Draft ID to edit"`
	Text string `arg:"" help:"New draft text"`
}

func (c *DraftEditCmd) Run(globals *Globals) error {
	store, err := draft.NewDraftStore()
	if err != nil {
		return fmt.Errorf("draft store: %w", err)
	}

	d, err := store.Update(c.ID, c.Text)
	if err != nil {
		return fmt.Errorf("update draft: %w", err)
	}

	fmt.Printf("Draft updated: %s\n", d.ID)
	return nil
}

type DraftDeleteCmd struct {
	ID string `arg:"" help:"Draft ID to delete"`
}

func (c *DraftDeleteCmd) Run(globals *Globals) error {
	store, err := draft.NewDraftStore()
	if err != nil {
		return fmt.Errorf("draft store: %w", err)
	}

	if err := store.Delete(c.ID); err != nil {
		return fmt.Errorf("delete draft: %w", err)
	}

	fmt.Printf("Draft deleted: %s\n", c.ID)
	return nil
}

type TemplateCmd struct {
	Save    TemplateSaveCmd    `cmd:"" help:"Save a template"`
	List    TemplateListCmd    `cmd:"" help:"List all templates"`
	Show    TemplateShowCmd    `cmd:"" help:"Show template content"`
	Delete  TemplateDeleteCmd  `cmd:"" help:"Delete a template"`
	Preview TemplatePreviewCmd `cmd:"" help:"Preview a template with variables"`
	Export  TemplateExportCmd  `cmd:"" help:"Export a template"`
	Import  TemplateImportCmd  `cmd:"" help:"Import a template"`
}

type TemplateSaveCmd struct {
	Name        string            `arg:"" help:"Template name"`
	File        string            `help:"Read template content from file"`
	Text        string            `help:"Template content (if not using file)"`
	Category    string            `help:"Template category (e.g., product-launch, daily-update)"`
	Description string            `help:"Template description"`
	Vars        map[string]string `help:"Default variables (key=value)"`
}

func (c *TemplateSaveCmd) Run(globals *Globals) error {
	content := c.Text
	if c.File != "" {
		data, err := os.ReadFile(c.File)
		if err != nil {
			return fmt.Errorf("read template file: %w", err)
		}
		content = string(data)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("template content cannot be empty")
	}

	store, err := template.NewTemplateStore()
	if err != nil {
		return fmt.Errorf("template store: %w", err)
	}

	tmpl, err := store.Save(c.Name, content, c.Category, c.Description, c.Vars)
	if err != nil {
		return fmt.Errorf("save template: %w", err)
	}

	fmt.Printf("Template saved: %s\n", tmpl.Name)
	return nil
}

type TemplateListCmd struct {
	Category string `help:"Filter by category"`
}

func (c *TemplateListCmd) Run(globals *Globals) error {
	store, err := template.NewTemplateStore()
	if err != nil {
		return fmt.Errorf("template store: %w", err)
	}

	var templates []*template.Template
	if c.Category != "" {
		templates = store.ListByCategory(c.Category)
	} else {
		templates = store.List()
	}

	if len(templates) == 0 {
		fmt.Println("No templates saved.")
		return nil
	}

	fmt.Printf("Templates (%d):\n\n", len(templates))
	for i, t := range templates {
		preview := t.Content
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}
		fmt.Printf("%d. %s", i+1, t.Name)
		if t.Category != "" {
			fmt.Printf(" [%s]", t.Category)
		}
		fmt.Println()
		fmt.Printf("   %s\n", preview)
		if t.Description != "" {
			fmt.Printf("   %s\n", t.Description)
		}
		fmt.Println()
	}

	return nil
}

type TemplateShowCmd struct {
	Name string `arg:"" help:"Template name"`
}

func (c *TemplateShowCmd) Run(globals *Globals) error {
	store, err := template.NewTemplateStore()
	if err != nil {
		return fmt.Errorf("template store: %w", err)
	}

	tmpl, err := store.Get(c.Name)
	if err != nil {
		return fmt.Errorf("get template: %w", err)
	}

	fmt.Printf("Template: %s\n", tmpl.Name)
	if tmpl.Category != "" {
		fmt.Printf("Category: %s\n", tmpl.Category)
	}
	if tmpl.Description != "" {
		fmt.Printf("Description: %s\n", tmpl.Description)
	}
	fmt.Printf("Created: %s\n", tmpl.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Updated: %s\n", tmpl.UpdatedAt.Format("2006-01-02 15:04"))
	fmt.Println()
	fmt.Println("Content:")
	fmt.Println(tmpl.Content)

	if len(tmpl.Variables) > 0 {
		fmt.Println()
		fmt.Println("Default Variables:")
		for k, v := range tmpl.Variables {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	return nil
}

type TemplateDeleteCmd struct {
	Name string `arg:"" help:"Template name"`
}

func (c *TemplateDeleteCmd) Run(globals *Globals) error {
	store, err := template.NewTemplateStore()
	if err != nil {
		return fmt.Errorf("template store: %w", err)
	}

	if err := store.Delete(c.Name); err != nil {
		return fmt.Errorf("delete template: %w", err)
	}

	fmt.Printf("Template deleted: %s\n", c.Name)
	return nil
}

type TemplatePreviewCmd struct {
	Name string            `arg:"" help:"Template name"`
	Vars map[string]string `help:"Template variables (key=value)"`
}

func (c *TemplatePreviewCmd) Run(globals *Globals) error {
	store, err := template.NewTemplateStore()
	if err != nil {
		return fmt.Errorf("template store: %w", err)
	}

	tmpl, err := store.Get(c.Name)
	if err != nil {
		return fmt.Errorf("get template: %w", err)
	}

	rendered, err := tmpl.Render(c.Vars)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	fmt.Println(rendered)
	return nil
}

type TemplateExportCmd struct {
	Name string `arg:"" help:"Template name"`
	File string `help:"Write to file (default: stdout)"`
}

func (c *TemplateExportCmd) Run(globals *Globals) error {
	store, err := template.NewTemplateStore()
	if err != nil {
		return fmt.Errorf("template store: %w", err)
	}

	data, err := store.Export(c.Name)
	if err != nil {
		return fmt.Errorf("export template: %w", err)
	}

	if c.File != "" {
		return os.WriteFile(c.File, data, 0644)
	}

	fmt.Println(string(data))
	return nil
}

type TemplateImportCmd struct {
	File string `arg:"" help:"Template file to import"`
}

func (c *TemplateImportCmd) Run(globals *Globals) error {
	data, err := os.ReadFile(c.File)
	if err != nil {
		return fmt.Errorf("read template file: %w", err)
	}

	store, err := template.NewTemplateStore()
	if err != nil {
		return fmt.Errorf("template store: %w", err)
	}

	tmpl, err := store.Import(data)
	if err != nil {
		return fmt.Errorf("import template: %w", err)
	}

	fmt.Printf("Template imported: %s\n", tmpl.Name)
	return nil
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
