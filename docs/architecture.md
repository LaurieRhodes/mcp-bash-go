# Architecture Overview

## System Design

The MCP Bash Server implements a clean separation between MCP protocol handling and bash execution.

```
┌─────────────────────────────────────────────┐
│         Claude Desktop / MCP Client          │
└─────────────┬───────────────────────────────┘
              │ JSON-RPC over stdio
              ↓
┌─────────────────────────────────────────────┐
│           MCP Protocol Layer                 │
│  - Message handling                          │
│  - Tool registration                         │
│  - Transport (stdio/network)                 │
└─────────────┬───────────────────────────────┘
              │
              ↓
┌─────────────────────────────────────────────┐
│         Bash Execution Layer                 │
│  - Session management                        │
│  - Command execution                         │
│  - Timeout handling                          │
│  - Environment injection (nested MCP)        │
└─────────────────────────────────────────────┘
```

## Core Components

### 1. Transport Layer (pkg/mcp/)

**Stdio Transport (Default):**
- Reads JSON-RPC messages from stdin
- Writes responses to stdout
- Errors/logs to stderr

**Network Transport (Optional):**
- TCP/IP server on configurable port
- IP-based access control
- JSON-RPC over HTTP

### 2. Bash Manager (pkg/bash/)

**Session Management:**
- Creates persistent bash process
- Maintains environment state
- Handles session restart

**Command Execution:**
- Executes commands with timeout
- Captures stdout/stderr
- Returns exit codes

**Environment Injection:**
- Detects nested MCP scenarios
- Automatically injects environment variables
- Enables Unix socket communication

### 3. Configuration (pkg/config/)

**Timeout Configuration:**
```go
type Config struct {
    CommandTimeout time.Duration
    Network        NetworkConfig
}
```

**Network Configuration:**
```go
type NetworkConfig struct {
    Enabled        bool
    Host           string
    Port           int
    AllowedIPs     []string
    AllowedSubnets []string
}
```

## Nested MCP Solution

### The Problem

```
Claude Desktop
  ↓ [stdio]
Bash MCP Server (owns stdin/stdout)
  ↓ [executes: mcp-cli --workflow X]
mcp-cli tries to connect to skills server
  ↓ [tries stdio - CONFLICT!]
Skills Server (also wants stdin/stdout)
  ↓ DEADLOCK - both waiting on same stdio
```

### The Solution

**Automatic Detection & Environment Injection:**

```go
// pkg/bash/bash.go
func (m *BashManager) ExecuteCommand(command string) (string, error) {
    // Create command
    cmd := exec.Command("bash", "-c", command)
    
    // CRITICAL: Inject environment variables for nested MCP
    cmd.Env = append(os.Environ(),
        "MCP_NESTED=1",
        "MCP_SOCKET_DIR=/tmp/mcp-sockets",
        "MCP_SKILLS_SOCKET=/tmp/mcp-sockets/skills.sock",
    )
    
    // Execute...
}
```

**Result:**
```
Claude Desktop
  ↓ [stdio]
Bash MCP Server
  ├─ Sets MCP_NESTED=1
  └─ [executes: mcp-cli --workflow X]
      ↓
mcp-cli detects MCP_NESTED=1
  ↓ [connects via Unix socket instead!]
Skills Server (listening on both stdio + Unix socket)
  ✓ NO CONFLICT - different channels!
```

### Why This Works

1. **Zero configuration** - Automatic environment detection
2. **Backward compatible** - Non-nested execution unchanged
3. **Secure** - Unix sockets use filesystem permissions
4. **Universal** - Works for any nested MCP scenario

## Data Flow

### Normal Command Execution

```
1. Claude sends: {"method": "tools/call", "params": {"name": "bash", "arguments": {"command": "ls"}}}
2. MCP layer parses request
3. Bash manager executes: bash -c "ls"
4. Output captured
5. Response: {"content": [{"type": "text", "text": "file1\nfile2"}]}
```

### Nested MCP Execution

```
1. Claude sends: bash tool → mcp-cli --workflow X
2. Bash manager executes with environment:
   - MCP_NESTED=1
   - MCP_SOCKET_DIR=/tmp/mcp-sockets
   - MCP_SKILLS_SOCKET=/tmp/mcp-sockets/skills.sock
3. mcp-cli detects MCP_NESTED=1
4. mcp-cli connects to skills via Unix socket
5. Workflow executes successfully
6. Output returned to Claude
```

## Security Considerations

### Stdio Mode (Default)
- **Secure**: Only accessible to parent process
- **Isolated**: No network exposure
- **Recommended**: For Claude Desktop integration

### Network Mode
- **Authentication**: None - relies on IP filtering
- **Access Control**: allowedIPs and allowedSubnets
- **Encryption**: None - use SSH tunnel or VPN
- **Use Case**: Cross-machine MCP connections only

### Unix Sockets (Nested MCP)
- **Permissions**: 0600 (owner only)
- **Location**: /tmp/mcp-sockets/*.sock
- **Access**: Filesystem-based ACL
- **Security**: Same as file permissions

## Performance Characteristics

### Timeout Handling
- Default: 600 seconds (10 minutes)
- Configurable per deployment
- Prevents hanging on infinite loops

### Session Persistence
- One bash process per server instance
- State maintained across commands
- Can be reset with restart parameter

### Resource Usage
- Memory: ~5-10 MB per instance
- CPU: Minimal when idle
- Disk: Negligible (no state files)

## Extension Points

### Adding New Tools

```go
// pkg/bash/tools.go
var BashTools = []ToolDefinition{
    {
        Name:        "bash",
        Description: "Execute bash commands",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "command": map[string]interface{}{
                    "type": "description": "The bash command to execute",
                },
            },
            "required": []string{"command"},
        },
    },
}
```

### Custom Transport

```go
// Implement mcp.Transport interface
type CustomTransport struct {
    // ...
}

func (t *CustomTransport) ReadMessage() ([]byte, error) { }
func (t *CustomTransport) WriteMessage([]byte) error { }
func (t *CustomTransport) Close() error { }
```

## Future Enhancements

- [ ] Command history/logging
- [ ] Multiple concurrent sessions
- [ ] Shell selection (bash/zsh/fish)
- [ ] Output streaming
- [ ] Authentication for network mode
- [ ] TLS support for network transport
