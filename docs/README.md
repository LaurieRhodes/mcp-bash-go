# MCP Bash Server Documentation

## Overview

The MCP Bash Server provides command execution capabilities through the Model Context Protocol (MCP). It allows AI assistants like Claude to execute bash commands in a persistent session with proper timeout handling and environment management.

## Key Features

- **Persistent bash sessions** - Commands maintain state across executions
- **Automatic timeout handling** - Configurable timeouts prevent hanging
- **Nested MCP support** - Automatic environment injection for nested tool execution
- **Network mode support** - Optional TCP/IP connectivity with IP filtering
- **Progress notifications** - Real-time command execution feedback

## Quick Links

- [Architecture Overview](architecture.md) - System design and components
- [Quick Start Guide](quickstart.md) - Get running in 5 minutes
- [Nested MCP Solution](nested-mcp.md) - How we solved the deadlock problem
- [Configuration Guide](configuration.md) - Timeout and network settings
- [Troubleshooting](troubleshooting.md) - Common issues and solutions

## Installation

**Prerequisites:**
- Linux or macOS (Windows users: install in WSL)
- Claude Desktop (or compatible MCP client)

### Option 1: Pre-Built Binaries (Recommended)

Download the latest release for your platform:

- **Linux x86_64:** [mcp-bash-linux-amd64](https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-linux-amd64)
- **Linux ARM64:** [mcp-bash-linux-arm64](https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-linux-arm64)
- **macOS Intel:** [mcp-bash-darwin-amd64](https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-darwin-amd64)
- **macOS Apple Silicon:** [mcp-bash-darwin-arm64](https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-darwin-arm64)

All binaries include SHA256 checksums for verification.

See [Quick Start Guide](quickstart.md) for detailed installation instructions.

### Option 2: Build from Source

**Prerequisites:** Go 1.21 or later

```bash
git clone https://github.com/LaurieRhodes/mcp-bash-go.git
cd mcp-bash-go
go build -o mcp-bash ./cmd/server
```

See [Development Guide](architecture.md#development) for building and testing.

## Basic Usage

### Configure Claude Desktop

Add to `~/.config/Claude/claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "bash": {
      "command": "/usr/local/bin/mcp-bash/mcp-bash"
    }
  }
}
```

### Execute Commands

Through Claude's bash tool:
```bash
# Simple command
ls -la

# Persistent session
cd /tmp
pwd  # Shows /tmp

# Restart session
# Set restart: true in tool arguments
```

## Configuration

### Timeout Settings

Edit `/usr/local/bin/mcp-bash/config.json`:
```json
{
  "commandTimeout": 600
}
```

Default: 600 seconds (10 minutes)

### Network Mode

**Warning:** Network mode exposes the server on TCP/IP. Use IP filtering!

Edit `config.json`:
```json
{
  "network": {
    "enabled": true,
    "host": "127.0.0.1",
    "port": 8080,
    "allowedIPs": ["127.0.0.1"],
    "allowedSubnets": ["192.168.1.0/24"]
  }
}
```

## Nested MCP Execution

The server automatically handles nested MCP scenarios where Claude uses the bash tool to execute other MCP tools.

**Automatic environment injection:**
- `MCP_NESTED=1` - Signals nested execution context
- `MCP_SOCKET_DIR=/tmp/mcp-sockets` - Socket directory location
- `MCP_SKILLS_SOCKET=/tmp/mcp-sockets/skills.sock` - Skills server socket

See [Nested MCP Documentation](nested-mcp.md) for details.

## Project Structure

```
mcp-bash-go/
├── .github/workflows/   # CI/CD automation
│   ├── ci.yml          # Testing, linting, coverage
│   └── release.yml     # Multi-platform builds
├── cmd/server/         # Main entry point
├── pkg/
│   ├── bash/          # Bash execution + env injection
│   ├── config/        # Configuration management
│   ├── env/           # Environment variables
│   └── mcp/           # MCP protocol
├── docs/              # Documentation
└── config.json        # Default configuration
```

## CI/CD

The project uses GitHub Actions for:

- **Testing** - Go 1.21 & 1.22, race detection, coverage
- **Linting** - golangci-lint on every PR
- **Releases** - Automated multi-platform builds on version tags

See [.github/workflows/](../.github/workflows/) for workflow definitions.

## Development

**Run tests:**
```bash
go test ./...
```

**Build:**
```bash
go build -o mcp-bash ./cmd/server
```

**Debug mode:**
```bash
# Logs to stderr
./mcp-bash 2>debug.log
```

## Support

- Issues: https://github.com/LaurieRhodes/mcp-bash-go/issues
- MCP Specification: https://modelcontextprotocol.io

## License

See LICENSE file for details.
