package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Server represents an MCP server
type Server struct {
	info        ServerInfo
	config      ServerConfig
	handlers    map[string]RequestHandler
	transport   Transport
	handlersMux sync.RWMutex
	initialized bool
}

// NewServer creates a new MCP server
func NewServer(info ServerInfo, config ServerConfig) *Server {
	return &Server{
		info:        info,
		config:      config,
		handlers:    make(map[string]RequestHandler),
		initialized: false,
	}
}

// SetRequestHandler sets a handler for a specific request method
func (s *Server) SetRequestHandler(method string, handler RequestHandler) {
	s.handlersMux.Lock()
	defer s.handlersMux.Unlock()
	s.handlers[method] = handler
}

// GetHandler gets a handler for a specific request method
func (s *Server) GetHandler(method string) RequestHandler {
	s.handlersMux.RLock()
	defer s.handlersMux.RUnlock()
	return s.handlers[method]
}

// Connect connects the server to a transport
func (s *Server) Connect(transport Transport) error {
	s.transport = transport
	return s.transport.Start(s.handleRequest)
}

// Disconnect disconnects the server from its transport
func (s *Server) Disconnect() error {
	if s.transport == nil {
		return nil
	}
	return s.transport.Stop()
}

// handleRequest handles incoming requests
func (s *Server) handleRequest(data []byte) ([]byte, error) {
	// Parse the request
	var request RequestMessage
	if err := json.Unmarshal(data, &request); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal request: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Handling method: %s, ID: %s\n", request.Method, request.ID.String())

	// Check if this is the initialize method
	if request.Method == "initialize" {
		fmt.Fprintf(os.Stderr, "Processing initialize request\n")
		return s.handleInitialize(request)
	}

	// Handle the initialized notification
	if request.Method == "notifications/initialized" {
		fmt.Fprintf(os.Stderr, "Received initialized notification, setting server as ready\n")
		s.initialized = true
		return nil, nil
	}

	// Handle initialized without the notifications/ prefix (just in case)
	if request.Method == "initialized" {
		fmt.Fprintf(os.Stderr, "Received initialized notification (legacy format), setting server as ready\n")
		s.initialized = true
		return nil, nil
	}

	// If not initialized and not a ping, reject the request
	if !s.initialized && request.Method != "ping" {
		fmt.Fprintf(os.Stderr, "Rejecting request %s because server is not initialized\n", request.Method)
		response := ResponseMessage{
			JsonRPC: "2.0",
			ID:      request.ID,
			Error: &ErrorResponse{
				Code:    -32002,
				Message: "Server not initialized",
			},
		}
		return json.Marshal(response)
	}

	// Get the handler for this method
	s.handlersMux.RLock()
	handler, ok := s.handlers[request.Method]
	s.handlersMux.RUnlock()

	if !ok {
		fmt.Fprintf(os.Stderr, "Method not supported: %s\n", request.Method)
		response := ResponseMessage{
			JsonRPC: "2.0",
			ID:      request.ID,
			Error: &ErrorResponse{
				Code:    -32601,
				Message: fmt.Sprintf("Method not supported: %s", request.Method),
			},
		}
		return json.Marshal(response)
	}

	// Call the handler
	fmt.Fprintf(os.Stderr, "Calling handler for method: %s\n", request.Method)
	result, err := handler(request.Params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Handler error for method %s: %v\n", request.Method, err)
		response := ResponseMessage{
			JsonRPC: "2.0",
			ID:      request.ID,
			Error: &ErrorResponse{
				Code:    -32000,
				Message: err.Error(),
			},
		}
		return json.Marshal(response)
	}

	// Return the result
	fmt.Fprintf(os.Stderr, "Handler successful for method: %s\n", request.Method)
	response := ResponseMessage{
		JsonRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}
	
	responseBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
		return nil, err
	}
	
	fmt.Fprintf(os.Stderr, "Response: %s\n", string(responseBytes))
	return responseBytes, nil
}

// handleInitialize handles the initialize method
func (s *Server) handleInitialize(request RequestMessage) ([]byte, error) {
	fmt.Fprintf(os.Stderr, "Parsing initialize params\n")
	var params InitializeParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid initialize parameters: %v\n", err)
		response := ResponseMessage{
			JsonRPC: "2.0",
			ID:      request.ID,
			Error: &ErrorResponse{
				Code:    -32602,
				Message: "Invalid initialize parameters",
			},
		}
		return json.Marshal(response)
	}

	fmt.Fprintf(os.Stderr, "Client info: %s %s\n", params.ClientInfo.Name, params.ClientInfo.Version)
	fmt.Fprintf(os.Stderr, "Protocol version: %s\n", params.ProtocolVersion)

	// Accept the client's protocol version
	protocolVersion := params.ProtocolVersion
	if protocolVersion == "" {
		protocolVersion = "2023-11-05"
	}

	// Create server info
	serverInfo := ServerInfo{
		Name:    s.info.Name,
		Version: s.info.Version,
	}

	// Create capabilities object
	capabilities := map[string]interface{}{
		"tools": map[string]interface{}{
			"list": true,
			"call": true,
		},
	}

	// Create the initialize result
	initializeResult := InitializeResult{
		ProtocolVersion: protocolVersion,
		ServerInfo:      serverInfo,
	}

	// Marshal capabilities
	capabilitiesJson, err := json.Marshal(capabilities)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal capabilities: %v\n", err)
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}
	initializeResult.Capabilities = capabilitiesJson

	// Marshal the result
	resultJson, err := json.Marshal(initializeResult)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal initialize result: %v\n", err)
		return nil, fmt.Errorf("failed to marshal initialize result: %w", err)
	}

	// Create the response message
	response := ResponseMessage{
		JsonRPC: "2.0",
		ID:      request.ID,
		Result:  resultJson,
	}

	// Marshal the response
	responseBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Initialize response: %s\n", string(responseBytes))
	
	s.initialized = true
	return responseBytes, nil
}
