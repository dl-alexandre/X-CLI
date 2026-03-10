# X-CLI

[![Go Version](https://img.shields.io/github/go-mod/go-version/dl-alexandre/X-CLI)](https://github.com/dl-alexandre/X-CLI)
[![Release](https://img.shields.io/github/v/release/dl-alexandre/X-CLI)](https://github.com/dl-alexandre/X-CLI/releases/latest)
[![License](https://img.shields.io/github/license/dl-alexandre/X-CLI)](License)
[![Go Report Card](https://goreportcard.com/badge/github.com/dl-alexandre/X-CLI)](https://goreportcard.com/report/github.com/dl-alexandre/X-CLI)

`x` is a terminal-first CLI for X built in Go. A professional-grade, AI-friendly tool for power users and automation workflows.

## Features

- **Terminal-first CLI** for X
- **Native Go implementation** (10x faster than browser automation)
- **25+ CLI commands** (read + write + diagnostics)
- **OAuth 2.0 PKCE authentication** with secure keychain storage
- **AI-friendly outputs** (JSON, Markdown)
- **Thread posting** with smart word splitting
- **URL shortening** for thread posts (TinyURL, is.gd, v.gd, custom)
- **Template system** with variable substitution
- **Media uploads** (images <5MB)
- **Intelligent rate limiting** with auto-retry
- **Profile management** for multiple accounts
- **MCP integration** for Claude Desktop/OpenCode
- **No API keys required**

---

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap dl-alexandre/homebrew-tap
brew install x-cli
```

### Snap (Linux)

```bash
sudo snap install x-cli
```

### Docker

```bash
docker pull ghcr.io/dl-alexandre/x-cli:latest
docker run --rm -v ~/.config/x-cli:/root/.config/x-cli x-cli x --help
```

### Binary Download

Download from [Releases](https://github.com/dl-alexandre/X-CLI/releases/latest):
- Linux: `x-linux-amd64`, `x-linux-arm64`
- macOS: `x-darwin-amd64`, `x-darwin-arm64`
- Windows: `x-windows-amd64.exe`

### From Source

```bash
go install github.com/dl-alexandre/X-CLI/cmd/x@latest
```

---

## Quick Start

### Authentication

```bash
# Authenticate with OAuth 2.0
x login

# Check authentication status
x doctor
```

### Basic Usage

```bash
# View your timeline
x feed

# Search posts
x search "golang" --type Latest

# View a user's profile
x user navalcadet

# View a specific post
x post 1234567890
```

---

## AI-Friendly Outputs

All commands support JSON and Markdown output for LLM consumption:

```bash
# JSON output (for data processing)
x user navalcadet -j

# Markdown output (for chat windows)
x user-posts navalcadet -m

# User summary (optimized for LLMs)
x user-summary navalcadet -j
```

### User Summary Command

Generate an LLM-readable summary of a user's recent posts:

```bash
x user-summary @username --max 20 -j
```

Returns:
- Top hashtags
- Top mentions
- Average engagement metrics
- Recent themes
- Activity patterns

---

## Custom Output Templates

X-CLI supports custom output templates using Go template syntax for flexible output formatting:

### Basic Usage

```bash
# Inline template for user output
x user navalcadet --template "{{.User.Name}}: {{.User.FollowersCount}} followers"

# Template from file
x search "golang" --template tweet-format.txt

# Use built-in preset
x user navalcadet --template @user.simple
```

### Template Variables

**User Data:**
| Variable | Description |
|----------|-------------|
| `{{.User.Name}}` | Display name |
| `{{.User.ScreenName}}` | Username (without @) |
| `{{.User.Bio}}` | Bio text |
| `{{.User.FollowersCount}}` | Follower count |
| `{{.User.FollowingCount}}` | Following count |
| `{{.User.TweetsCount}}` | Post count |
| `{{.User.Location}}` | Location |
| `{{.User.URL}}` | Profile URL |
| `{{.User.Verified}}` | Verified status (bool) |

**Tweet Data:**
| Variable | Description |
|----------|-------------|
| `{{.Tweet.ID}}` | Post ID |
| `{{.Tweet.Text}}` | Post text |
| `{{.Tweet.CreatedAt}}` | Creation timestamp |
| `{{.Tweet.Author.Name}}` | Author display name |
| `{{.Tweet.Author.ScreenName}}` | Author username |
| `{{.Tweet.Metrics.Likes}}` | Like count |
| `{{.Tweet.Metrics.Retweets}}` | Repost count |
| `{{.Tweet.Metrics.Replies}}` | Reply count |
| `{{.Tweet.Metrics.Views}}` | View count |

**Multiple Tweets:**
| Variable | Description |
|----------|-------------|
| `{{.Tweets}}` | Slice of tweets |
| `{{range .Tweets}}` | Iterate over tweets |
| `{{count .Tweets}}` | Tweet count |

### Template Functions

**Text Functions:**
| Function | Description | Example |
|----------|-------------|---------|
| `upper` | Uppercase | `{{.User.Name \| upper}}` |
| `lower` | Lowercase | `{{.User.ScreenName \| lower}}` |
| `title` | Title case | `{{.User.Name \| title}}` |
| `trim` | Trim whitespace | `{{.User.Bio \| trim}}` |
| `truncate` | Truncate text | `{{truncate .Tweet.Text 50}}` |
| `replace` | Replace text | `{{.Tweet.Text \| replace "old" "new"}}` |

**Number Functions:**
| Function | Description | Example |
|----------|-------------|---------|
| `k` | Format as K/M | `{{.User.FollowersCount \| k}}` → 10.5K |
| `num` | Format number | `{{.Tweet.Metrics.Likes \| num}}` |
| `add` | Add numbers | `{{add 1 2}}` → 3 |

**Date Functions:**
| Function | Description | Example |
|----------|-------------|---------|
| `date` | Format date | `{{.Tweet.CreatedAt \| date "2006-01-02"}}` |
| `reltime` | Relative time | `{{.Tweet.CreatedAt \| reltime}}` → 2h |

**Extraction Functions:**
| Function | Description | Example |
|----------|-------------|---------|
| `hashtag` | Extract hashtags | `{{range .Tweet.Text \| hashtag}}{{.}}{{end}}` |
| `mention` | Extract mentions | `{{range .Tweet.Text \| mention}}{{.}}{{end}}` |
| `url` | Extract URLs | `{{range .Tweet.Text \| url}}{{.}}{{end}}` |

**Utility Functions:**
| Function | Description | Example |
|----------|-------------|---------|
| `default` | Default value | `{{.User.Bio \| default "No bio"}}` |
| `coalesce` | First non-empty | `{{coalesce .User.Name .User.ScreenName}}` |
| `contains` | Check substring | `{{.Tweet.Text \| contains "golang"}}` |
| `count` | Count items | `{{count .Tweets}}` |

### Built-in Presets

Use presets with `@preset-name`:

| Preset | Output |
|--------|--------|
| `@user.simple` | `Name (@username)` |
| `@user.bio` | `Name (@username)\nBio` |
| `@user.stats` | `Name: 10.5K followers, 500 following` |
| `@tweet.simple` | `Tweet text` |
| `@tweet.full` | `@username: Tweet text (100 likes)` |
| `@tweet.metrics` | `❤️ 100 🔄 50 💬 10 👁️ 1000` |
| `@tweets.list` | List of tweets |
| `@tweets.csv` | CSV format |

### Examples

**Custom user format:**
```bash
x user navalcadet -t "{{.User.Name}} (@{{.User.ScreenName}}) - {{.User.FollowersCount | k}} followers"
# Output: John Doe (@johndoe) - 10.5K followers
```

**Search with custom format:**
```bash
x search "golang" -t "{{range .Tweets}}• {{.Text | truncate 60}} ({{.Metrics.Likes}} likes)\n{{end}}"
```

**Export to CSV:**
```bash
x user-posts navalcadet -t "@tweets.csv" > posts.csv
```

**Relative timestamps:**
```bash
x search "news" -t "[{{.Tweet.CreatedAt | reltime}}] @{{.Tweet.Author.ScreenName}}: {{.Tweet.Text}}"
# Output: [2h] @newsite: Breaking news...
```

---

## Thread Posting

Post long-form content as numbered threads with automatic splitting:

```bash
# Post a thread from text
x post --thread "This is a very long text that will be automatically split into multiple posts at word boundaries..."

# Post a thread from file
x post --thread --text-file article.txt

# Each post gets numbered: (1/5), (2/5), etc.
```

**Features:**
- Smart word boundary detection
- UTF-8 aware character counting
- 1-2 second randomized jitter between posts
- Automatic reply chaining

---

## URL Shortening

Automatically shorten URLs in thread posts to save characters:

```bash
# Enable URL shortening in config
# ~/.config/x/config.yaml:
url_shortener:
  enabled: true
  service: tinyurl  # tinyurl, isgd, vgd, custom
  timeout: 10s

# Post a thread with shortened URLs
x post --thread "Check out this article: https://example.com/very/long/url/path"
```

**Supported Services:**
- `tinyurl` - TinyURL (default)
- `isgd` - is.gd
- `vgd` - v.gd
- `custom` - Custom API endpoint

**Custom API:**
```yaml
url_shortener:
  enabled: true
  service: custom
  custom_api_url: "https://your-shortener.com/api"
  timeout: 10s
```

The custom API should accept `?url=` parameter and return either:
- JSON: `{"short_url": "https://..."}`
- Plain text: `https://...`

**Features:**
- Automatic URL detection in text
- Preserves URL functionality
- Handles URLs at split boundaries
- Configurable timeout
- Fallback to original URL on error

---

## Advanced Search Filters

X-CLI supports powerful search filters to narrow down results:

### Filter Options

| Flag | Description | Example |
|------|-------------|---------|
| `--from-user` | Filter by author | `--from-user golang` |
| `--min-likes` | Minimum likes threshold | `--min-likes 100` |
| `--min-reposts` | Minimum reposts | `--min-reposts 50` |
| `--date-range` | Date range (start,end) | `--date-range "2024-01-01,2024-01-31"` |
| `--has-media` | Only posts with media | `--has-media` |
| `--is-reply` | Only replies | `--is-reply` |
| `--is-retweet` | Only reposts | `--is-retweet` |

### Examples

```bash
# Find popular posts from a specific user
x search "golang" --from-user golang --min-likes 100

# Find viral posts in a date range
x search "AI" --min-likes 1000 --min-reposts 500 --date-range "2024-01-01,2024-03-31"

# Find media posts from a user
x search "photo" --from-user navalcadet --has-media

# Find replies in a conversation
x search "help" --is-reply --type Latest

# Combine multiple filters
x search "tutorial" --from-user golang --min-likes 50 --has-media --date-range "2024-01-01,"
```

### Date Format Support

The `--date-range` flag supports multiple formats:
- `YYYY-MM-DD` (recommended)
- `YYYY/MM/DD`
- `Jan 2, 2006`
- `January 2, 2006`

---

## Media Uploads

Attach images and videos to your posts:

### Image Uploads

```bash
# Post with image
x post "Check out this photo!" --file photo.jpg

# Supported formats: JPEG, PNG, GIF
# Max size: 5MB
```

### Video Uploads

```bash
# Post with video (chunked upload with progress)
x post "Check out this video!" --file video.mp4

# Supported formats: MP4, MOV
# Max size: 512MB
```

**Video Upload Features:**
- **Chunked upload** - 4MB chunks for reliable large file uploads
- **Progress bar** - Real-time upload progress with ETA
- **Resume support** - Interrupted uploads can be resumed
- **State persistence** - Upload state saved to `~/.config/x/video-uploads/`
- **Auto-retry** - Automatic retry on network errors

**Progress Display:**
```
[████████████░░░░░░░░░░░░░░░░░░░░] 40.0% 2.5 MB/s ETA: 24s
```

**Video Requirements:**
- Format: MP4 (H.264) or MOV
- Max resolution: 1920x1080 (1080p)
- Max duration: 140 seconds
- Max file size: 512MB

**Resume Interrupted Upload:**
```bash
# If upload is interrupted, simply run the same command again
x post "My video" --file video.mp4
# Upload will resume from where it left off
```

**Clean up expired upload states:**
```bash
# Expired upload states are automatically cleaned up
# Or manually check state:
ls ~/.config/x/video-uploads/
```

---

## Post Scheduling

Schedule posts to be posted at a future time:

### Basic Usage

```bash
# Schedule a post for a specific time
x schedule save "Big announcement coming!" --time "2024-01-15 09:00"

# Schedule using natural language
x schedule save "Good morning!" --time "tomorrow at 9am"
x schedule save "Meeting reminder" --time "in 2 hours"
x schedule save "Weekly update" --time "next monday at 10am"

# List scheduled posts
x schedule list

# List all posts (including posted/cancelled)
x schedule list --all

# Cancel a scheduled post
x schedule cancel <id>

# Post a scheduled post immediately
x schedule post <id>
```

### Time Format Support

X-CLI supports multiple time formats:

**Absolute Times:**
- `2024-01-15 09:00` - Date and time
- `2024-01-15 09:00:00` - With seconds
- `01/15/2024 09:00` - US date format

**Relative Times:**
- `in 30 minutes`
- `in 2 hours`
- `in 1 day`
- `in 2 weeks`
- `30 minutes from now`

**Natural Language:**
- `now`
- `tomorrow`
- `tomorrow at 9am`
- `tomorrow at 2pm`
- `today at 5pm`
- `at 5pm` (defaults to today or tomorrow if past)
- `next monday`
- `next friday`

### Storage

Scheduled posts are stored in:
- `~/.config/x-cli/scheduled.json`

Each scheduled post includes:
- Unique ID
- Post text
- Scheduled time
- Creation time
- Status (pending, posted, cancelled)

### Status Icons

When listing scheduled posts:
- `⏳` - Pending (waiting to be posted)
- `✓` - Posted (successfully posted)
- `✗` - Cancelled (cancelled before posting)

### Automation

For automated posting, you can run a scheduler:

```bash
# Check and post due posts (can be run via cron)
# This is a placeholder for future daemon mode
*/5 * * * * x schedule process-due
```

---

## Template System

Create and manage reusable post templates with variable substitution:

### Basic Usage

```bash
# Save a template
x template save daily-update --text "Daily standup complete! {{.Date}} - {{.Summary}}"

# List all templates
x template list

# Show template details
x template show daily-update

# Preview a template with variables
x template preview daily-update --vars Summary="Shipped new feature"

# Post from a template
x post --template daily-update --vars Summary="Fixed critical bug"
```

### Template Variables

Templates support Go template syntax with built-in and custom variables:

**Built-in Variables:**
| Variable | Description | Example |
|----------|-------------|---------|
| `{{.Date}}` | Current date | 2024-01-15 |
| `{{.Time}}` | Current time | 14:30:00 |
| `{{.DateTime}}` | Date and time | 2024-01-15 14:30:00 |
| `{{.Year}}` | Year | 2024 |
| `{{.Month}}` | Month | 01 |
| `{{.Day}}` | Day | 15 |
| `{{.Weekday}}` | Day of week | Monday |
| `{{.Hour}}` | Hour | 14 |
| `{{.Minute}}` | Minute | 30 |
| `{{.Timestamp}}` | Unix timestamp | 1705325400 |
| `{{.ISODate}}` | ISO date | 2024-01-15 |
| `{{.ISODateTime}}` | ISO datetime | 2024-01-15T14:30:00Z |

**Custom Variables:**
```bash
# Define custom variables when saving
x template save product-launch \
  --text "Introducing {{.ProductName}}! {{.Description}}" \
  --vars ProductName="X-CLI" \
  --vars Description="A powerful CLI for X"

# Override defaults when posting
x post --template product-launch \
  --vars ProductName="NewTool" \
  --vars Description="An amazing new product"
```

### Template Categories

Organize templates by category:

```bash
# Save with category
x template save morning-standup \
  --category daily \
  --description "Morning standup update template" \
  --text "Good morning! {{.Date}} - {{.Update}}"

# List by category
x template list --category daily
```

### Import/Export

Share templates between machines:

```bash
# Export a template
x template export daily-update > daily-update.json

# Import a template
x template import daily-update.json

# Export to file
x template export daily-update --file exported.json
```

### Template Storage

Templates are stored in: `~/.config/x-cli/templates/`

Each template is a JSON file with:
- Name
- Content
- Category
- Description
- Default variables
- Created/Updated timestamps

### Example Templates

**Product Launch:**
```bash
x template save product-launch \
  --category announcements \
  --text "🚀 Introducing {{.ProductName}}!

{{.Description}}

Available now: {{.Link}}

#{{.Hashtag}}" \
  --vars Hashtag="ProductLaunch"
```

**Daily Update:**
```bash
x template save daily-update \
  --category daily \
  --text "📊 Daily Update - {{.Date}}

✅ Completed: {{.Completed}}
🔄 In Progress: {{.InProgress}}
🎯 Tomorrow: {{.Tomorrow}}"
```

**Weekly Recap:**
```bash
x template save weekly-recap \
  --category weekly \
  --text "📅 Week {{.WeekNumber}} Recap

Highlights:
{{.Highlights}}

Stats:
- Commits: {{.Commits}}
- PRs: {{.PRs}}
- Reviews: {{.Reviews}}"
```

---

## Profile Management

Manage multiple X accounts:

```bash
# Login with a specific profile
x login --profile work

# Use a specific profile for commands
x feed --profile work

# List all profiles
x profiles

# Logout from a specific profile
x logout --profile work
```

### Profile Export/Import

Export and import profiles for backup or migration:

```bash
# Export a single profile
x profiles export work --output work-profile.json

# Import a profile
x profiles import work-profile.json

# Import with conflict resolution
x profiles import work-profile.json --overwrite  # Overwrite existing
x profiles import work-profile.json --rename      # Rename if exists
x profiles import work-profile.json --new-name work-backup  # Import with new name

# Backup all profiles
x profiles backup x-cli-backup.json

# Restore from backup
x profiles restore x-cli-backup.json --overwrite  # Overwrite existing
x profiles restore x-cli-backup.json --skip       # Skip existing
```

**Export Format:**
```json
{
  "version": "1.0",
  "format": "x-cli-profile",
  "exported_at": "2024-01-15T10:30:00Z",
  "profile_name": "work",
  "token": {
    "encrypted_data": "...",
    "nonce": "...",
    "key_hint": "hostname"
  },
  "metadata": {
    "name": "work",
    "created_at": "2024-01-10T08:00:00Z",
    "last_used": "2024-01-15T10:30:00Z",
    "token_status": "valid for 1h30m",
    "source": "hostname"
  }
}
```

**Security:**
- Tokens are encrypted with AES-256-GCM
- Encryption key derived from hostname + home directory
- Export files should be stored securely
- Import validates file format and integrity

**Storage:**
- macOS: Keychain Access
- Windows: Credential Manager
- Linux: Secret Service (libsecret)
- Headless: Encrypted file fallback
## Rate Limit Intelligence

X-CLI handles X's strict rate limits automatically:

- **Auto-detection** of 429 responses
- **Real-time countdown**: `Rate limited. Retrying in 45s... 44s...`
- **Automatic retry** with configurable max attempts
- **Jitter support** for anti-spam measures

---

## Mention Monitoring

Monitor mentions of your account with real-time notifications and filtering:

### Basic Usage

```bash
# Show mentions from the last hour
x mentions --since 1h

# Show mentions from the last 24 hours
x mentions --since 24h

# Show mentions from the last 7 days
x mentions --since 7d
```

### Continuous Monitoring

```bash
# Start continuous monitoring (polls every 60 seconds)
x mentions --watch

# Custom polling interval (30 seconds)
x mentions --watch --interval 30s

# Monitor with desktop notifications
x mentions --watch --notify
```

### Filtering Mentions

```bash
# Filter by keywords
x mentions --filter "urgent,help,support"

# Filter by users
x mentions --from-user "alice,bob"

# Combine filters
x mentions --filter "bug,issue" --from-user "customer1,customer2"
```

### Monitor State

The monitor stores its state in `~/.config/x-cli/monitor.json`:

```bash
# Reset monitor state (clear last checked timestamp)
x mentions --reset
```

### Desktop Notifications

Desktop notifications are supported on:
- **macOS**: via `osascript` (built-in)
- **Linux**: via `notify-send` (libnotify)
- **macOS (alternative)**: via `terminal-notifier`

Notifications show:
- Author's screen name
- Truncated mention text
- Timestamp

### Examples

```bash
# Monitor for customer support mentions
x mentions --watch --filter "help,support,issue" --notify

# Check for urgent mentions in the last 30 minutes
x mentions --since 30m --filter "urgent"

# Monitor mentions from specific users
x mentions --watch --from-user "important_user" --interval 30s

# Get JSON output for processing
x mentions --since 1h -j | jq '.mentions[] | select(.is_new == true)'
```

---

## Analytics Dashboard

Track engagement metrics and analyze your X performance:

### User Analytics

```bash
# Get comprehensive analytics for a user (last 30 days)
x analytics --user navalcadet --days 30

# Export analytics to JSON
x analytics --user navalcadet --export analytics.json

# Export analytics to CSV
x analytics --user navalcadet --export analytics.csv
```

**User Analytics Report includes:**
- Total posts, likes, reposts, replies
- Average engagement per post
- Engagement rate (based on followers)
- Top performing posts
- Best posting hours
- Best posting days
- Follower growth tracking
- Top hashtags and mentions

### Tweet Analytics

```bash
# Track a specific tweet's performance over time
x analytics --tweet 1234567890
```

**Tweet Analytics Report includes:**
- Current metrics (likes, reposts, replies, views)
- Historical metrics tracking
- Growth rate per hour

### Trending Topics

```bash
# Find trending topics for a user
x analytics --user navalcadet --trending --days 7
```

**Trending Topics Report includes:**
- Most frequently used words
- Engagement per topic
- Topic frequency

### Best Posting Times

```bash
# Analyze best times to post
x analytics --user navalcadet --best-times
```

**Best Posting Times Report includes:**
- Top 5 hours with highest engagement
- Best days of the week
- Average engagement per time slot

### Data Storage

Analytics data is stored in: `~/.config/x-cli/analytics.db`

The database includes:
- Tweet records with timestamps
- User snapshots (follower counts over time)
- Engagement metrics history

### Export Formats

**JSON Export:**
```bash
x analytics --user navalcadet --export user-analytics.json
```

**CSV Export:**
```bash
x analytics --user navalcadet --export user-analytics.csv
```

CSV columns: tweet_id, text, created_at, likes, retweets, replies, quotes, views, bookmarks, recorded_at

---

## MCP Integration

Use X-CLI as a tool in Claude Desktop or OpenCode:

```bash
# Copy the MCP config
cp mcp-config.json ~/.config/claude/claude_desktop_config.json
```

**Available Tools:**
- `x_user_summary` - Generate LLM-readable user summary
- `x_search` - Search posts
- `x_user_posts` - Fetch user's posts
- `x_tweet` - Fetch post and replies
- `x_user_profile` - Get user profile
- `x_feed` - Fetch home timeline
- `x_followers` - Get user followers
- `x_following` - Get user following

---

## Commands Reference

### Authentication
- `x login` - Authenticate with OAuth 2.0
- `x logout` - Remove stored credentials
- `x profiles` - List all profiles
- `x doctor` - Check auth and system status

### Reading
- `x feed` - Fetch home timeline
- `x favorites` - Fetch bookmarked posts
- `x search` - Search posts (with advanced filters)
- `x tweet` - Show a post and replies
- `x list` - Fetch posts from a list
- `x user` - Show a user profile
- `x user-posts` - Fetch posts from a user
- `x user-summary` - Generate LLM-readable summary
- `x likes` - Fetch posts liked by a user
- `x followers` - Fetch followers
- `x following` - Fetch following
- `x mentions` - Monitor mentions (with filtering and notifications)

### Analytics
- `x analytics --user <name>` - User engagement analytics
- `x analytics --tweet <id>` - Tweet performance tracking
- `x analytics --trending` - Trending topics analysis
- `x analytics --best-times` - Best posting times

### Writing
- `x post` - Create a new post
- `x delete` - Delete a post
- `x like` - Like a post
- `x unlike` - Unlike a post
- `x retweet` - Repost a post
- `x unretweet` - Undo a repost
- `x bookmark` - Bookmark a post
- `x unbookmark` - Remove a bookmark

### Scheduling
- `x schedule save` - Schedule a post for later
- `x schedule list` - List scheduled posts
- `x schedule cancel` - Cancel a scheduled post
- `x schedule post` - Post a scheduled post now

### Templates
- `x template save` - Save a template
- `x template list` - List all templates
- `x template show` - Show template content
- `x template delete` - Delete a template
- `x template preview` - Preview with variables
- `x template export` - Export a template
- `x template import` - Import a template

### Diagnostics
- `x status` - Show project status
- `x version` - Show version info

---

## Configuration

Config file: `~/.config/x/config.yaml`

```yaml
output:
  format: table  # table, json, markdown

rate_limit:
  request_delay: 2500ms
  max_retries: 3
  retry_base_delay: 5s

auth:
  source: browser  # browser, env

http:
  proxy: ""  # HTTP or SOCKS proxy

url_shortener:
  enabled: false  # Enable URL shortening
  service: tinyurl  # tinyurl, isgd, vgd, custom
  custom_api_url: ""  # For custom service
  timeout: 10s  # API timeout
```

---

## Environment Variables

- `X_CONFIG` - Config file path
- `X_FORMAT` - Output format
- `X_PROFILE` - Default profile
- `X_VERBOSE` - Enable verbose logging
- `X_AUTH_TOKEN` - Auth token (alternative to OAuth)
- `X_CT0` - CT0 token (alternative to OAuth)
- `X_PROXY` - HTTP/SOCKS proxy

---

## Examples

### For LLMs/AI Agents

```bash
# Get user summary for analysis
x user-summary @navalcadet -j | llm "Summarize this user's interests"

# Search and analyze
x search "golang tips" -j | llm "Extract the top 5 tips"

# Monitor a topic
x search "AI news" --type Latest -m > ai_news.md
```

### For Power Users

```bash
# Post a thread from a blog post
cat blog.md | x post --thread

# Schedule monitoring (with cron)
*/30 * * * * x search "mycompany" -j > /var/log/x-mentions.json

# Multi-account workflow
x login --profile personal
x login --profile work
x post "Work update" --profile work
x post "Personal thought" --profile personal
```

---

## Security

- **OAuth 2.0 PKCE** - No client secrets in code
- **Keychain Storage** - OS-level secure storage
- **Encrypted Fallback** - AES-256-GCM for headless environments
- **Auto Token Refresh** - Transparent background refresh
- **State Parameter** - CSRF protection

---

## Development

```bash
# Clone the repo
git clone https://github.com/dl-alexandre/X-CLI.git
cd X-CLI

# Build
go build ./cmd/x

# Test
go test ./...

# Run
go run ./cmd/x --help
```

---

## License

MIT License - see [LICENSE](LICENSE)

---

## Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## Acknowledgments

Built with:
- [chromedp](https://github.com/chromedp/chromedp) - Browser automation
- [go-keyring](https://github.com/zalando/go-keyring) - Secure storage
- [kong](https://github.com/alecthomas/kong) - CLI framework
