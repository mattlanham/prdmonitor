// PRDMonitor is a terminal-based Kanban board that monitors prd.json files.
package main

import (
	"fmt"
	"os"
)

// Config holds the application configuration parsed from CLI arguments.
type Config struct {
	RootDir string
}

// ParseArgs parses command-line arguments and returns a Config.
// If no folder path is provided, it defaults to the current directory.
// Returns an error if the folder doesn't exist or isn't readable.
func ParseArgs(args []string) (*Config, error) {
	var rootDir string

	if len(args) > 1 {
		rootDir = args[1]
	} else {
		// Default to current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		rootDir = cwd
	}

	// Validate the folder exists and is readable
	if err := ValidateFolder(rootDir); err != nil {
		return nil, err
	}

	return &Config{RootDir: rootDir}, nil
}

// ValidateFolder checks that the given path exists and is a readable directory.
func ValidateFolder(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("folder does not exist: %s", path)
		}
		return fmt.Errorf("cannot access folder: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	// Check if directory is readable by attempting to open it
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("folder is not readable: %w", err)
	}
	dir.Close()

	return nil
}

func main() {
	config, err := ParseArgs(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// For now, just print the root directory to confirm parsing works
	fmt.Printf("PRDMonitor starting with root directory: %s\n", config.RootDir)
}
