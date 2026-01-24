package env

import (
	"os"
	"path/filepath"
	"strings"
)

// EnsureStandardPaths adds standard system directories to PATH if not already present
// This fixes issues when running from non-interactive shells (Claude Desktop, systemd, cron)
// that don't have full PATH set
func EnsureStandardPaths() {
	currentPath := os.Getenv("PATH")
	
	// Standard paths by platform
	var standardPaths []string
	
	// Detect platform and set appropriate paths
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") || 
	   filepath.Separator == '\\' {
		// Windows standard paths
		standardPaths = []string{
			`C:\Windows\system32`,
			`C:\Windows`,
			`C:\Windows\System32\Wbem`,
			`C:\Windows\System32\WindowsPowerShell\v1.0`,
		}
	} else {
		// Unix-like (Linux, macOS) standard paths
		standardPaths = []string{
			"/usr/local/bin",
			"/usr/bin",
			"/bin",
			"/usr/local/sbin",
			"/usr/sbin",
			"/sbin",
		}
	}
	
	// Build map of existing paths for quick lookup
	existingPaths := make(map[string]bool)
	for _, p := range strings.Split(currentPath, string(os.PathListSeparator)) {
		if p != "" {
			existingPaths[p] = true
		}
	}
	
	// Add missing standard paths
	var pathsToAdd []string
	for _, stdPath := range standardPaths {
		if !existingPaths[stdPath] {
			pathsToAdd = append(pathsToAdd, stdPath)
		}
	}
	
	// Update PATH if we have paths to add
	if len(pathsToAdd) > 0 {
		// Add new paths to beginning of PATH (standard paths take precedence)
		newPath := strings.Join(pathsToAdd, string(os.PathListSeparator))
		if currentPath != "" {
			newPath = newPath + string(os.PathListSeparator) + currentPath
		}
		os.Setenv("PATH", newPath)
	}
}
