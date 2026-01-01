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

// processRequests reads and processes requests from stdin
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
			
			fmt.Fprintf(os.Stderr, "Received message: %s\n", line)

			// Process the request
			response, err := handler([]byte(line))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error processing request: %v\n", err)
				continue
			}

			// If empty response, don't send anything (notification)
			if len(response) == 0 {
				continue
			}

			// Add newline to the response
			response = append(response, '\n')

			fmt.Fprintf(os.Stderr, "Sending response: %s", string(response))

			// Write the response
			_, err = t.writer.Write(response)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing response: %v\n", err)
				continue
			}
			
			// Flush the buffer
			err = t.writer.Flush()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error flushing response: %v\n", err)
				continue
			}

			fmt.Fprintf(os.Stderr, "Response sent successfully\n")
		}
	}
}
