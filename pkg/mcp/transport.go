package mcp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// RequestHandlerFunc is a function that processes a request and returns a response
type RequestHandlerFunc func(data []byte) ([]byte, error)

// Transport defines the interface for MCP transport mechanisms
type Transport interface {
	Start(handler RequestHandlerFunc) error
	Stop() error
}

// StdioTransport implements the Transport interface using stdin/stdout
type StdioTransport struct {
	running   bool
	stopChan  chan struct{}
	waitGroup sync.WaitGroup
	reader    *bufio.Reader
	writer    *bufio.Writer
	mutex     sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{
		reader:   bufio.NewReader(os.Stdin),
		writer:   bufio.NewWriter(os.Stdout),
		stopChan: make(chan struct{}),
	}
}

// Start starts the transport
func (t *StdioTransport) Start(handler RequestHandlerFunc) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.running {
		return fmt.Errorf("transport already running")
	}

	t.running = true
	t.waitGroup.Add(1)

	go t.processRequests(handler)

	return nil
}

// Stop stops the transport
func (t *StdioTransport) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.running {
		return nil
	}

	close(t.stopChan)
	t.waitGroup.Wait()
	t.running = false

	return nil
}

// processRequests reads and processes requests from stdin.
// Messages are read in the main loop and dispatched to goroutines so that
// notifications (e.g. notifications/cancelled) can be processed even while
// a long-running request is in flight.
func (t *StdioTransport) processRequests(handler RequestHandlerFunc) {
	defer t.waitGroup.Done()

	for {
		select {
		case <-t.stopChan:
			return
		default:
			// Read a line from stdin
			line, err := t.reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Fprintf(os.Stderr, "Received EOF from stdin, exiting\n")
					return
				}
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
				continue
			}

			// Trim the trailing newline
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				continue
			}
			
			// Log received message (truncated for large payloads)
			const maxMsgLog = 200
			if len(line) > maxMsgLog {
				fmt.Fprintf(os.Stderr, "Received message (%d bytes): %s...[truncated]\n", len(line), line[:maxMsgLog])
			} else {
				fmt.Fprintf(os.Stderr, "Received message: %s\n", line)
			}

			// Dispatch to goroutine so we can keep reading stdin.
			// This allows notifications/cancelled to be processed while
			// a long-running tools/call is still executing.
			go t.handleAndRespond(handler, []byte(line))
		}
	}
}

// handleAndRespond processes a single message and writes the response.
// Thread-safe: uses t.mutex to serialise writes to stdout.
func (t *StdioTransport) handleAndRespond(handler RequestHandlerFunc, data []byte) {
	response, err := handler(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing request: %v\n", err)
		return
	}

	// If empty response, don't send anything (notification)
	if len(response) == 0 {
		return
	}

	// Add newline to the response
	response = append(response, '\n')

	fmt.Fprintf(os.Stderr, "Sending response (%d bytes)\n", len(response))

	// Serialise writes to stdout
	t.mutex.Lock()
	defer t.mutex.Unlock()

	_, err = t.writer.Write(response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing response: %v\n", err)
		return
	}
	
	err = t.writer.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error flushing response: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Response sent successfully\n")
}
