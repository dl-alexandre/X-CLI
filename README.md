# X-CLI

`x` is a terminal-first CLI for X/Twitter built in Go. This repo starts from your Go template and is now retargeted toward timelines, search, tweet detail, lists, profiles, likes, followers, and bookmarks.

Install from source:

```bash
go install github.com/dl-alexandre/X-CLI/cmd/x@latest
```

## Current State

- Project name: `X-CLI`
- Binary name: `x`
- Module path: `github.com/dl-alexandre/X-CLI`
- Status: `feed`, `favorites`, `list`, `user`, `user-posts`, `likes`, `followers`, `following`, `tweet`, `search`, `post`, `delete`, `like`, `unlike`, `bookmark`, `unbookmark`, `retweet`, and `unretweet` are working against live X data with a Go-only implementation; `post` and `delete` are verified through the remote-debug browser path

## Command Surface

```bash
x status
x doctor
x analyze-txid /tmp/x-txid-trace.jsonl
x feed --type for-you --max 20
x favorites --max 20
x search "golang" --type Latest --max 20
x tweet 1234567890
x list 1234567890
x user jack
x user-posts jack --max 20
x likes jack --max 20
x followers jack --max 20
x following jack --max 20
x post "hello from x-cli"
x delete 1234567890
x like 1234567890
x unlike 1234567890
x retweet 1234567890
x unretweet 1234567890
x bookmark 1234567890
x unbookmark 1234567890
x version
```

Right now `feed`, `favorites`, `list`, `user`, `user-posts`, `likes`, `followers`, `following`, `tweet`, and `search` are live. `status` is also functional. Browser cookie extraction and authenticated browser-backed commands now run in Go as well, so the current implementation no longer depends on Python subprocess bridges. All current write actions are working; `post` and `delete` are verified through a real Chrome remote-debug session.

`x doctor` checks auth loading, remote-debug availability, the native uTLS transport, and the current native transaction-id provider status.

`x analyze-txid <file>` summarizes a captured txid JSONL corpus so you can study encoded length, decoded byte length, same-operation bit deltas, and bit-level probabilities while reverse-engineering the native signer. Use `--operation CreateTweet` to isolate one mutation class and `--full-bits` to print the full 560-bit probability map.

For focused mutation research, set `X_BROWSER_TRACE_TXID_FILE` and optionally `X_BROWSER_TRACE_TXID_OPS=CreateTweet,DeleteTweet,FavoriteTweet,UnfavoriteTweet` to build a filtered write-only corpus.

## Configuration

Configuration is read from `~/.config/x/config.yaml` by default.

Environment variables:

- `X_FORMAT`
- `X_CONFIG`
- `X_VERBOSE`
- `X_DEBUG`
- `X_BROWSER_REMOTE_DEBUG_URL`
- `X_BROWSER_TRACE_TXID_FILE`
- `X_BROWSER_TRACE_TXID_OPS`
- `TWITTER_AUTH_TOKEN`
- `TWITTER_CT0`
- `TWITTER_PROXY`

Example config:

```yaml
output:
  format: table

fetch:
  count: 50

filter:
  mode: topN
  top_n: 20
  min_score: 50
  exclude_retweets: false
  weights:
    likes: 1.0
    retweets: 3.0
    replies: 2.0
    bookmarks: 5.0
    views_log: 0.5

rate_limit:
  request_delay: 2500ms
  max_retries: 3
  retry_base_delay: 5s
  max_count: 200

auth:
  source: browser

http:
  graphql_base_url: https://x.com/i/api/graphql

browser:
  remote_debug_url: ""
  trace_txid_mode: writes
  trace_txid_ops: ""
```

For harder write flows, you can point X-CLI at a real Chrome debug session instead of the temporary cloned profile:

```bash
rm -rf /tmp/x-cli-remote-debug
open -na "Google Chrome" --args --user-data-dir=/tmp/x-cli-remote-debug --remote-debugging-port=9222
export X_BROWSER_REMOTE_DEBUG_URL=http://127.0.0.1:9222/json/version
```

On macOS, Chrome may not expose the DevTools socket when launched against the default profile. Using a dedicated `--user-data-dir` works reliably; sign in to X in that debug window once, then X-CLI can attach to it. When that endpoint is available, X-CLI will attach to the remote-debug browser first and only fall back to the cloned profile when no debugger is running.

## Project Structure

```text
cmd/x/                  entrypoint for the x binary
internal/cli/           Kong command definitions
internal/config/        config loading and defaults
internal/model/         tweet and profile models
internal/output/        table/json/markdown rendering
internal/xapi/          X/Twitter client scaffold
```

## Development

```bash
make build
make test
make format
./bin/x status
./bin/x --help
```

## Next Implementation Steps

1. Replace the browser-backed authenticated paths with native authenticated API transport.
2. Tighten auth caching and error handling.
3. Add stronger integration tests around the browser-backed paths.
4. Smooth out the remote-debug setup UX.
5. Reduce the remaining browser automation dependence over time.
