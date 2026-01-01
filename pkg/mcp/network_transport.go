package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
)

// NetworkConfig holds configuration for network transport
type NetworkConfig struct {
	Host           string
	Port           int
	AllowedIPs     []string
	AllowedSubnets []*net.IPNet
}

// NetworkTransport implements the Transport interface using TCP sockets
type NetworkTransport struct {
	config    NetworkConfig
	listener  net.Listener
	running   bool
	stopChan  chan struct{}
	waitGroup sync.WaitGroup
	mutex     sync.Mutex
	handler   RequestHandlerFunc
}

// NewNetworkTransport creates a new network transport
func NewNetworkTransport(config NetworkConfig) (*NetworkTransport, error) {
	return &NetworkTransport{
		config:   config,
		stopChan: make(chan struct{}),
	}, nil
}

// ParseNetworkConfig parses network configuration including CIDR subnets
func ParseNetworkConfig(host string, port int, allowedIPs []string, allowedSubnetStrs []string) (NetworkConfig, error) {
	config := NetworkConfig{
		Host:           host,
		Port:           port,
		AllowedIPs:     allowedIPs,
		AllowedSubnets: make([]*net.IPNet, 0, len(allowedSubnetStrs)),
	}

	for _, subnet := range allowedSubnetStrs {
		_, ipNet, err := net.ParseCIDR(subnet)
		if err != nil {
			return config, fmt.Errorf("invalid subnet %s: %w", subnet, err)
		}
		config.AllowedSubnets = append(config.AllowedSubnets, ipNet)
	}

	return config, nil
}

// Start starts the network transport
func (t *NetworkTransport) Start(handler RequestHandlerFunc) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.running {
		return fmt.Errorf("transport already running")
	}

	t.handler = handler

	addr := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	t.listener = listener
	t.running = true

	fmt.Fprintf(os.Stderr, "MCP Network Transport listening on %s\n", addr)
	if len(t.config.AllowedIPs) > 0 || len(t.config.AllowedSubnets) > 0 {
		fmt.Fprintf(os.Stderr, "IP Whitelist enabled: IPs=%v, Subnets=%v\n", 
			t.config.AllowedIPs, formatSubnets(t.config.AllowedSubnets))
	} else {
		fmt.Fprintf(os.Stderr, "WARNING: No IP restrictions configured - all connections allowed\n")
	}

	t.waitGroup.Add(1)
	go t.acceptConnections()

	return nil
}

// Stop stops the network transport
func (t *NetworkTransport) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.running {
		return nil
	}

	close(t.stopChan)
	if t.listener != nil {
		t.listener.Close()
	}
	t.waitGroup.Wait()
	t.running = false

	return nil
}

func (t *NetworkTransport) acceptConnections() {
	defer t.waitGroup.Done()

	for {
		select {
		case <-t.stopChan:
			return
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				select {
				case <-t.stopChan:
					return
				default:
					fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
					continue
				}
			}

			if !t.isIPAllowed(conn.RemoteAddr()) {
				fmt.Fprintf(os.Stderr, "Connection rejected from %s - not in whitelist\n", conn.RemoteAddr())
				conn.Close()
				continue
			}

			fmt.Fprintf(os.Stderr, "Accepted connection from %s\n", conn.RemoteAddr())
			t.waitGroup.Add(1)
			go t.handleConnection(conn)
		}
	}
}

func (t *NetworkTransport) isIPAllowed(addr net.Addr) bool {
	if len(t.config.AllowedIPs) == 0 && len(t.config.AllowedSubnets) == 0 {
		return true
	}

	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return false
	}
	
	ip := tcpAddr.IP.String()
	
	for _, allowedIP := range t.config.AllowedIPs {
		if ip == allowedIP {
			return true
		}
	}
	
	for _, subnet := range t.config.AllowedSubnets {
		if subnet.Contains(tcpAddr.IP) {
			return true
		}
	}
	
	return false
}

func (t *NetworkTransport) handleConnection(conn net.Conn) {
	defer t.waitGroup.Done()
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		select {
		case <-t.stopChan:
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Fprintf(os.Stderr, "Client %s disconnected\n", conn.RemoteAddr())
					return
				}
				return
			}

			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				continue
			}

			response, err := t.handler([]byte(line))
			if err != nil {
				errorResp := map[string]interface{}{
					"jsonrpc": "2.0",
					"error": map[string]interface{}{
						"code":    -32603,
						"message": err.Error(),
					},
				}
				errorBytes, _ := json.Marshal(errorResp)
				writer.Write(errorBytes)
				writer.Write([]byte("\n"))
				writer.Flush()
				continue
			}

			if len(response) == 0 {
				continue
			}

			response = append(response, '\n')
			writer.Write(response)
			writer.Flush()
		}
	}
}

func formatSubnets(subnets []*net.IPNet) []string {
	result := make([]string, len(subnets))
	for i, subnet := range subnets {
		result[i] = subnet.String()
	}
	return result
}
