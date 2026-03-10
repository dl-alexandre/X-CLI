# X-CLI Project Information

## Project Overview

X-CLI is a terminal-first CLI for X built in Go. It provides a professional-grade, AI-friendly tool for power users and automation workflows.

## Architecture

### Package Structure

```
cmd/x/                    - Main application entry point
internal/
  auth/                    - Authentication (OAuth 2.0 PKCE, keychain storage)
  cache/                   - Caching layer
  cli/                     - CLI command definitions
  config/                  - Configuration management
  model/                   - Data models
  output/                  - Output formatting (table, JSON, markdown)
  text/                    - Text processing (thread splitting)
  xapi/                    - X API client
```

### Key Components

1. **Authentication** (`internal/auth/`)
   - OAuth 2.0 PKCE flow
   - Keychain storage (macOS/Windows/Linux)
   - Encrypted file fallback for headless environments
   - Automatic token refresh

2. **Rate Limiting** (`internal/xapi/ratelimit.go`)
   - 429 detection and automatic retry
   - Real-time countdown display
   - Configurable max retries and delays
   - Jitter support for anti-spam

3. **Thread Posting** (`internal/text/splitter.go`)
   - Smart word boundary detection
   - UTF-8 aware character counting
   - Automatic numbering (1/N, 2/N)
   - Reply chaining

4. **Media Upload** (`internal/xapi/media.go`)
   - Simple POST for images <5MB
   - Support: JPEG, PNG, GIF
   - Multipart form-data

5. **Output Formatting** (`internal/output/formatter.go`)
   - Table (default)
   - JSON (`-j` flag)
   - Markdown (`-m` flag)
   - User summary for LLMs

## Commands

### Authentication
- `x login [--name profile]` - Authenticate with OAuth 2.0
- `x logout [--name profile]` - Remove credentials
- `x profiles` - List all profiles
- `x doctor` - Check auth status

### Reading
- `x feed` - Home timeline
- `x search` - Search posts
- `x user` - User profile
- `x user-posts` - User's posts
- `x user-summary` - LLM-readable summary
- `x tweet` - Post and replies

### Writing
- `x post [--thread] [--file image.jpg]` - Create post
- `x delete` - Delete post
- `x like/unlike` - Like actions
- `x repost/unrepost` - Repost actions
- `x bookmark/unbookmark` - Bookmark actions

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
  proxy: ""
```

## Environment Variables

- `X_CONFIG` - Config file path
- `X_FORMAT` - Output format
- `X_PROFILE` - Default profile
- `X_VERBOSE` - Verbose logging
- `X_AUTH_TOKEN` - Auth token (alternative)
- `X_CT0` - CT0 token (alternative)

## Testing

Run all tests:
```bash
go test ./... -v
```

Run specific package:
```bash
go test ./internal/auth -v
go test ./internal/xapi -v
go test ./internal/text -v
```

## Build

```bash
go build ./cmd/x
```

## MCP Integration

X-CLI can be used as a tool in Claude Desktop or OpenCode via MCP.

Config: `mcp-config.json`

Available tools:
- `x_user_summary`
- `x_search`
- `x_user_posts`
- `x_tweet`
- `x_user_profile`
- `x_feed`
- `x_followers`
- `x_following`

## Security

- OAuth 2.0 PKCE (no client secrets)
- Keychain storage (OS-level)
- AES-256-GCM encryption (headless fallback)
- Automatic token refresh
- CSRF protection (state parameter)

## Dependencies

Key dependencies:
- `github.com/chromedp/chromedp` - Browser automation
- `github.com/zalando/go-keyring` - Secure storage
- `github.com/alecthomas/kong` - CLI framework
- `github.com/rodaine/table` - Table output

## Code Style

- No comments unless requested
- Follow existing patterns
- Use standard library when possible
- Keep functions small and focused
- Return errors, don't panic

## Common Tasks

### Add a new command

1. Define command struct in `internal/cli/cli.go`
2. Add to CLI struct
3. Implement Run method
4. Add to README.md

### Add a new API endpoint

1. Add method to `internal/xapi/client.go`
2. Use `doRequestWithRetry` for rate limit handling
3. Parse response into model
4. Add output formatting in `internal/output/formatter.go`

### Add a new output format

1. Add to `Print*` methods in `internal/output/formatter.go`
2. Add case in switch statement
3. Update README.md

## Known Limitations

- Media upload limited to 5MB (simple POST)
- No video upload support yet
- Thread splitting doesn't handle URLs specially
- OAuth tokens expire after 2 hours (auto-refreshed)

## Future Enhancements

- Chunked media upload for videos
- URL shortening in thread splitting
- Post scheduling
- Advanced search filters
- Export/import profiles
