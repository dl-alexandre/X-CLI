# Distribution Setup Complete

## Overview

X-CLI is now fully configured for distribution across multiple platforms.

---

## Distribution Channels

### 1. Homebrew (macOS/Linux) ✅

**Formula Location:** `homebrew/x-cli.rb`

**Installation:**
```bash
brew tap dl-alexandre/homebrew-tap
brew install x-cli
```

**Automated Updates:**
- GoReleaser automatically updates the tap on release
- Requires `HOMEBREW_TAP_GITHUB_TOKEN` secret in GitHub

**Setup Required:**
1. Create GitHub Personal Access Token with repo permissions
2. Add as secret: `HOMEBREW_TAP_GITHUB_TOKEN`
3. Tag a release: `git tag v1.0.0 && git push --tags`

---

### 2. Snap (Linux) ✅

**Snapcraft File:** `snap/snapcraft.yaml`

**Installation:**
```bash
sudo snap install x-cli
```

**Automated Builds:**
- GoReleaser builds Snap package on release
- Requires `SNAPCRAFT_TOKEN` secret in GitHub

**Setup Required:**
1. Register at https://snapcraft.io
2. Create Snap Store credentials
3. Add as secret: `SNAPCRAFT_TOKEN`
4. Tag a release

---

### 3. Docker ✅

**Dockerfile:** `Dockerfile`

**Pull:**
```bash
docker pull ghcr.io/dl-alexandre/x-cli:latest
```

**Run:**
```bash
docker run --rm \
  -v ~/.config/x-cli:/root/.config/x-cli \
  ghcr.io/dl-alexandre/x-cli:latest \
  x --help
```

**Automated Builds:**
- GoReleaser builds and pushes to GitHub Container Registry
- No additional secrets needed (uses GITHUB_TOKEN)

---

### 4. Scoop (Windows) ✅

**Configuration:** `.goreleaser.yml`

**Installation:**
```powershell
scoop bucket add dl-alexandre https://github.com/dl-alexandre/scoop-bucket
scoop install x-cli
```

**Setup Required:**
1. Create scoop-bucket repository
2. Add `SCOOP_BUCKET_GITHUB_TOKEN` secret
3. Tag a release

---

### 5. Binary Releases ✅

**Platforms:**
- Linux AMD64/ARM64
- macOS AMD64/ARM64 (Intel/Apple Silicon)
- Windows AMD64

**Download:**
https://github.com/dl-alexandre/X-CLI/releases/latest

**Automated by GoReleaser:**
- Builds all platforms
- Creates archives (tar.gz/zip)
- Generates checksums
- Creates GitHub release

---

## Release Process

### Automated Release

```bash
# Tag and push
git tag v1.0.0
git push origin v1.0.0

# GoReleaser automatically:
# 1. Runs tests
# 2. Builds binaries for all platforms
# 3. Creates GitHub release
# 4. Updates Homebrew tap
# 5. Publishes Snap package
# 6. Pushes Docker image
# 7. Updates Scoop bucket
```

### Manual Release

```bash
# Build all platforms
make build-all

# Create distribution packages
make dist-packages

# Build Snap
make snap

# Build Docker
make docker-build
```

---

## Required GitHub Secrets

| Secret | Purpose | Required For |
|--------|---------|--------------|
| `GITHUB_TOKEN` | GitHub API access | All (auto-provided by GitHub Actions) |
| `SNAPCRAFT_TOKEN` | Snap Store authentication | Snap only |

**Note:** Homebrew and Scoop use the auto-provided `GITHUB_TOKEN`, so no additional secrets needed for those!

---

## File Structure

```
X-CLI/
├── .github/
│   └── workflows/
│       └── release.yml          # GoReleaser workflow
├── .goreleaser.yml              # GoReleaser config
├── snap/
│   └── snapcraft.yaml           # Snap package config
├── homebrew/
│   └── x-cli.rb                 # Homebrew formula
├── Dockerfile                   # Docker image
├── Makefile                     # Build automation
└── DISTRIBUTION.md              # Distribution guide
```

---

## Testing Distribution

### Test Homebrew Locally
```bash
brew install --formula homebrew/x-cli.rb
x --help
```

### Test Snap Locally
```bash
snapcraft
sudo snap install --dangerous x-cli_*.snap
x --help
```

### Test Docker Locally
```bash
docker build -t x-cli:test .
docker run --rm x-cli:test x --help
```

---

## Next Steps

1. **Create GitHub Secret:**
   - `SNAPCRAFT_TOKEN` (only if using Snap)

2. **Create Repository:**
   - `dl-alexandre/scoop-bucket` (for Windows)

3. **Tag First Release:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. **Verify:**
   - Check GitHub Actions workflow
   - Verify release appears on GitHub
   - Test `brew install x-cli`
   - Test `snap install x-cli` (if configured)
   - Test `docker pull ghcr.io/dl-alexandre/x-cli`

**Note:** Homebrew, Scoop, and Docker all work with the auto-provided `GITHUB_TOKEN` - no additional setup needed!

---

## Support

- **Issues:** https://github.com/dl-alexandre/X-CLI/issues
- **Docs:** README.md, DISTRIBUTION.md
- **Releases:** https://github.com/dl-alexandre/X-CLI/releases
