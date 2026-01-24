# Changelog

All notable changes to the MCP Bash Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-01-23

### Added - Nested MCP Solution

This release solves the critical nested MCP deadlock problem that prevented workflows from executing when called through the bash tool.

#### New Features

- **Automatic environment injection** in bash command execution
  
  - `MCP_NESTED=1` - Signals nested MCP context
  - `MCP_SOCKET_DIR=/tmp/mcp-sockets` - Socket directory location
  - `MCP_SKILLS_SOCKET=/tmp/mcp-sockets/skills.sock` - Specific socket path

- **Unix socket support** for nested MCP communication
  
  - Avoids stdio conflicts when multiple MCP tools interact
  - Secure filesystem-based permissions (0600)
  - Dual-mode operation (stdio + Unix socket simultaneously)

- **Auto-detection in workflow execution**
  
  - mcp-cli automatically detects `MCP_NESTED=1`
  - Switches from stdio to Unix socket when appropriate
  - Falls back to stdio if socket unavailable

# 

### Changed

- Enhanced bash command execution to support nested MCP scenarios
- Improved error handling for tool execution in nested contexts

### Fixed

- **Critical:** Resolved stdio deadlock when bash tool executes workflows
- Fixed hanging workflows that call other MCP tools
- Improved connection handling for nested tool execution

## [1.0.0] - 2026-01-02

### Added - Initial Release

#### Core Features

- **Persistent bash sessions** - Commands maintain state across executions
- **Configurable timeouts** - Prevent hanging commands (default: 600 seconds)
- **MCP protocol implementation** - Full Model Context Protocol support
- **Stdio transport** - Integration with Claude Desktop
- **Network mode** (optional) - TCP/IP connectivity with IP filtering
- **Progress notifications** - Real-time command execution feedback

#### Configuration

- JSON-based configuration file support
- Timeout configuration
- Network mode settings with IP allowlisting
- Subnet-based access control

#### Session Management

- Persistent bash process across commands
- Environment state preservation
- Session restart capability
- Working directory persistence

#### Security

- Stdio mode by default (no network exposure)
- Optional network mode with IP filtering
- Subnet-based access control
- Configurable timeouts to prevent resource exhaustion

---

## Version History Summary

| Version | Date       | Key Achievement                    |
| ------- | ---------- | ---------------------------------- |
| 1.1.0   | 2026-01-23 | **Nested MCP deadlock solved** 🎉  |
| 1.0.0   | 2026-01-02 | Initial release with core features |

## Upgrading

### From 1.0.0 to 1.1.0

**No breaking changes!** This is a backward-compatible enhancement.

**To get nested MCP support:**

1. Rebuild and deploy the bash server:
   
   ```bash
   cd /media/laurie/Data/Github/mcp-bash-go
   go build -o mcp-bash ./cmd/server
   sudo cp mcp-bash /usr/local/bin/mcp-bash/mcp-bash
   ```

2. Configure skills server for Unix socket (add to Claude config):
   
   ```json
   "skills": {
     "command": "/path/to/mcp-cli",
     "args": ["serve", "config.yaml"],
     "env": {
       "MCP_SOCKET_PATH": "/tmp/mcp-sockets/skills.sock"
     }
   }
   ```

3. Restart Claude Desktop

4. Verify:
   
   ```bash
   # Through Claude's bash tool:
   env | grep MCP_NESTED
   # Should output: MCP_NESTED=1
   ```

**Existing functionality continues to work exactly as before.**

## Future Roadmap

Planned features for future releases:

- [ ] Multiple concurrent bash sessions
- [ ] Shell selection (bash/zsh/fish)
- [ ] Command history and logging
- [ ] Output streaming for long-running commands
- [ ] TLS support for network mode
- [ ] Authentication for network mode
- [ ] Multiple Unix socket support for multiple servers
- [ ] Automatic socket cleanup on shutdown
- [ ] Metrics and monitoring endpoints

## Links

- [Documentation](docs/README.md)
- [Issue Tracker](https://github.com/LaurieRhodes/mcp-bash-go/issues)
- [MCP Specification](https://modelcontextprotocol.io)
