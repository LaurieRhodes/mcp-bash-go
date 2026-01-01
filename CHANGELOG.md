# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-02

### Added

- Initial standalone release of bash MCP server
- Persistent bash session management
- Stdio transport for MCP communication
- Network transport with IP whitelisting
- Container escape mode for host access
- Configurable command timeout
- Automatic session restart capability
- Comprehensive error handling and logging
- Support for pipelines, environment variables, and redirects
- GitHub Actions CI/CD workflows
- Automated multi-platform binary builds

### Features

- Execute bash commands in stateful session
- Session state persists between calls
- Support for cd, export, aliases
- Background process handling
- 120-second default timeout
- Optional restart parameter
- Network mode with security controls
- Linux and macOS support (WSL for Windows users)

### Security

- Runs with process-level permissions
- No root access by default
- Configurable timeout prevents runaway commands
- IP whitelisting for network mode
- Container isolation by default
- Optional host access via nsenter

---

## Release Types

### Major (x.0.0)

- Breaking changes
- Major feature additions
- Architecture changes

### Minor (0.x.0)

- New features
- Non-breaking enhancements
- New capabilities

### Patch (0.0.x)

- Bug fixes
- Documentation updates
- Performance improvements

[Unreleased]: https://github.com/LaurieRhodes/mcp-bash-go/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/LaurieRhodes/mcp-bash-go/releases/tag/v1.0.0
