// Package scanner provides functionality to discover prd.json files in directories.
package scanner

import (
	"os"
	"path/filepath"
)

const prdFileName = "prd.json"

// SkipDirs contains directory names that should be skipped during scanning.
// These are commonly large directories that won't contain prd.json files.
var SkipDirs = map[string]bool{
	"node_modules":  true,
	".git":          true,
	".hg":           true,
	".svn":          true,
	"vendor":        true,
	".idea":         true,
	".vscode":       true,
	"__pycache__":   true,
	".cache":        true,
	".npm":          true,
	".yarn":         true,
	"dist":          true,
	"build":         true,
	".next":         true,
	".nuxt":         true,
	"coverage":      true,
	".tox":          true,
	".pytest_cache": true,
	"venv":          true,
	".venv":         true,
	"env":           true,
	".env":          true,
	"Pods":          true,
	"DerivedData":   true,
	".gradle":       true,
	"target":        true,
	"bin":           true,
	"obj":           true,
	".terraform":    true,
	".cargo":        true,
	"pkg":           true,
	"Carthage":      true,
}

// ScanResult contains the results of scanning a directory for prd.json files.
type ScanResult struct {
	// Files is a list of absolute paths to discovered prd.json files.
	Files []string
	// Errors contains any non-fatal errors encountered during scanning.
	Errors []error
}

// Scan recursively searches the given root directory for prd.json files.
// It returns all discovered prd.json file paths and any non-fatal errors encountered.
// Directories that cannot be read are skipped with errors recorded.
func Scan(root string) (*ScanResult, error) {
	result := &ScanResult{
		Files:  make([]string, 0),
		Errors: make([]error, 0),
	}

	// Validate root directory exists
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &NotADirectoryError{Path: root}
	}

	// Walk the directory tree
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Record the error but continue walking
			result.Errors = append(result.Errors, err)
			return nil
		}

		// Skip common large directories that won't contain prd.json files
		if d.IsDir() {
			if ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if this is a prd.json file
		if d.Name() == prdFileName {
			absPath, err := filepath.Abs(path)
			if err != nil {
				result.Errors = append(result.Errors, err)
				return nil
			}
			result.Files = append(result.Files, absPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Count returns the number of discovered prd.json files.
func (r *ScanResult) Count() int {
	return len(r.Files)
}

// HasErrors returns true if any errors were encountered during scanning.
func (r *ScanResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// NotADirectoryError is returned when the root path is not a directory.
type NotADirectoryError struct {
	Path string
}

func (e *NotADirectoryError) Error() string {
	return "path is not a directory: " + e.Path
}

// ShouldSkipDir returns true if the directory name should be skipped during scanning.
// This helps avoid scanning large directories that won't contain prd.json files.
func ShouldSkipDir(name string) bool {
	return SkipDirs[name]
}
