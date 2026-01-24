package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/LaurieRhodes/mcp-bash-go/pkg/bash"
	"github.com/LaurieRhodes/mcp-bash-go/pkg/config"
	"github.com/LaurieRhodes/mcp-bash-go/pkg/env"
	"github.com/LaurieRhodes/mcp-bash-go/pkg/mcp"
)

func init() {
	// Ensure standard system paths are in PATH (ALWAYS runs)
	// Fixes issues when running from non-interactive shells (Claude Desktop, systemd, cron)
	// that may have minimal PATH set
	env.EnsureStandardPaths()
}

func main() {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create the bash manager
	bashManager := bash.NewBashManager(cfg.GetTimeout())
	defer bashManager.Close()

	// Set up graceful shutdown
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "Shutting down...")
		bashManager.Close()
		os.Exit(0)
	}()

	// Create and configure the MCP server
	server := mcp.NewServer(
		mcp.ServerInfo{
			Name:    "bash-mcp-server",
			Version: "1.0.0",
		},
		mcp.ServerConfig{
			Capabilities: mcp.ServerCapabilities{
				Tools: map[string]interface{}{
					"list": true,
					"call": true,
				},
			},
		},
	)

	// Set up handlers
	setupServerHandlers(server, bashManager)

	// Choose transport based on configuration
	var transport mcp.Transport
	
	if cfg.Network.Enabled {
		// Network mode
		fmt.Fprintf(os.Stderr, "Starting in NETWORK mode on %s:%d\n", cfg.Network.Host, cfg.Network.Port)
		
		netConfig, err := mcp.ParseNetworkConfig(
			cfg.Network.Host,
			cfg.Network.Port,
			cfg.Network.AllowedIPs,
			cfg.Network.AllowedSubnets,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating network config: %v\n", err)
			os.Exit(1)
		}
		
		transport, err = mcp.NewNetworkTransport(netConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating network transport: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Stdio mode (default)
		fmt.Fprintf(os.Stderr, "Starting in STDIO mode\n")
		transport = mcp.NewStdioTransport()
	}

	// Start the server with the chosen transport
	fmt.Fprintf(os.Stderr, "Bash MCP Server v1.0.0 starting\n")
	fmt.Fprintf(os.Stderr, "Command timeout: %v\n", cfg.GetTimeout())

	err = server.Connect(transport)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}

	// The server is now running
	select {}
}

// setupServerHandlers sets up the request handlers for the server
func setupServerHandlers(server *mcp.Server, bashManager *bash.BashManager) {
	// Handler for tools/list
	server.SetRequestHandler("tools/list", func(params json.RawMessage) (json.RawMessage, error) {
		tools := make([]mcp.Tool, 0, len(bash.BashTools))

		for _, toolDef := range bash.BashTools {
			inputSchema, err := json.Marshal(toolDef.InputSchema)
			if err != nil {
				continue
			}

			tools = append(tools, mcp.Tool{
				Name:        toolDef.Name,
				Description: toolDef.Description,
				InputSchema: inputSchema,
			})
		}

		response := mcp.ListToolsResponse{
			Tools: tools,
		}

		return json.Marshal(response)
	})

	// Handler for list_tools (backward compatibility)
	server.SetRequestHandler("list_tools", func(params json.RawMessage) (json.RawMessage, error) {
		handler := server.GetHandler("tools/list")
		return handler(params)
	})

	// Handler for tools/call
	server.SetRequestHandler("tools/call", func(params json.RawMessage) (json.RawMessage, error) {
		var request mcp.CallToolRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, fmt.Errorf("invalid call parameters: %w", err)
		}

		// Process the tool call with server instance for progress notifications
		return handleToolCall(request, params, bashManager, server)
	})

	// Handler for call_tool (backward compatibility)
	server.SetRequestHandler("call_tool", func(params json.RawMessage) (json.RawMessage, error) {
		handler := server.GetHandler("tools/call")
		return handler(params)
	})
}

// handleToolCall handles a tool call request
func handleToolCall(request mcp.CallToolRequest, rawParams json.RawMessage, bashManager *bash.BashManager, server *mcp.Server) (json.RawMessage, error) {
	var response mcp.CallToolResponse

	switch request.Name {
	case "bash":
		// Parse bash-specific arguments
		command, restart, err := bash.ParseBashArgs(request.Arguments)
		if err != nil {
			return createErrorResponse(err.Error())
		}

		// Restart session if requested
		if restart {
			if err := bashManager.RestartSession(); err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to restart session: %v", err))
			}
			fmt.Fprintf(os.Stderr, "Bash session restarted\n")
		}

		// Execute the command (simple, no progress notifications)
		fmt.Fprintf(os.Stderr, "Executing command: %s\n", command)
		output, err := bashManager.ExecuteCommand(command)

		if err != nil {
			return createErrorResponse(fmt.Sprintf("Command execution failed: %v", err))
		}

		response = mcp.CallToolResponse{
			Content: []mcp.ContentItem{
				{Type: "text", Text: output},
			},
		}

	default:
		return createErrorResponse(fmt.Sprintf("Unknown tool: %s", request.Name))
	}

	return json.Marshal(response)
}

// createErrorResponse creates an error response for a tool call
func createErrorResponse(message string) (json.RawMessage, error) {
	response := mcp.CallToolResponse{
		Content: []mcp.ContentItem{
			{Type: "text", Text: fmt.Sprintf("Error: %s", message)},
		},
		IsError: true,
	}

	return json.Marshal(response)
}
