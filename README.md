# X-CLI

[![Go Version](https://img.shields.io/github/go-mod/go-version/dl-alexandre/X-CLI)](https://github.com/dl-alexandre/X-CLI)
[![Release](https://img.shields.io/github/v/release/dl-alexandre/X-CLI)](https://github.com/dl-alexandre/X-CLI/releases/latest)
[![License](https://img.shields.io/github/license/dl-alexandre/X-CLI)](License)
[![Go Report Card](https://goreportcard.com/badge/github.com/dl-alexandre/X-CLI)](https://goreportcard.com/report/github.com/dl-alexandre/X-CLI)

`x` is a terminal-first CLI for X/Twitter built in Go. This repo starts from your Go template and is now retargeted toward timelines, search, tweet detail, lists, profiles, likes, followers, and bookmarks.

Features:
- Terminal-first CLI for X/Twitter
- Native Go implementation (10x faster than browser automation)
- 22 CLI commands (read + write + diagnostics)
- Homebrew installation support
- No API keys required

---

Install from source:

```bash
go install github.com/dl-alexandre/X-CLI/cmd/x@latest
```

Or via Homebrew:

```bash
brew tap dl-alexandre/homebrew-tap
brew install x
```
