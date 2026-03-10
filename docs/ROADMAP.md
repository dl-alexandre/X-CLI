# X-CLI Roadmap: Post-Native Transport

## 🚀 Leveraging 10x Performance Gains

Now that the native transport layer is complete, we can build features that were previously impractical with browser automation (3-5s per operation).

## Phase 1: Batch Operations (High Priority)

### Bulk Tweet Management
```bash
# Delete multiple tweets efficiently
x bulk-delete --from-file tweets-to-delete.txt --confirm

# Like/unlike entire conversation threads
x thread-like <tweet-id>       # Like OP + all replies
x thread-unlike <tweet-id>     # Unlike entire thread

# Mass bookmark operations
x bulk-bookmark --search "golang tips" --max 50
```

### List Management
```bash
# Create curated lists from search results
x list-create-from-search "rust developers" --min-followers 1000

# Bulk add/remove list members
x list-bulk-add <list-id> --from-file users.txt
```

## Phase 2: Real-Time Monitoring (Medium Priority)

### Timeline Streaming
```bash
# Monitor home timeline in real-time
x stream --type mentions --exec "notify-send 'New mention'"

# Watch for specific keywords
x stream --search "#golang" --interval 30s --alert

# Follower growth tracking
x stream-followers --user <handle> --report hourly
```

### Event-Driven Actions
```bash
# Auto-like replies to your tweets
x auto-like-replies --window 5m

# Auto-retweet posts matching criteria
x auto-retweet --search "#buildinpublic" --filter "min_likes:10"
```

## Phase 3: Media & Content (Medium Priority)

### Native Media Upload
```bash
# Upload images without browser
x post-with-media "Check this out" --images img1.jpg,img2.png

# Video uploads with progress bar
x post-with-video "Demo" --video demo.mp4 --trim 0:30

# Thread with media
x thread --file thread.md --media-dir ./images/
```

### Content Export
```bash
# Export your timeline
x export --timeline --since 2024-01-01 --format json

# Backup bookmarks
x export-bookmarks --all --include-media

# Archive entire conversations
x export-thread <tweet-id> --recursive --format markdown
```

## Phase 4: Analytics & Intelligence (Lower Priority)

### Performance Metrics
```bash
# Tweet analytics
x stats <tweet-id> --engagement --impressions

# Profile analytics
x profile-stats --user <handle> --period 30d

# Compare engagement rates
x compare-stats <tweet-id-1> <tweet-id-2>
```

### Follower Analysis
```bash
# Find mutual followers
x mutuals --user <handle> --export

# Identify inactive followers
x follower-cleanup --inactive-since 90d --dry-run

# Growth analysis
x growth-report --period 30d --chart
```

## Phase 5: Developer Experience (Ongoing)

### Shell Integration
```bash
# Auto-completion for all shells
x completion bash > /etc/bash_completion.d/x
x completion zsh > ~/.zfunc/_x
x completion fish > ~/.config/fish/completions/x.fish

# FZF integration for interactive selection
x fzf-search "golang" | x like
```

### Plugin System
```bash
# Load custom processors
x plugin load ./my-filter.so

# Webhook support
x webhook --port 8080 --exec "./notify.sh"
```

## Technical Architecture for New Features

### Batch Operation Engine
```go
// internal/batch/processor.go
type BatchProcessor struct {
    concurrency int           // Parallel workers
    rateLimit   time.Duration // Respect API limits
    retryPolicy RetryConfig    // Exponential backoff
}
```

### Streaming Infrastructure
```go
// internal/stream/poller.go
type TimelinePoller struct {
    interval    time.Duration
    lastCheck   time.Time
    checkFunc   func() ([]Tweet, error)
}
```

### Media Pipeline
```go
// internal/media/upload.go
type MediaUploader struct {
    chunkSize   int64
    compression bool
    formatCheck func([]byte) error
}
```

## Performance Targets

| Feature | Browser Time | Native Target | Speedup |
|---------|-------------|---------------|---------|
| Bulk delete 100 tweets | 5-8 minutes | 30-60 seconds | **10x** |
| Stream mentions | N/A | Real-time | **∞x** |
| Export 1000 tweets | 15-20 minutes | 2-3 minutes | **8x** |
| Media upload | 30-60s | 5-10s | **6x** |

## Implementation Priority

1. **Bulk Operations** - Immediate value, uses existing APIs
2. **Streaming** - Real-time features differentiate the CLI
3. **Media Upload** - Complete the write capability set
4. **Export/Backup** - Data ownership features
5. **Analytics** - Power-user features

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:
- Adding new commands
- Extending the native transport
- Writing tests for batch operations
- Documentation standards

---

**Next immediate task**: Implement `x bulk-delete` with parallel processing and progress bars.
