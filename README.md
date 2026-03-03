# {{APPNAME}}

{{DESCRIPTION}}

## Features

- **Modern CLI Framework**: Built with [Kong](https://github.com/alecthomas/kong) for declarative, struct-based command definition
- **Multiple Output Formats**: Support for table, JSON, and markdown output
- **Configuration Management**: Flexible configuration via files, environment variables, and flags
- **Caching Layer**: Built-in file-based caching with TTL support
- **Cross-Platform**: Builds for Linux, macOS, and Windows (AMD64 and ARM64)
- **Shell Completions**: Bash, Zsh, Fish, and PowerShell completion support
- **Release Automation**: GoReleaser configuration for automated releases

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap {{OWNER}}/{{APPNAME}}
brew install {{APPNAME}}
```

### Go Install

```bash
go install github.com/{{OWNER}}/{{APPNAME}}/cmd/{{APPNAME}}@latest
```

### Binary Download

Download the appropriate binary for your platform from the [releases page](https://github.com/{{OWNER}}/{{REPO_NAME}}/releases).

## Quick Start

### Creating from Template

1. Click the green **"Use this template"** button on GitHub, or clone and run:
   ```bash
   git clone https://github.com/{{OWNER}}/{{APPNAME}}.git my-cli
   cd my-cli
   ./scripts/setup.sh
   ```

2. Follow the setup prompts, then see **[SETUP_GUIDE.md](SETUP_GUIDE.md)** for:
   - GitHub repository configuration
   - Enabling automated releases
   - First release checklist
   - pkg.go.dev registration

### Example Commands

```bash
# Show help
{{APPNAME}} --help

# List resources
{{APPNAME}} list

# Get a specific resource
{{APPNAME}} get <id>

# Search resources
{{APPNAME}} search "query"

# Show version
{{APPNAME}} version
```

## Configuration

Configuration can be provided via:

1. **Config file**: `~/.config/{{APPNAME}}/config.yaml`
2. **Environment variables**: `{{APPNAME}}_API_URL`, `{{APPNAME}}_TIMEOUT`, etc.
3. **Command-line flags**: `--api-url`, `--timeout`, etc.

### Example Config File

```yaml
api:
  url: "https://api.example.com"
  timeout: 30
  key: "your-api-key"

cache:
  enabled: true
  dir: "~/.config/{{APPNAME}}/cache"
  ttl: 60m
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `{{APPNAME}}_API_URL` | API base URL | `https://api.example.com` |
| `{{APPNAME}}_TIMEOUT` | Request timeout (seconds) | `30` |
| `{{APPNAME}}_NO_CACHE` | Disable caching | `false` |
| `{{APPNAME}}_VERBOSE` | Enable verbose output | `false` |
| `{{APPNAME}}_FORMAT` | Default output format | `table` |

## Development

### Prerequisites

- Go 1.24 or later
- golangci-lint (for linting)
- GoReleaser (for releases)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Development build
make dev
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Run all checks
make check
```

### Available Make Targets

```
make build          Build for current platform
make build-all      Build for all platforms
make test           Run tests
make lint           Run linter
make format         Format code
make release        Build optimized release binary
make clean          Clean build artifacts
make install        Install locally
make install-hooks  Install git hooks
make help           Show help
```

## Shell Completions

### Bash

```bash
{{APPNAME}} completion bash > /usr/local/etc/bash_completion.d/{{APPNAME}}
```

### Zsh

```bash
{{APPNAME}} completion zsh > "${fpath[1]}/_{{APPNAME}}"
```

### Fish

```bash
{{APPNAME}} completion fish > ~/.config/fish/completions/{{APPNAME}}.fish
```

### PowerShell

```powershell
{{APPNAME}} completion powershell > {{APPNAME}}.ps1
# Source the file in your PowerShell profile
```

## Project Structure

```
.
├── cmd/{{APPNAME}}/         # Entry point
│   └── main.go
├── internal/
│   ├── cli/                 # CLI command definitions
│   ├── api/                 # HTTP client
│   ├── config/              # Configuration management
│   ├── output/              # Output formatters
│   └── cache/               # Caching layer
├── .github/
│   └── workflows/           # CI/CD workflows
├── Makefile                 # Build automation
├── .goreleaser.yml         # Release configuration
├── .golangci.yml           # Linter configuration
├── go.mod                  # Go module definition
├── README.md               # This file
└── LICENSE                 # MIT License
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and linting (`make check`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and development process.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Kong](https://github.com/alecthomas/kong) - Command-line parser
- [resty](https://github.com/go-resty/resty) - HTTP client
- [Viper](https://github.com/spf13/viper) - Configuration management
- [rodaine/table](https://github.com/rodaine/table) - Table formatting
