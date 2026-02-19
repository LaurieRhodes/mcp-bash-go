# MCP Bash Server

Bash command execution for Claude through the Model Context Protocol (MCP).

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![CI Status](https://github.com/LaurieRhodes/mcp-bash-go/workflows/CI/badge.svg)](https://github.com/LaurieRhodes/mcp-bash-go/actions)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/LaurieRhodes/mcp-bash-go)](https://github.com/LaurieRhodes/mcp-bash-go/releases)

## Features

✅ **Persistent bash sessions** - Commands maintain state  
✅ **Automatic timeout handling** - Configurable command timeouts  
✅ **Nested MCP support** - Run workflows that call other MCP tools  
✅ **Zero configuration** - Automatic environment injection  
✅ **Secure Unix sockets** - Filesystem-based access control  
✅ **Network mode (optional)** - TCP/IP with IP filtering  
✅ **Multi-platform releases** - Pre-built binaries for Linux and macOS

## Platform Support

| Platform | Architecture          | Status      | Download                                                                      |
| -------- | --------------------- | ----------- | ----------------------------------------------------------------------------- |
| Linux    | x86_64 (amd64)        | ✅ Supported | [Latest Release](https://github.com/LaurieRhodes/mcp-bash-go/releases/latest) |
| Linux    | ARM64                 | ✅ Supported | [Latest Release](https://github.com/LaurieRhodes/mcp-bash-go/releases/latest) |
| macOS    | Intel (amd64)         | ✅ Supported | [Latest Release](https://github.com/LaurieRhodes/mcp-bash-go/releases/latest) |
| macOS    | Apple Silicon (arm64) | ✅ Supported | [Latest Release](https://github.com/LaurieRhodes/mcp-bash-go/releases/latest) |
| Windows  | Any                   | ❌ Use WSL   | Install Linux binary in WSL                                                   |

## Quick Start

### Option 1: Download Pre-Built Binary (Recommended)

**Linux x86_64:**

```bash
# Download latest release
wget https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-linux-amd64


# Install
sudo mkdir -p /usr/local/bin/mcp-bash
sudo mv mcp-bash-linux-amd64 /usr/local/bin/mcp-bash/mcp-bash
sudo chmod +x /usr/local/bin/mcp-bash/mcp-bash

# Create config file
sudo tee /usr/local/bin/mcp-bash/config.json > /dev/null <<'EOF'
{
  "commandTimeout": 600
}
EOF
```

**macOS (Intel):**

```bash
wget https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-darwin-amd64
sudo mkdir -p /usr/local/bin/mcp-bash
sudo mv mcp-bash-darwin-amd64 /usr/local/bin/mcp-bash/mcp-bash
sudo chmod +x /usr/local/bin/mcp-bash/mcp-bash
sudo tee /usr/local/bin/mcp-bash/config.json > /dev/null <<'EOF'
{
  "commandTimeout": 600
}
EOF
```

**macOS (Apple Silicon):**

```bash
wget https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-darwin-arm64
sudo mkdir -p /usr/local/bin/mcp-bash
sudo mv mcp-bash-darwin-arm64 /usr/local/bin/mcp-bash/mcp-bash
sudo chmod +x /usr/local/bin/mcp-bash/mcp-bash
sudo tee /usr/local/bin/mcp-bash/config.json > /dev/null <<'EOF'
{
  "commandTimeout": 600
}
EOF
```

### Option 2: Build from Source

**Prerequisites:** Go 1.21 or later

```bash
# Clone repository
git clone https://github.com/LaurieRhodes/mcp-bash-go.git
cd mcp-bash-go

# Build
go build -o mcp-bash ./cmd/server

# Deploy
sudo mkdir -p /usr/local/bin/mcp-bash
sudo cp mcp-bash /usr/local/bin/mcp-bash/mcp-bash

# Create config
sudo tee /usr/local/bin/mcp-bash/config.json > /dev/null <<'EOF'
{
  "commandTimeout": 600
}
EOF
```

### Configure Claude Desktop (or other MCP Client)

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

### Install the Claude Skill (claude.ai)

When using this MCP server with claude.ai, Claude has access to both its built-in sandboxed bash **and** the MCP bash tool. Without guidance, Claude defaults to the sandbox — which runs in an isolated container with no access to your real filesystem. The included Claude skill overrides this behaviour so that all bash execution is routed through the MCP server.

1. In claude.ai, navigate to **Settings → Profile → Claude Skills**
2. Upload `claude-skill/bash-preference.zip`
3. The skill takes effect immediately for new conversations

Without this skill, commands like `ls /home` will silently run inside a throwaway container instead of on your machine.

### Restart Claude Desktop

Close and reopen Claude Desktop to load the bash server.

### Verify Installation

```bash
# Check version
/usr/local/bin/mcp-bash/mcp-bash --version

# Through Claude's bash tool:
# "Run: echo 'Hello from bash!'"
```

## Documentation

📖 **[Full Documentation](docs/README.md)**

- [Quick Start Guide](docs/quickstart.md) - Get running in 5 minutes
- [Architecture Overview](docs/architecture.md) - System design and components
- [Nested MCP Solution](docs/nested-mcp.md) - How we solved the deadlock problem
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues and solutions

The Nested MCP Problem (Solved!)

**Problem:** When Claude used the bash tool to execute workflows that called other MCP tools, both tried to use stdin/stdout simultaneously, causing a deadlock.

**Solution:** Automatic environment injection + Unix socket fallback.

```
Claude Desktop
  ↓ [stdio]
Bash Server (sets MCP_NESTED=1)
  ↓ [executes workflow]
mcp-cli (detects nested context)
  ↓ [uses Unix socket instead]
Skills Server
  ✓ No conflict!
```

**Result:** Workflows complete in ~46 seconds instead of hanging indefinitely.

See [Nested MCP Documentation](docs/nested-mcp.md) for details.

## Configuration

### Timeout (Optional)

Create `/usr/local/bin/mcp-bash/config.json`:

```json
{
  "commandTimeout": 1800
}
```

Default: 600 seconds (10 minutes)

### Network Mode (Advanced)

```json
{
  "commandTimeout": 600,
  "network": {
    "enabled": true,
    "host": "127.0.0.1",
    "port": 8080,
    "allowedIPs": ["127.0.0.1"],
    "allowedSubnets": ["192.168.1.0/24"]
  }
}
```

**Warning:** Network mode exposes bash execution over TCP/IP. Use IP filtering - unauthenticated!

## Project Structure

```
mcp-bash-go/
├── .github/
│   └── workflows/
│       ├── ci.yml          # Automated testing & linting
│       └── release.yml     # Multi-platform binary builds
├── claude-skill/          # Claude skill to prefer MCP bash over sandbox
│   └── bash-preference.zip
├── cmd/server/             # Main server entry point
├── pkg/
│   ├── bash/              # Bash execution and environment injection
│   ├── config/            # Configuration management
│   ├── env/               # Environment variable handling
│   └── mcp/               # MCP protocol implementation
├── docs/                  # Documentation
│   ├── README.md          # Documentation index
│   ├── architecture.md    # System design
├── config.json            # Default configuration
├── config.network.json    # Network mode example
├── CHANGELOG.md           # Version history
└── LICENSE                # MIT license
```



[![CI Status](https://github.com/LaurieRhodes/mcp-bash-go/workflows/CI/badge.svg)](https://github.com/LaurieRhodes/mcp-bash-go/actions)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Model Context Protocol](https://modelcontextprotocol.io) - MCP specification
- [Anthropic](https://anthropic.com) - Claude Desktop and MCP ecosystem

## Support

- **Documentation:** [docs/](docs/)
- **Issues:** https://github.com/LaurieRhodes/mcp-bash-go/issues
- **Discussions:** https://github.com/LaurieRhodes/mcp-bash-go/discussions

---

**Made with ❤️ for the MCP community**
