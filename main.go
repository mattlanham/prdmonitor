// PRDMonitor is a terminal-based Kanban board that monitors prd.json files.
package main

import (
	"fmt"
	"os"

	"lanham/prdmonitor/internal/parser"
	"lanham/prdmonitor/internal/scanner"
	"lanham/prdmonitor/internal/tui"
	"lanham/prdmonitor/internal/watcher"
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

	// Scan for prd.json files
	result, err := scanner.Scan(config.RootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	// Log any non-fatal scanning errors
	if result.HasErrors() {
		for _, scanErr := range result.Errors {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", scanErr)
		}
	}

	// Display the number of discovered projects
	fmt.Printf("PRDMonitor starting with root directory: %s\n", config.RootDir)
	fmt.Printf("Discovered %d project(s)\n", result.Count())

	// Parse all discovered prd.json files
	parseResults, parseErrors := parser.ParseFiles(result.Files)

	// Log any parsing errors
	for _, parseErr := range parseErrors {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", parseErr)
	}

	// Display parsed project information
	totalStories := 0
	for _, pr := range parseResults {
		totalStories += len(pr.PRD.UserStories)
	}
	fmt.Printf("Parsed %d project(s) with %d total user story(ies)\n", len(parseResults), totalStories)

	// Set up file watcher
	w, err := watcher.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not start file watcher: %v\n", err)
		// Continue without file watching
		if err := tui.Run(parseResults); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
		return
	}
	defer w.Stop()

	// Add all discovered prd.json files to the watcher
	if err := w.AddFiles(result.Files); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not watch some files: %v\n", err)
	}

	// Start the watcher in a goroutine
	go w.Start()

	fmt.Printf("Watching %d file(s) for changes\n", len(result.Files))

	// Launch the TUI with file watching
	if err := tui.RunWithWatcher(parseResults, w, config.RootDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
