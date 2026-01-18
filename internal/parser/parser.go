// Package parser provides functionality to parse prd.json files.
package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"lanham/prdmonitor/internal/model"
)

// ParseResult contains a successfully parsed PRD along with its source file path.
type ParseResult struct {
	PRD      *model.PRD
	FilePath string
	ModTime  time.Time // Last modification time of the source file
}

// ParseError represents an error that occurred while parsing a prd.json file.
type ParseError struct {
	FilePath string
	Err      error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse %s: %v", e.FilePath, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// ParseFile reads and parses a prd.json file from the given path.
// It returns a ParseResult on success, or a ParseError if the file
// cannot be read or contains invalid JSON.
func ParseFile(filePath string) (*ParseResult, error) {
	// Get file info for modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, &ParseError{FilePath: filePath, Err: err}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &ParseError{FilePath: filePath, Err: err}
	}

	prd, err := ParseBytes(data)
	if err != nil {
		return nil, &ParseError{FilePath: filePath, Err: err}
	}

	return &ParseResult{
		PRD:      prd,
		FilePath: filePath,
		ModTime:  fileInfo.ModTime(),
	}, nil
}

// ParseBytes parses JSON data into a PRD struct.
// It returns an error if the JSON is malformed or cannot be unmarshaled.
func ParseBytes(data []byte) (*model.PRD, error) {
	var prd model.PRD
	if err := json.Unmarshal(data, &prd); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &prd, nil
}

// ParseFiles parses multiple prd.json files and returns the results.
// Files that fail to parse are recorded in the errors slice but do not
// stop the parsing of other files.
func ParseFiles(filePaths []string) ([]*ParseResult, []error) {
	results := make([]*ParseResult, 0, len(filePaths))
	errors := make([]error, 0)

	for _, path := range filePaths {
		result, err := ParseFile(path)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		results = append(results, result)
	}

	return results, errors
}
