package bash

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
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
}

// BashManager manages bash sessions
type BashManager struct {
	session         *BashSession
	sessionMutex    sync.Mutex
	defaultTimeout  time.Duration
	containerEscape bool
}

// NewBashManager creates a new bash manager
func NewBashManager(timeout time.Duration, containerEscape bool) *BashManager {
	if timeout == 0 {
		timeout = 120 * time.Second // Default 120 second timeout
	}
	
	return &BashManager{
		defaultTimeout:  timeout,
		containerEscape: containerEscape,
	}
}

// ExecuteCommand executes a bash command in the session
func (bm *BashManager) ExecuteCommand(command string) (string, error) {
	bm.sessionMutex.Lock()
	defer bm.sessionMutex.Unlock()

	// Create session if it doesn't exist
	if bm.session == nil || !bm.session.running {
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
		timeout: bm.defaultTimeout,
		running: true,
	}

	// Create the bash command
	// If containerEscape is enabled, use nsenter to execute on the host
	if bm.containerEscape {
		// Use nsenter to escape the container and run bash on the host
		// This accesses PID 1's namespaces (the host)
		session.cmd = exec.Command("nsenter", "-t", "1", "-m", "-u", "-i", "-n", "-p", "--", "bash")
	} else {
		// Run bash within the container
		session.cmd = exec.Command("bash")
	}
	
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

	bm.session = session
	return nil
}

// execute runs a command in the bash session
func (bs *BashSession) execute(command string, timeout time.Duration) (string, error) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if !bs.running {
		return "", fmt.Errorf("bash session is not running")
	}

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

	// Read output
	outputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		var output strings.Builder
		scanner := bufio.NewScanner(bs.stdout)
		
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
			
			output.WriteString(line)
			output.WriteString("\n")
		}
		
		if err := scanner.Err(); err != nil {
			errorChan <- err
		}
	}()

	// Also capture stderr
	stderrChan := make(chan string, 1)
	go func() {
		var stderr strings.Builder
		scanner := bufio.NewScanner(bs.stderr)
		
		// Read stderr with a short buffer
		for scanner.Scan() {
			stderr.WriteString(scanner.Text())
			stderr.WriteString("\n")
		}
		
		stderrChan <- stderr.String()
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
		
		// Check if there's stderr output (non-blocking)
		select {
		case stderrOutput := <-stderrChan:
			if stderrOutput != "" {
				output = output + "\n\nSTDERR:\n" + stderrOutput
			}
		case <-time.After(100 * time.Millisecond):
			// No stderr available yet, that's OK
		}
		
		return output, nil
	}
}

// close closes the bash session
func (bs *BashSession) close() {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if !bs.running {
		return
	}

	bs.running = false

	// Close pipes
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
