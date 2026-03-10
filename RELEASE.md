# Quick Release Guide

## Simplified Setup

X-CLI uses the auto-provided `GITHUB_TOKEN` for most distribution channels, minimizing setup requirements.

---

## What You Need

### Required
- ✅ GitHub repository (already have)
- ✅ `dl-alexandre/homebrew-tap` repository (already have)
- ✅ `dl-alexandre/scoop-bucket` repository (create once)

### Optional
- `SNAPCRAFT_TOKEN` secret (only for Snap Store)

---

## One-Time Setup

### 1. Create Scoop Bucket (Windows)
```bash
# Create new repository on GitHub
# Name: dl-alexandre/scoop-bucket
# Description: Scoop bucket for X-CLI
```

### 2. Add Snap Token (Optional)
If you want Snap distribution:
1. Go to https://snapcraft.io/x-cli
2. Register the snap
3. Generate credentials: `snapcraft export-login --snaps x-cli -`
4. Add as GitHub secret: `SNAPCRAFT_TOKEN`

---

## Release Process

### Tag and Release
```bash
git tag v1.0.0
git push origin v1.0.0
```

### What Happens Automatically
1. ✅ Tests run
2. ✅ Binaries built for all platforms
3. ✅ GitHub release created
4. ✅ Homebrew tap updated (uses GITHUB_TOKEN)
5. ✅ Scoop bucket updated (uses GITHUB_TOKEN)
6. ✅ Docker image pushed (uses GITHUB_TOKEN)
7. ✅ Snap published (if SNAPCRAFT_TOKEN set)

---

## Distribution Channels

| Channel | Status | Token Required |
|---------|--------|----------------|
| GitHub Releases | ✅ Ready | None (auto) |
| Homebrew | ✅ Ready | None (auto) |
| Docker | ✅ Ready | None (auto) |
| Scoop | ✅ Ready | None (auto) |
| Snap | ⚠️ Optional | SNAPCRAFT_TOKEN |

---

## Install Commands

### Homebrew
```bash
brew tap dl-alexandre/homebrew-tap
brew install x-cli
```

### Docker
```bash
docker pull ghcr.io/dl-alexandre/x-cli:latest
```

### Scoop (Windows)
```powershell
scoop bucket add dl-alexandre https://github.com/dl-alexandre/scoop-bucket
scoop install x-cli
```

### Snap (Linux)
```bash
sudo snap install x-cli
```

---

## Troubleshooting

### Homebrew not updating?
- Check GitHub Actions workflow completed
- Verify homebrew-tap repo has new commit
- Run: `brew update && brew upgrade x-cli`

### Scoop not updating?
- Check scoop-bucket repo has new commit
- Run: `scoop update && scoop update x-cli`

### Docker image not found?
- Wait 5-10 minutes after release
- Check GitHub Packages: https://github.com/dl-alexandre/X-CLI/pkgs

### Snap not published?
- Verify SNAPCRAFT_TOKEN secret is set
- Check Snap Store: https://snapcraft.io/x-cli

---

## Manual Release (if needed)

```bash
# Build all platforms
make build-all

# Create packages
make dist-packages

# Build Docker
make docker-build

# Build Snap
make snap
```

---

## Files

- `.github/workflows/release.yml` - Release automation
- `.goreleaser.yml` - Distribution configuration
- `homebrew/x-cli.rb` - Homebrew formula
- `snap/snapcraft.yaml` - Snap configuration
- `Dockerfile` - Docker image

---

## Support

- Issues: https://github.com/dl-alexandre/X-CLI/issues
- Releases: https://github.com/dl-alexandre/X-CLI/releases
- Documentation: README.md, DISTRIBUTION.md
