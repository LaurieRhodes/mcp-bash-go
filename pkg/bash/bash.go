package bash

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	// MaxScannerBufferSize is the maximum size for the bufio.Scanner buffer.
	// Default bufio.Scanner limit is 64KB which can be exceeded by commands
	// producing long output lines (e.g., raw JSON from APIs).
	MaxScannerBufferSize = 1024 * 1024 // 1MB

	// MaxOutputSize is the maximum size of captured command output.
	// Prevents unbounded memory growth from commands producing huge output.
	MaxOutputSize = 512 * 1024 // 512KB
)

// BashSession represents a persistent bash session
type BashSession struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	mutex        sync.Mutex
	sessionMutex sync.RWMutex
	running      bool
	workingDir   string
	timeout      time.Duration

	// stderrBuf holds accumulated stderr output between commands.
	// A single persistent goroutine drains stderr into this buffer,
	// avoiding the goroutine-per-execute leak.
	stderrBuf   strings.Builder
	stderrMutex sync.Mutex
	stderrDone  chan struct{} // closed when stderr drainer goroutine exits
}

// BashManager manages bash sessions
type BashManager struct {
	session        *BashSession
	sessionMutex   sync.Mutex
	defaultTimeout time.Duration
}

// NewBashManager creates a new bash manager
func NewBashManager(timeout time.Duration) *BashManager {
	if timeout == 0 {
		timeout = 600 * time.Second // Default 10 minute timeout
	}

	return &BashManager{
		defaultTimeout: timeout,
	}
}

// ExecuteCommand executes a bash command in the session
func (bm *BashManager) ExecuteCommand(command string) (string, error) {
	bm.sessionMutex.Lock()
	defer bm.sessionMutex.Unlock()

	// Create session if it doesn't exist or is dead
	if bm.session == nil || !bm.session.running {
		// FIX: Clean up the old session before creating a new one.
		// Previously, createSession() silently overwrote bm.session,
		// leaving the old bash process running as an orphan.
		if bm.session != nil {
			fmt.Fprintf(os.Stderr, "Cleaning up dead session before creating new one (PID: %d)\n",
				bm.session.getPID())
			bm.session.close()
		}
		if err := bm.createSession(); err != nil {
			return "", fmt.Errorf("failed to create bash session: %w", err)
		}
	}

	return bm.session.execute(command, bm.defaultTimeout)
}

// RestartSession restarts the bash session
func (bm *BashManager) RestartSession() error {
	bm.sessionMutex.Lock()
	defer bm.sessionMutex.Unlock()

	// Close existing session
	if bm.session != nil {
		bm.session.close()
	}

	// Create new session
	return bm.createSession()
}

// createSession creates a new bash session
func (bm *BashManager) createSession() error {
	session := &BashSession{
		timeout:    bm.defaultTimeout,
		running:    true,
		stderrDone: make(chan struct{}),
	}

	// Create the bash command
	session.cmd = exec.Command("bash")

	// NESTED MCP SUPPORT: Set environment variables for child processes
	// This allows mcp-cli to detect nested execution and use Unix sockets
	socketDir := "/tmp/mcp-sockets"
	os.MkdirAll(socketDir, 0700) // Create socket directory with restrictive permissions

	session.cmd.Env = append(os.Environ(),
		"MCP_NESTED=1",                                // Signal nested MCP execution
		"MCP_SOCKET_DIR="+socketDir,                    // Unix socket directory
		"MCP_SKILLS_SOCKET="+socketDir+"/skills.sock",  // Skills server socket path
	)

	// Get stdin/stdout/stderr pipes
	var err error
	session.stdin, err = session.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	session.stdout, err = session.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	session.stderr, err = session.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the bash process
	if err := session.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start bash: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created new bash session (PID: %d)\n", session.cmd.Process.Pid)

	// FIX: Start a single persistent stderr drainer goroutine per session.
	// Previously, execute() spawned a new goroutine per command that competed
	// to read from the same stderr pipe and never terminated, leaking goroutines
	// and causing data races.
	go session.drainStderr()

	bm.session = session
	return nil
}

// drainStderr continuously reads stderr from the bash process into a buffer.
// This single goroutine replaces the per-execute goroutine that was leaking.
func (bs *BashSession) drainStderr() {
	defer close(bs.stderrDone)

	scanner := bufio.NewScanner(bs.stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), MaxScannerBufferSize)

	for scanner.Scan() {
		line := scanner.Text()
		bs.stderrMutex.Lock()
		// Cap stderr buffer to prevent unbounded growth
		if bs.stderrBuf.Len() < MaxOutputSize {
			bs.stderrBuf.WriteString(line)
			bs.stderrBuf.WriteString("\n")
		}
		bs.stderrMutex.Unlock()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Stderr drainer error: %v\n", err)
	}
}

// consumeStderr returns and clears the accumulated stderr output.
func (bs *BashSession) consumeStderr() string {
	bs.stderrMutex.Lock()
	defer bs.stderrMutex.Unlock()
	s := bs.stderrBuf.String()
	bs.stderrBuf.Reset()
	return s
}

// getPID returns the process ID of the bash session, or 0 if not available.
func (bs *BashSession) getPID() int {
	if bs.cmd != nil && bs.cmd.Process != nil {
		return bs.cmd.Process.Pid
	}
	return 0
}

