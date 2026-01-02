# Bash MCP Server

<div align="center">

![Model Context Protocol](https://img.shields.io/badge/MCP-Bash-blue)
![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green)
[![Release](https://img.shields.io/github/v/release/LaurieRhodes/mcp-bash-go)](https://github.com/LaurieRhodes/mcp-bash-go/releases)

</div>

## 🚀 Overview

A Model Context Protocol (MCP) server implementation that provides bash command execution for AI models. Enables Large Language Models to execute bash commands in a persistent, stateful session with proper isolation and security controls.

> ⚠️ **IMPORTANT SECURITY NOTICE**: This server executes arbitrary bash commands on your system. Only use it in trusted environments and with AI models you trust. Review all commands before execution in production environments.

This server addresses the issue where Claude Sonnet models have been trained to expect a `bash_tool` as part of Anthropic's computer use feature, but this tool is not available in standard Claude Desktop MCP environments.

## 📥 Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [Releases page](https://github.com/LaurieRhodes/mcp-bash-go/releases).

**Linux (x86_64)**:

```bash
wget https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-linux-amd64
chmod +x mcp-bash-linux-amd64
sudo mkdir -p /usr/local/bin/mcp-bash 
sudo mv mcp-bash-linux-amd64 /usr/local/bin/mcp-bash/mcp-bash

# Create default config file
sudo tee /usr/local/bin/mcp-bash/config.json > /dev/null <<'EOF'
{
  "commandTimeout": 120,
  "enabled": true
}
EOF
```

**macOS (Apple Silicon)**:

```bash
wget https://github.com/LaurieRhodes/mcp-bash-go/releases/latest/download/mcp-bash-darwin-arm64
chmod +x mcp-bash-darwin-arm64
sudo mkdir -p /usr/local/bin/mcp-bash 
sudo mv mcp-bash-darwin-arm64 /usr/local/bin/mcp-bash/mcp-bash

# Create default config file
sudo tee /usr/local/bin/mcp-bash/config.json > /dev/null <<'EOF'
{
  "commandTimeout": 120,
  "enabled": true
}
EOF
```

**Windows**: Not supported natively (no bash). Use WSL (Windows Subsystem for Linux) and install the Linux binary.

### Building from Source

```bash
git clone https://github.com/LaurieRhodes/mcp-bash-go.git
cd mcp-bash-go
go build -o mcp-bash ./cmd/server
```

## ⚙️ Configuration

The server requires a `config.json` file in the same directory as the executable.

**Minimal Configuration**:

```json
{
  "commandTimeout": 120,
  "enabled": true
}
```

**Configuration Options**:

- `commandTimeout`: Maximum execution time in seconds (default: 120)
- `enabled`: Set to false to disable the bash tool entirely

### Network Mode (Advanced)

For network-based MCP communication, see `config.network.json` example:

```json
{
  "commandTimeout": 120,
  "enabled": true,
  "network": {
    "enabled": true,
    "host": "localhost",
    "port": 3000,
    "allowedIPs": ["127.0.0.1"],
    "allowedSubnets": ["192.168.1.0/24"]
  }
}
```

## 🔧 MCP Client Configuration

### Claude Desktop

Edit your Claude Desktop configuration file:

**Linux/macOS**: `~/.config/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "bash": {
      "command": "/usr/local/bin/mcp-bash/mcp-bash",
      "args": []
    }
  }
}
```

### Other MCP Clients

Point your MCP client to the bash server binary. The server communicates via stdio by default.

## 🧰 Available Tool

### bash

Execute bash commands in a persistent session.

| Parameter | Required | Type    | Description                              |
| --------- | -------- | ------- | ---------------------------------------- |
| command   | Yes      | string  | The bash command to execute              |
| restart   | No       | boolean | Set to true to restart the session first |

**Supported Features**:

- Pipelines: `ls | grep pattern`
- Command chaining: `cd /tmp && ls -la`
- Environment variables: `export VAR=value`
- Background processes: `sleep 10 &`
- File I/O redirection: `echo "text" > file.txt`
- Command substitution: `echo $(date)`
- Conditional execution: `test -f file && cat file`

**Unsupported Features**:

- Interactive commands: `vim`, `less`, `top`
- Commands requiring user input
- `sudo` without NOPASSWD configuration

## 🔒 Security Considerations

**Security Features**:

- ✅ Commands run with server process permissions only
- ✅ No sudo/root access by default
- ✅ Configurable timeout prevents infinite loops
- ✅ Session can be restarted if needed
- ✅ Optional IP whitelisting for network mode

**Security Risks**:

- ❌ No command whitelisting
- ❌ No sandboxing
- ❌ Executes arbitrary commands from AI

**Best Practices**:

- Run with minimal necessary permissions
- Use in development/testing environments
- Review AI-generated commands before production use
- Use Docker/VMs for additional isolation
- Monitor logs for unexpected commands

## 📊 Architecture

```
mcp-bash-go/
├── cmd/server/          # Server entry point
├── pkg/
│   ├── bash/            # Bash session management
│   ├── mcp/             # MCP protocol implementation
│   │   ├── types.go     # Type definitions
│   │   ├── server.go    # Server logic
│   │   ├── transport.go # Stdio transport
│   │   └── network_transport.go  # Network transport
│   └── config/          # Configuration management
└── .github/workflows/   # CI/CD automation
```

## 📚 Related Documentation

- [Anthropic Bash Tool Documentation](https://docs.claude.com/en/docs/agents-and-tools/tool-use/bash-tool)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [GitHub Issue #4027](https://github.com/cline/cline/issues/4027)

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details.

## 👏 Attribution

Addresses community-identified gaps in Claude Desktop's MCP tool availability, specifically the missing `bash_tool` that Claude models have been trained to use.

## ⚠️ Disclaimer

This tool executes arbitrary bash commands. Use responsibly and only in trusted environments. The authors are not responsible for any damage caused by misuse of this tool.

## 🤝 Contributing

Contributions welcome! Please open an issue or pull request.

---

**Platform**: Linux, macOS (use WSL on Windows)  
**Status**: Production Ready
