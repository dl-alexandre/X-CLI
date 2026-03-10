# Distribution Guide

This document explains how to install X-CLI on different platforms.

## Homebrew (macOS/Linux)

### From Tap Repository
```bash
brew tap dl-alexandre/homebrew-tap
brew install x-cli
```

### From Local Formula
```bash
brew install --formula homebrew/x-cli.rb
```

### Upgrade
```bash
brew upgrade x-cli
```

### Uninstall
```bash
brew uninstall x-cli
brew untap dl-alexandre/homebrew-tap
```

---

## Snap (Linux)

### Install from Snap Store
```bash
sudo snap install x-cli
```

### Install from Local Build
```bash
sudo snap install --dangerous x-cli_1.0.0_amd64.snap
```

### Upgrade
```bash
sudo snap refresh x-cli
```

### Uninstall
```bash
sudo snap remove x-cli
```

---

## Docker

### Pull from Registry
```bash
docker pull ghcr.io/dl-alexandre/x-cli:latest
```

### Run
```bash
docker run --rm -v ~/.config/x-cli:/root/.config/x-cli x-cli x --help
```

### Build Locally
```bash
make docker-build
```

---

## Binary Installation

### Download
Download the latest release for your platform:
- Linux AMD64: `x-linux-amd64`
- Linux ARM64: `x-linux-arm64`
- macOS AMD64: `x-darwin-amd64`
- macOS ARM64: `x-darwin-arm64`
- Windows AMD64: `x-windows-amd64.exe`

### Install
```bash
# Linux/macOS
chmod +x x-*
sudo mv x-* /usr/local/bin/x

# Windows
# Add to PATH manually
```

---

## Build from Source

### Prerequisites
- Go 1.21 or later
- Git

### Clone and Build
```bash
git clone https://github.com/dl-alexandre/X-CLI.git
cd X-CLI
make deps
make build
sudo make install
```

### Run Tests
```bash
make test
```

---

## Shell Completions

### Bash
```bash
x completion bash > /etc/bash_completion.d/x
source ~/.bashrc
```

### Zsh
```bash
x completion zsh > "${fpath[1]}/_x"
autoload -U compinit && compinit
```

### Fish
```bash
x completion fish > ~/.config/fish/completions/x.fish
```

---

## Configuration

Config file location: `~/.config/x-cli/config.yaml`

First run will create default config automatically.

---

## Troubleshooting

### Permission Denied
```bash
sudo chmod +x /usr/local/bin/x
```

### Config Not Found
```bash
mkdir -p ~/.config/x-cli
```

### Keychain Access (macOS)
Allow X-CLI access to Keychain when prompted.

### Snap Connection Issues
```bash
sudo snap connect x-cli:network
sudo snap connect x-cli:home
```
