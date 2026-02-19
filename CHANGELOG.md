# Changelog

All notable changes to the MCP Bash Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.1] - 2026-02-20

### Fixed

- **Orphaned bash processes** - Dead sessions are now explicitly closed before creating replacements. Previously, when a session died (timeout, pipe error), the next command silently created a new bash process without killing the old one, leaking a process on every retry.
- **Stderr goroutine leak and data race** - Replaced per-command stderr goroutine with a single persistent drainer per session. Each `execute()` call previously spawned a new goroutine reading from the same stderr pipe; these never terminated and raced on the shared reader.
- **Scanner buffer overflow on large output** - Increased `bufio.Scanner` buffer from the default 64KB to 1MB. Commands producing long output lines (e.g., raw JSON from API responses) triggered `token too long` errors that killed the session.
- **Session close idempotency** - `close()` now always proceeds to kill the process and clean up pipes, even when `running` is already false. Previously it returned early, leaving dead processes unkillable.
- **Unbounded output memory growth** - Command output is now capped at 512KB with a clear truncation message.
- **Verbose debug logging** - Response and request payloads in stderr logs are now truncated to prevent flooding when commands return large output.

### Changed

- **Default config no longer includes network settings** - The auto-generated `config.json` now contains only `commandTimeout` and `enabled`. Network mode configuration is intentionally omitted to avoid exposing an unauthenticated TCP listener by default. See `config.network.json` for an example of how to enable it.
- `Config.Network` is now a pointer field (`*NetworkConfig`) with `omitempty`, so it is absent from serialised JSON when not configured.

### Added

- **Claude skill** (`claude-skill/bash-preference.zip`) - Instructs claude.ai to route all bash execution through the MCP server instead of its built-in sandbox. See README for deployment steps.

## [1.1.0] - 2026-01-23

### Added

- **Automatic environment injection** for nested MCP support
  - `MCP_NESTED=1` signals nested MCP context
  - `MCP_SOCKET_DIR` and `MCP_SKILLS_SOCKET` for Unix socket communication
- **Unix socket support** for nested MCP communication avoiding stdio conflicts

### Fixed

- **Critical:** Resolved stdio deadlock when bash tool executes workflows that call other MCP tools

## [1.0.0] - 2026-01-02

### Added

- Persistent bash sessions with state across executions
- Configurable command timeouts (default: 600 seconds)
- Full MCP protocol implementation with stdio transport
- Optional network mode with IP and subnet filtering
- JSON-based configuration with auto-generation of defaults
- Session restart capability
