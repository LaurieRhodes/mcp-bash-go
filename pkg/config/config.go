package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NetworkConfig holds network-specific configuration.
// This is an advanced capability with security implications and is not
// included in the default configuration. See config.network.json for
// an example of how to enable network mode.
type NetworkConfig struct {
	Enabled        bool     `json:"enabled"`
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	AllowedIPs     []string `json:"allowedIPs"`
	AllowedSubnets []string `json:"allowedSubnets"`
}

// Config holds the application configuration
type Config struct {
	CommandTimeout int            `json:"commandTimeout"` // in seconds
	Enabled        bool           `json:"enabled"`
	Network        *NetworkConfig `json:"network,omitempty"`
}

// Default config file name
const configFileName = "config.json"

// ErrBashDisabled is returned when bash tool is disabled
var ErrBashDisabled = errors.New("bash tool is disabled in configuration")

// LoadConfig loads the configuration from a JSON file
func LoadConfig() (*Config, error) {
	// Get the directory of the executable
	executablePath, err := getExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Executable directory: %s\n", executablePath)

	// Build the path to the config file
	configFilePath := filepath.Join(executablePath, configFileName)
	fmt.Fprintf(os.Stderr, "Looking for config file at: %s\n", configFilePath)

	// Check if the config file exists
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// Try in current working directory as fallback
		cwd, err := os.Getwd()
		if err == nil {
			cwdConfigPath := filepath.Join(cwd, configFileName)
			fmt.Fprintf(os.Stderr, "Config not found in executable directory, checking current directory: %s\n", cwdConfigPath)

			if _, err := os.Stat(cwdConfigPath); err == nil {
				configFilePath = cwdConfigPath
				fmt.Fprintf(os.Stderr, "Found config file in current directory\n")
			} else {
				// Create a default config if none exists
				fmt.Fprintf(os.Stderr, "No config file found, creating default in executable directory\n")
				return createDefaultConfig(configFilePath)
			}
		} else {
			fmt.Fprintf(os.Stderr, "No config file found, creating default in executable directory\n")
			return createDefaultConfig(configFilePath)
		}
	}

	// Read the config file
	fmt.Fprintf(os.Stderr, "Reading config from: %s\n", configFilePath)
	file, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the config file
	config := &Config{}
	if err := json.Unmarshal(file, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate the config
	if !config.Enabled {
		return nil, ErrBashDisabled
	}

	if config.CommandTimeout == 0 {
		config.CommandTimeout = 600 // default 10 minutes - allows longer workflows
	}

	// Set network defaults only when network mode is explicitly configured
	if config.Network != nil && config.Network.Enabled {
		if config.Network.Host == "" {
			config.Network.Host = "localhost"
		}
		if config.Network.Port == 0 {
			config.Network.Port = 3000
		}
	}

	fmt.Fprintf(os.Stderr, "Configuration loaded successfully\n")
	fmt.Fprintf(os.Stderr, "Command timeout: %d seconds\n", config.CommandTimeout)
	if config.Network != nil && config.Network.Enabled {
		fmt.Fprintf(os.Stderr, "Network mode: enabled (%s:%d)\n", config.Network.Host, config.Network.Port)
	} else {
		fmt.Fprintf(os.Stderr, "Network mode: disabled (stdio only)\n")
	}
	return config, nil
}

// GetTimeout returns the command timeout as a duration
func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.CommandTimeout) * time.Second
}

// IsNetworkEnabled returns true if network mode is explicitly enabled
func (c *Config) IsNetworkEnabled() bool {
	return c.Network != nil && c.Network.Enabled
}

// createDefaultConfig creates a default config file.
// The default config uses stdio mode only — network configuration
// is intentionally excluded for security. Users who need network
// mode should refer to config.network.json for an example.
func createDefaultConfig(configFilePath string) (*Config, error) {
	config := &Config{
		CommandTimeout: 600, // 10 minutes - allows longer workflows without progress notifications
		Enabled:        true,
		// Network is intentionally nil — not included in default config.
		// See config.network.json for network mode configuration.
	}

	// Convert config to JSON
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default config: %w", err)
	}

	// Write the config file
	if err := os.WriteFile(configFilePath, jsonData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write default config file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created default config file at %s\n", configFilePath)
	return config, nil
}

// getExecutablePath returns the directory of the current executable
func getExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realPath = execPath
	}

	return filepath.Dir(realPath), nil
}