// execute runs a command in the bash session
func (bs *BashSession) execute(command string, timeout time.Duration) (string, error) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if !bs.running {
		return "", fmt.Errorf("bash session is not running")
	}

	// Clear any accumulated stderr from previous commands
	bs.consumeStderr()

	// Create a unique marker for command completion
	marker := fmt.Sprintf("__BASH_CMD_DONE_%d__", time.Now().UnixNano())

	// Construct command with marker and error capture
	fullCommand := fmt.Sprintf("%s\necho '%s'$?\n", command, marker)

	// Write command to bash
	if _, err := bs.stdin.Write([]byte(fullCommand)); err != nil {
		bs.running = false
		return "", fmt.Errorf("failed to write command: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Read stdout until we see the completion marker
	outputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		var output strings.Builder
		truncated := false
		scanner := bufio.NewScanner(bs.stdout)
		// FIX: Increase scanner buffer to handle long output lines.
		// Default 64KB limit caused "token too long" errors with large
		// JSON output from commands like curl piped through python3.
		scanner.Buffer(make([]byte, 0, 64*1024), MaxScannerBufferSize)

		for scanner.Scan() {
			line := scanner.Text()

			// Check if this is our completion marker
			if strings.HasPrefix(line, marker) {
				// Extract exit code
				exitCode := strings.TrimPrefix(line, marker)
				if exitCode != "0" {
					output.WriteString(fmt.Sprintf("\n[Exit code: %s]", exitCode))
				}
				outputChan <- output.String()
				return
			}

			// FIX: Cap output size to prevent unbounded memory growth
			if !truncated && output.Len() < MaxOutputSize {
				output.WriteString(line)
				output.WriteString("\n")
			} else if !truncated {
				truncated = true
				output.WriteString(fmt.Sprintf("\n... [output truncated at %d bytes] ...\n", MaxOutputSize))
			}
		}

		if err := scanner.Err(); err != nil {
			errorChan <- err
			return
		}
		// Scanner finished without finding marker — pipe was closed
		errorChan <- fmt.Errorf("stdout closed before command completion marker was received")
	}()

	// Wait for completion or timeout
	select {
	case <-ctx.Done():
		bs.running = false
		return "", fmt.Errorf("command timed out after %v", timeout)
	case err := <-errorChan:
		bs.running = false
		return "", fmt.Errorf("error reading output: %w", err)
	case output := <-outputChan:
		// Trim trailing newline
		output = strings.TrimRight(output, "\n")

		// Give stderr a brief moment to flush, then collect it
		time.Sleep(50 * time.Millisecond)
		stderrOutput := bs.consumeStderr()
		if stderrOutput != "" {
			output = output + "\n\nSTDERR:\n" + stderrOutput
		}

		return output, nil
	}
}

// close closes the bash session and kills the process.
// Safe to call multiple times and on sessions where running is already false.
func (bs *BashSession) close() {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	wasRunning := bs.running
	bs.running = false

	pid := bs.getPID()

	// Close pipes — this will also unblock the stderr drainer goroutine
	if bs.stdin != nil {
		bs.stdin.Close()
	}
	if bs.stdout != nil {
		bs.stdout.Close()
	}
	if bs.stderr != nil {
		bs.stderr.Close()
	}

	// Kill the process
	if bs.cmd != nil && bs.cmd.Process != nil {
		bs.cmd.Process.Kill()
		bs.cmd.Wait()
	}

	// Wait for the stderr drainer to finish (with a timeout to avoid hanging)
	if bs.stderrDone != nil {
		select {
		case <-bs.stderrDone:
		case <-time.After(2 * time.Second):
			fmt.Fprintf(os.Stderr, "Warning: stderr drainer did not exit within timeout\n")
		}
	}

	if wasRunning {
		fmt.Fprintf(os.Stderr, "Closed bash session (PID: %d)\n", pid)
	} else {
		fmt.Fprintf(os.Stderr, "Cleaned up dead bash session (PID: %d)\n", pid)
	}
}

// Close closes the bash manager and all sessions
func (bm *BashManager) Close() {
	bm.sessionMutex.Lock()
	defer bm.sessionMutex.Unlock()

	if bm.session != nil {
		bm.session.close()
		bm.session = nil
	}
}

// Tool schemas

// BashToolSchema defines the schema for bash_tool input
var BashToolSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"command": map[string]interface{}{
			"type":        "string",
			"description": "The bash command to execute",
		},
		"restart": map[string]interface{}{
			"type":        "boolean",
			"description": "Set to true to restart the bash session before executing the command",
		},
	},
	"required": []string{"command"},
}

// BashTool defines the bash tool
type BashTool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

// BashTools is the tool definition
var BashTools = map[string]BashTool{
	"bash": {
		Name: "bash",
		Description: "Execute bash commands in a persistent session. Commands are executed in a stateful bash environment " +
			"where environment variables, working directory changes, and other session state persist between calls. " +
			"Long-running commands will timeout after 120 seconds by default. " +
			"Use 'restart: true' to start a fresh session if needed. " +
			"Supports: pipelines, environment variables, cd commands, command chaining with && or ||, " +
			"background processes, file I/O redirection, and most bash built-ins. " +
			"Avoid: interactive commands (vim, less, top), commands requiring user input, sudo without NOPASSWD.",
		InputSchema: BashToolSchema,
	},
}

// Argument parsing

// ParseBashArgs parses arguments for bash tool
func ParseBashArgs(args json.RawMessage) (command string, restart bool, err error) {
	var params struct {
		Command string `json:"command"`
		Restart bool   `json:"restart"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return "", false, fmt.Errorf("invalid arguments for bash tool: %w", err)
	}

	if params.Command == "" {
		return "", false, fmt.Errorf("command parameter is required")
	}

	return params.Command, params.Restart, nil
}
