// Package watcher provides file system watching for prd.json files using fsnotify.
package watcher

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

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
	fsWatcher *fsnotify.Watcher
	events    chan Event
	errors    chan error
	done      chan struct{}
	files     map[string]bool // Set of watched files
	mu        sync.RWMutex
	debounce  time.Duration // Debounce duration for rapid events
}

// New creates a new Watcher.
func New() (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fsWatcher: fsWatcher,
		events:    make(chan Event, 100),
		errors:    make(chan error, 10),
		done:      make(chan struct{}),
		files:     make(map[string]bool),
		debounce:  100 * time.Millisecond,
	}, nil
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
	absPath, err := filepath.Abs(filePath)
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
	absPath, err := filepath.Abs(filePath)
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

			absPath, err := filepath.Abs(event.Name)
			if err != nil {
				continue
			}

			// Only process events for files we're watching
			w.mu.RLock()
			isWatched := w.files[absPath]
			w.mu.RUnlock()

			if !isWatched {
				continue
			}

			// Debounce rapid events
			lastEventMu.Lock()
			last, exists := lastEvent[absPath]
			now := time.Now()
			if exists && now.Sub(last) < w.debounce {
				lastEventMu.Unlock()
				continue
			}
			lastEvent[absPath] = now
			lastEventMu.Unlock()

			// Convert fsnotify operation to our Operation type
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
				continue
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

// Stop stops the watcher and releases resources.
func (w *Watcher) Stop() error {
	close(w.done)
	return w.fsWatcher.Close()
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
