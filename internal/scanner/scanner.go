// Package scanner provides functionality to discover prd.json files in directories.
package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const prdFileName = "prd.json"

// ScanProgress contains progress information during directory scanning.
type ScanProgress struct {
	DirsScanned int64  // Number of directories scanned so far
	FilesFound  int64  // Number of prd.json files found so far
	CurrentDir  string // Current directory being scanned (for display)
	IsComplete  bool   // True when scanning is finished
}

// ProgressCallback is called periodically during scanning to report progress.
type ProgressCallback func(progress ScanProgress)

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
	return ScanWithProgress(root, nil)
}

// ScanWithProgress recursively searches the given root directory for prd.json files.
// The optional progress callback is called periodically to report scanning progress.
// It returns all discovered prd.json file paths and any non-fatal errors encountered.
func ScanWithProgress(root string, progress ProgressCallback) (*ScanResult, error) {
	// Validate root directory exists
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &NotADirectoryError{Path: root}
	}

	// Use parallel scanning for better performance on large directories
	return parallelScan(root, progress)
}

// parallelScan performs parallel directory walking using a worker pool pattern.
// This significantly improves performance on large directory trees.
func parallelScan(root string, progressCb ProgressCallback) (*ScanResult, error) {
	// Number of worker goroutines (use CPU count for optimal parallelism)
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 8 {
		numWorkers = 8 // Cap at 8 to avoid too many file handles
	}

	// Channels for work distribution and result collection
	dirQueue := make(chan string, 1000) // Directories to scan
	results := make(chan string, 100)   // Found prd.json paths
	errors := make(chan error, 100)     // Non-fatal errors

	// Atomic counters for progress tracking
	var dirsScanned int64
	var filesFound int64
	var pendingDirs int64 // Tracks directories queued but not yet processed

	// Track current directory for each worker (for progress display)
	currentDirs := make([]atomic.Value, numWorkers)
	for i := range currentDirs {
		currentDirs[i].Store("")
	}

	// WaitGroup to track when all workers are done
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()
			for dir := range dirQueue {
				currentDirs[workerID].Store(dir)
				scanDirectory(dir, dirQueue, results, errors, &pendingDirs)
				atomic.AddInt64(&dirsScanned, 1)
				atomic.AddInt64(&pendingDirs, -1)

				// Check if all work is done
				if atomic.LoadInt64(&pendingDirs) == 0 {
					close(dirQueue)
				}
			}
		}()
	}

	// Start the root directory
	atomic.AddInt64(&pendingDirs, 1)
	dirQueue <- root

	// Collector goroutine for results
	var resultFiles []string
	var resultErrors []error
	var resultMu sync.Mutex
	resultDone := make(chan struct{})

	go func() {
		for {
			select {
			case file, ok := <-results:
				if !ok {
					results = nil
				} else {
					resultMu.Lock()
					resultFiles = append(resultFiles, file)
					resultMu.Unlock()
					atomic.AddInt64(&filesFound, 1)
				}
			case err, ok := <-errors:
				if !ok {
					errors = nil
				} else {
					resultMu.Lock()
					resultErrors = append(resultErrors, err)
					resultMu.Unlock()
				}
			}
			if results == nil && errors == nil {
				close(resultDone)
				return
			}
		}
	}()

	// Progress reporter goroutine
	progressDone := make(chan struct{})
	if progressCb != nil {
		go func() {
			ticker := newTicker(100) // 100ms progress updates
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C():
					// Find a current directory to report
					var currentDir string
					for i := range currentDirs {
						if d := currentDirs[i].Load().(string); d != "" {
							currentDir = d
							break
						}
					}
					progressCb(ScanProgress{
						DirsScanned: atomic.LoadInt64(&dirsScanned),
						FilesFound:  atomic.LoadInt64(&filesFound),
						CurrentDir:  currentDir,
						IsComplete:  false,
					})
				case <-progressDone:
					return
				}
			}
		}()
	}

	// Wait for all workers to finish
	wg.Wait()
	close(results)
	close(errors)

	// Wait for collector to finish
	<-resultDone

	// Stop progress reporter and send final progress
	close(progressDone)
	if progressCb != nil {
		progressCb(ScanProgress{
			DirsScanned: atomic.LoadInt64(&dirsScanned),
			FilesFound:  atomic.LoadInt64(&filesFound),
			CurrentDir:  "",
			IsComplete:  true,
		})
	}

	return &ScanResult{
		Files:  resultFiles,
		Errors: resultErrors,
	}, nil
}

// scanDirectory scans a single directory, queuing subdirectories and reporting prd.json files.
func scanDirectory(dir string, dirQueue chan<- string, results chan<- string, errors chan<- error, pendingDirs *int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		select {
		case errors <- err:
		default: // Don't block if error channel is full
		}
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)

		if entry.IsDir() {
			// Skip common large directories
			if ShouldSkipDir(name) {
				continue
			}
			// Queue subdirectory for processing
			atomic.AddInt64(pendingDirs, 1)
			select {
			case dirQueue <- path:
			default:
				// Channel full, process synchronously to avoid deadlock
				atomic.AddInt64(pendingDirs, -1)
				scanDirectory(path, dirQueue, results, errors, pendingDirs)
			}
		} else if name == prdFileName {
			// Found a prd.json file
			absPath, err := filepath.Abs(path)
			if err != nil {
				select {
				case errors <- err:
				default:
				}
				continue
			}
			select {
			case results <- absPath:
			default: // Don't block if results channel is full
			}
		}
	}
}

// ticker wraps time.Ticker for progress reporting
type ticker struct {
	t *time.Ticker
}

func newTicker(ms int) *ticker {
	return &ticker{t: time.NewTicker(time.Duration(ms) * time.Millisecond)}
}

func (t *ticker) C() <-chan time.Time {
	return t.t.C
}

func (t *ticker) Stop() {
	t.t.Stop()
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
