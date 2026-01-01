package mcp

import (
	"encoding/json"
	"fmt"
)

// RequestID can be either a string or number as per JSON-RPC spec
type RequestID struct {
	value interface{}
}

// UnmarshalJSON implements custom unmarshaling for RequestID
func (r *RequestID) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a number first
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		r.value = num
		return nil
	}
	
	// Try to unmarshal as a string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		r.value = str
		return nil
	}
	
	return nil // ID can be omitted in notifications
}

// MarshalJSON implements custom marshaling for RequestID
func (r RequestID) MarshalJSON() ([]byte, error) {
	if r.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(r.value)
}

// String returns the string representation of the ID
func (r RequestID) String() string {
	if r.value == nil {
		return ""
	}
	return fmt.Sprintf("%v", r.value)
}

// IsEmpty returns true if the ID is empty/nil
func (r RequestID) IsEmpty() bool {
	return r.value == nil
}

// RequestMessage represents a request message from the client
type RequestMessage struct {
	JsonRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// ResponseMessage represents a response message sent to the client
type ResponseMessage struct {
	JsonRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorResponse  `json:"error,omitempty"`
}

// NotificationMessage represents a notification message that doesn't expect a response
type NotificationMessage struct {
	JsonRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ServerInfo information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientInfo information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams represents the parameters for the initialize request
type InitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	ClientInfo      ClientInfo      `json:"clientInfo"`
	Capabilities    json.RawMessage `json:"capabilities"`
}

// InitializeResult represents the response to the initialize request
type InitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	ServerInfo      ServerInfo      `json:"serverInfo"`
	Capabilities    json.RawMessage `json:"capabilities"`
}

// Tool represents a tool that can be called by the client
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ListToolsRequest represents a request to list available tools
type ListToolsRequest struct {
	// No parameters needed for list_tools
}

// ListToolsResponse represents a response to list_tools
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// CallToolRequest represents a request to call a tool
type CallToolRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ContentItem represents an item in the content array
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallToolResponse represents a response from calling a tool
type CallToolResponse struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// RequestHandler is a function that handles a specific request method
type RequestHandler func(params json.RawMessage) (json.RawMessage, error)

// ServerCapabilities represents the capabilities of the server
type ServerCapabilities struct {
	Tools map[string]interface{} `json:"tools"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}
