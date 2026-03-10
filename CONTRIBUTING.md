# Contributing to x

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to this project.

## Development Setup

### Prerequisites

- Go 1.24 or later
- Make
- golangci-lint
- GoReleaser (optional, for testing releases)

### Setting Up Your Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/X-CLI.git
   cd X-CLI
   ```

3. Install dependencies:
   ```bash
   make deps
   ```

4. Install git hooks:
   ```bash
   make install-hooks
   ```

## Development Workflow

### Making Changes

1. Create a new branch for your feature or fix:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

2. Make your changes and ensure tests pass:
   ```bash
   make check  # Runs format, vet, lint, and test
   ```

3. Commit your changes with a clear commit message following [Conventional Commits](https://www.conventionalcommits.org/):
   ```bash
   git commit -m "feat: add new command for X"
   # or
   git commit -m "fix: resolve issue with Y"
   ```

### Commit Message Format

We follow the Conventional Commits specification:

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, missing semi colons, etc)
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `test:` - Adding or correcting tests
- `chore:` - Changes to build process, dependencies, etc

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run integration tests (requires API access)
make test-integration

# Run linter
make lint

# Format code
make format
```

### Pre-commit Checks

The pre-commit hook will automatically run:
- `go fmt` - Format check
- `go vet` - Static analysis
- `go test -short` - Quick tests

If any check fails, the commit will be blocked. Fix the issues and try again.

## Project Structure

```
.
├── cmd/x/         # Entry point
├── internal/
│   ├── cli/                 # CLI command definitions (Kong structs)
│   ├── api/                 # HTTP client and API types
│   ├── config/              # Configuration management (Viper)
│   ├── output/              # Output formatters (table, json, markdown)
│   └── cache/               # Caching layer
├── .github/
│   └── workflows/           # CI/CD workflows
├── scripts/                 # Build and setup scripts
└── Makefile                 # Build automation
```

## Code Guidelines

### Go Code Style

- Follow standard Go conventions
- Use meaningful variable names
- Add comments for exported functions and types
- Keep functions focused and small
- Handle errors explicitly

### Adding New Commands

1. Define the command struct in `internal/cli/cli.go`:
   ```go
   type NewCmd struct {
       Arg1 string `arg:"" help:"Description"`
       Flag bool   `help:"Description"`
   }
   ```

2. Implement the `Run` method:
   ```go
   func (c *NewCmd) Run(globals *Globals) error {
       // Implementation
       return nil
   }
   ```

3. Register in the CLI struct:
   ```go
   type CLI struct {
       // ...
       New NewCmd `cmd:"" help:"Description"`
   }
   ```

4. Add tests in `internal/cli/cli_test.go`

### API Client Guidelines

- Use the `resty` client for HTTP requests
- Define clear request/response types
- Handle errors with appropriate error types
- Support context for cancellation
- Add retries for transient failures

### Output Format Guidelines

- Support table, JSON, and markdown formats
- Use the `output.Printer` for consistent formatting
- Respect the `--format` flag
- Handle empty results gracefully

## Testing

### Unit Tests

- Place tests in `*_test.go` files
- Use table-driven tests where appropriate
- Mock external dependencies
- Test error cases

### Integration Tests

- Use the `integration` build tag: `//go:build integration`
- Test against a real API when possible
- Clean up resources after tests

### Example Test

```go
func TestListCmd(t *testing.T) {
    cmd := &ListCmd{
        Limit: 10,
    }
    
    globals := &Globals{
        Format: "json",
    }
    
    // Mock client and test
    err := cmd.Run(globals)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```

## Documentation

- Update README.md for user-facing changes
- Add examples to help text
- Document flags and their behavior
- Keep CHANGELOG.md updated

## Release Process

Releases are automated via GoReleaser when a new tag is pushed:

```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for general questions
- Check existing issues before creating new ones

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code:

- Be respectful and inclusive
- Accept constructive criticism gracefully
- Focus on what's best for the community
- Show empathy towards others

Thank you for contributing!
