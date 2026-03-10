# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure with Kong CLI framework
- Example commands: list, get, search
- Multiple output formats (table, json, markdown)
- File-based caching with TTL
- Configuration via files, environment variables, and flags
- Shell completion support (bash, zsh, fish, powershell)
- Makefile with standard targets
- GitHub Actions CI/CD workflows
- GoReleaser configuration for releases
- golangci-lint configuration

## [1.0.0] - YYYY-MM-DD

### Added
- Initial release
- Basic API client with resty
- Configuration management with Viper
- Output formatting with rodaine/table
- Git hooks for pre-commit checks
- Comprehensive documentation

[Unreleased]: https://github.com/dl-alexandre/X-CLI/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/dl-alexandre/X-CLI/releases/tag/v1.0.0
