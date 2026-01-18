// Package watcher provides file system watching for prd.json files using fsnotify.
package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"lanham/prdmonitor/internal/scanner"
)

const prdFileName = "prd.json"

// MaxWatches is the maximum number of directories to watch.
// This prevents resource exhaustion when scanning large directory trees.
// On macOS the default limit is ~256, on Linux it's configurable but we use
// a reasonable default that should work on most systems.
const MaxWatches = 1000

// ErrTooManyWatches is returned when the maximum number of watches is exceeded.
var ErrTooManyWatches = fmt.Errorf("too many directories to watch (limit: %d)", MaxWatches)

// resolvePath returns the absolute path with symlinks resolved.
// This ensures consistent path comparison on systems like macOS where
// /var is a symlink to /private/var.
func resolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	// Resolve symlinks to get the canonical path
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the file doesn't exist yet (e.g., for new files), return abs path
		if os.IsNotExist(err) {
			// Try resolving the parent directory instead
			dir := filepath.Dir(absPath)
			resolvedDir, dirErr := filepath.EvalSymlinks(dir)
			if dirErr != nil {
				return absPath, nil
			}
			return filepath.Join(resolvedDir, filepath.Base(absPath)), nil
		}
		return absPath, nil
	}
	return resolved, nil
}

// Event represents a file change event.
type Event struct {
	FilePath string
	Op       Operation
}

// Operation represents the type of file system operation.
type Operation int

const (
	// OpModify indicates a file was modified.
	OpModify Operation = iota
	// OpCreate indicates a new file was created.
	OpCreate
	// OpDelete indicates a file was deleted.
	OpDelete
)

// String returns the string representation of the operation.
func (op Operation) String() string {
	switch op {
	case OpModify:
		return "modify"
	case OpCreate:
		return "create"
	case OpDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Watcher monitors prd.json files for changes.
type Watcher struct {
	fsWatcher     *fsnotify.Watcher
	events        chan Event
	errors        chan error
	done          chan struct{}
	files         map[string]bool // Set of watched files
	directories   map[string]bool // Set of watched directories (for new file detection)
	mu            sync.RWMutex
	debounce      time.Duration // Debounce duration for rapid events
	watchNewDirs  bool          // Whether to detect new prd.json files in directories
	stopped       bool          // Whether the watcher has been stopped
	watchCount    int           // Number of directories being watched
	watchLimitHit bool          // Whether the watch limit was reached
}

// New creates a new Watcher.
func New() (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fsWatcher:    fsWatcher,
		events:       make(chan Event, 100),
		errors:       make(chan error, 10),
		done:         make(chan struct{}),
		files:        make(map[string]bool),
		directories:  make(map[string]bool),
		debounce:     100 * time.Millisecond,
		watchNewDirs: false,
	}, nil
}

// WatchDirectory recursively watches a directory tree for new prd.json files.
// This enables detection of new prd.json files in existing and new subdirectories.
// It skips common large directories (node_modules, .git, etc.) and limits the
// total number of watches to prevent resource exhaustion.
func (w *Watcher) WatchDirectory(rootDir string) error {
	absRoot, err := resolvePath(rootDir)
	if err != nil {
		return err
	}

	// Walk the directory tree and watch all directories
	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable directories but continue walking
			return nil
		}

		if d.IsDir() {
			// Skip common large directories that won't contain prd.json files
			if scanner.ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}

			// Check if we've hit the watch limit
			w.mu.RLock()
			hitLimit := w.watchLimitHit
			w.mu.RUnlock()
			if hitLimit {
				return filepath.SkipDir
			}

			addErr := w.addDirectory(path)
			if addErr != nil {
				// If we can't add more watches, mark limit hit but continue
				// so we can still watch what we have
				w.mu.Lock()
				w.watchLimitHit = true
				w.mu.Unlock()
				return filepath.SkipDir
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	w.mu.Lock()
	w.watchNewDirs = true
	limitHit := w.watchLimitHit
	w.mu.Unlock()

	if limitHit {
		return ErrTooManyWatches
	}

	return nil
}

// addDirectory adds a directory to be watched for new prd.json files.
// Returns an error if the watch limit is exceeded or fsnotify fails.
func (w *Watcher) addDirectory(dirPath string) error {
	absPath, err := resolvePath(dirPath)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Already watching this directory
	if w.directories[absPath] {
		return nil
	}

	// Check if we've hit the watch limit
	if w.watchCount >= MaxWatches {
		w.watchLimitHit = true
		return ErrTooManyWatches
	}

	if err := w.fsWatcher.Add(absPath); err != nil {
		// Mark limit hit if it looks like a resource limit error
		w.watchLimitHit = true
		return err
	}

	w.directories[absPath] = true
	w.watchCount++
	return nil
}

// AddFiles adds multiple files to the watcher.
func (w *Watcher) AddFiles(files []string) error {
	for _, file := range files {
		if err := w.AddFile(file); err != nil {
			return err
		}
	}
	return nil
}

// AddFile adds a single file to the watcher.
// It watches the parent directory to catch file modifications.
func (w *Watcher) AddFile(filePath string) error {
	absPath, err := resolvePath(filePath)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Already watching this file
	if w.files[absPath] {
		return nil
	}

	// Watch the parent directory (fsnotify works better with directories)
	dir := filepath.Dir(absPath)
	if err := w.fsWatcher.Add(dir); err != nil {
		return err
	}

	w.files[absPath] = true
	return nil
}

// RemoveFile stops watching a file.
func (w *Watcher) RemoveFile(filePath string) error {
	absPath, err := resolvePath(filePath)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.files, absPath)
	return nil
}

// Events returns the channel for receiving file change events.
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Errors returns the channel for receiving watcher errors.
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

// Start begins watching for file changes.
// This should be called in a goroutine.
func (w *Watcher) Start() {
	// Track last event time per file for debouncing
	lastEvent := make(map[string]time.Time)
	var lastEventMu sync.Mutex

	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Skip empty events (closed watcher)
			if event.Name == "" {
				continue
			}

			absPath, err := resolvePath(event.Name)
			if err != nil {
				continue
			}

			// Check if this is a watched file
			w.mu.RLock()
			isWatched := w.files[absPath]
			watchingNewDirs := w.watchNewDirs
			w.mu.RUnlock()

			// Handle new prd.json file detection
			if !isWatched && watchingNewDirs {
				if w.handleNewFileOrDirectory(event, absPath) {
					continue
				}
				// If it's not a new prd.json or directory, skip
				continue
			}

			if !isWatched {
				continue
			}

			// Convert fsnotify operation to our Operation type
			// Note: We check operation type BEFORE debouncing so that irrelevant
			// operations (like CHMOD) don't affect the debounce timing
			var op Operation
			switch {
			case event.Op&fsnotify.Write == fsnotify.Write:
				op = OpModify
			case event.Op&fsnotify.Create == fsnotify.Create:
				op = OpCreate
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				op = OpDelete
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				// Rename is treated as delete (file may have been renamed away)
				op = OpDelete
			default:
				// Skip events we don't care about (CHMOD, etc.)
				continue
			}

			// Debounce rapid events (only for events we care about)
			lastEventMu.Lock()
			last, exists := lastEvent[absPath]
			now := time.Now()
			if exists && now.Sub(last) < w.debounce {
				lastEventMu.Unlock()
				continue
			}
			lastEvent[absPath] = now
			lastEventMu.Unlock()

			// For delete operations, remove the file from tracking
			if op == OpDelete {
				w.mu.Lock()
				delete(w.files, absPath)
				w.mu.Unlock()
			}

			// Send the event
			select {
			case w.events <- Event{FilePath: absPath, Op: op}:
			default:
				// Channel full, skip event
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			select {
			case w.errors <- err:
			default:
				// Error channel full, skip
			}
		}
	}
}

// handleNewFileOrDirectory checks if the event is for a new prd.json file or a new directory.
// Returns true if the event was handled.
func (w *Watcher) handleNewFileOrDirectory(event fsnotify.Event, absPath string) bool {
	// Only handle create events for new file/directory detection
	if event.Op&fsnotify.Create != fsnotify.Create {
		return false
	}

	// Check if it's a directory or file
	info, err := os.Stat(absPath)
	if err != nil {
		return false
	}

	if info.IsDir() {
		// New directory created - watch it for future prd.json files
		w.watchNewDirectory(absPath)
		return true
	}

	// Check if it's a prd.json file
	if filepath.Base(absPath) == prdFileName {
		// Add to watched files and emit create event
		w.mu.Lock()
		w.files[absPath] = true
		w.mu.Unlock()

		select {
		case w.events <- Event{FilePath: absPath, Op: OpCreate}:
		default:
			// Channel full, skip event
		}
		return true
	}

	return false
}

// watchNewDirectory watches a newly created directory and all its subdirectories.
// Respects the directory skip list and watch limits.
func (w *Watcher) watchNewDirectory(dirPath string) {
	// Walk the new directory tree to watch it and find any prd.json files
	filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		absPath, err := resolvePath(path)
		if err != nil {
			return nil
		}

		if d.IsDir() {
			// Skip common large directories
			if scanner.ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}

			// Check if we've hit the watch limit
			w.mu.Lock()
			if w.watchLimitHit || w.watchCount >= MaxWatches {
				w.watchLimitHit = true
				w.mu.Unlock()
				return filepath.SkipDir
			}

			// Watch the directory if not already watching
			if !w.directories[absPath] {
				if err := w.fsWatcher.Add(absPath); err != nil {
					w.watchLimitHit = true
					w.mu.Unlock()
					return filepath.SkipDir
				}
				w.directories[absPath] = true
				w.watchCount++
			}
			w.mu.Unlock()
		} else if d.Name() == prdFileName {
			// Found a prd.json file - add to watched files and emit event
			w.mu.Lock()
			if !w.files[absPath] {
				w.files[absPath] = true
				w.mu.Unlock()

				select {
				case w.events <- Event{FilePath: absPath, Op: OpCreate}:
				default:
				}
			} else {
				w.mu.Unlock()
			}
		}

		return nil
	})
}

// Stop stops the watcher and releases resources.
// It closes the done channel to signal the Start goroutine to exit,
// then closes the events and errors channels to unblock any listeners,
// and finally closes the underlying fsnotify watcher.
func (w *Watcher) Stop() error {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return nil
	}
	w.stopped = true
	w.mu.Unlock()

	// Signal the Start goroutine to exit
	close(w.done)

	// Close the fsnotify watcher first to stop receiving events
	err := w.fsWatcher.Close()

	// Close our channels to unblock any listeners
	// This must be done after closing fsWatcher to prevent sending on closed channels
	close(w.events)
	close(w.errors)

	return err
}

// Stopped returns true if the watcher has been stopped.
func (w *Watcher) Stopped() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stopped
}

// WatchedFiles returns a slice of all currently watched file paths.
func (w *Watcher) WatchedFiles() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	files := make([]string, 0, len(w.files))
	for f := range w.files {
		files = append(files, f)
	}
	return files
}

// WatchCount returns the number of directories currently being watched.
func (w *Watcher) WatchCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.watchCount
}

// WatchLimitHit returns true if the watch limit was reached during setup.
func (w *Watcher) WatchLimitHit() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.watchLimitHit
}
