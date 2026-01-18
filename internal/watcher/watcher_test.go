package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolvePath(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "prd.json")

	// Create the file
	if err := os.WriteFile(tmpFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Get abs path and resolved path
	absPath, _ := filepath.Abs(tmpFile)
	resolvedPath, _ := filepath.EvalSymlinks(absPath)

	// On macOS, /var is a symlink to /private/var
	// The resolved path should have /private prefix
	t.Logf("absPath: %s", absPath)
	t.Logf("resolvedPath: %s", resolvedPath)

	// Test that watcher stores resolved path
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	if err := w.AddFile(tmpFile); err != nil {
		t.Fatalf("AddFile() returned error: %v", err)
	}

	watched := w.WatchedFiles()
	if len(watched) != 1 {
		t.Fatalf("Expected 1 watched file, got %d", len(watched))
	}

	t.Logf("Watched path: %s", watched[0])

	// The watched file should be the resolved path
	if watched[0] != resolvedPath {
		t.Errorf("Expected watched path %q, got %q", resolvedPath, watched[0])
	}
}

func TestNew(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	if w.fsWatcher == nil {
		t.Error("New() did not create fsnotify watcher")
	}
	if w.events == nil {
		t.Error("New() did not create events channel")
	}
	if w.errors == nil {
		t.Error("New() did not create errors channel")
	}
	if w.files == nil {
		t.Error("New() did not create files map")
	}
}

func TestAddFile(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	// Create a temp file to watch
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "prd.json")
	if err := os.WriteFile(tmpFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Add the file
	if err := w.AddFile(tmpFile); err != nil {
		t.Errorf("AddFile() returned error: %v", err)
	}

	// Verify file is tracked (use resolved path for comparison)
	absPath, _ := filepath.Abs(tmpFile)
	resolvedPath, _ := filepath.EvalSymlinks(absPath)
	if !w.files[resolvedPath] {
		t.Error("AddFile() did not track the file")
	}
}

func TestAddFile_Duplicate(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "prd.json")
	if err := os.WriteFile(tmpFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Add the same file twice - should not error
	if err := w.AddFile(tmpFile); err != nil {
		t.Errorf("First AddFile() returned error: %v", err)
	}
	if err := w.AddFile(tmpFile); err != nil {
		t.Errorf("Second AddFile() returned error: %v", err)
	}

	// Should only have one entry
	if len(w.files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(w.files))
	}
}

func TestAddFiles(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	files := []string{
		filepath.Join(tmpDir, "prd1.json"),
		filepath.Join(tmpDir, "prd2.json"),
	}

	for _, f := range files {
		if err := os.WriteFile(f, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
	}

	if err := w.AddFiles(files); err != nil {
		t.Errorf("AddFiles() returned error: %v", err)
	}

	if len(w.files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(w.files))
	}
}

func TestRemoveFile(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "prd.json")
	if err := os.WriteFile(tmpFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	w.AddFile(tmpFile)

	if err := w.RemoveFile(tmpFile); err != nil {
		t.Errorf("RemoveFile() returned error: %v", err)
	}

	absPath, _ := filepath.Abs(tmpFile)
	resolvedPath, _ := filepath.EvalSymlinks(absPath)
	if w.files[resolvedPath] {
		t.Error("RemoveFile() did not remove the file from tracking")
	}
}

func TestWatchedFiles(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	files := []string{
		filepath.Join(tmpDir, "prd1.json"),
		filepath.Join(tmpDir, "prd2.json"),
	}

	for _, f := range files {
		if err := os.WriteFile(f, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
	}

	w.AddFiles(files)

	watched := w.WatchedFiles()
	if len(watched) != 2 {
		t.Errorf("Expected 2 watched files, got %d", len(watched))
	}
}

func TestOperation_String(t *testing.T) {
	tests := []struct {
		op       Operation
		expected string
	}{
		{OpModify, "modify"},
		{OpCreate, "create"},
		{OpDelete, "delete"},
		{Operation(99), "unknown"},
	}

	for _, tt := range tests {
		result := tt.op.String()
		if result != tt.expected {
			t.Errorf("Operation(%d).String() = %q, want %q", tt.op, result, tt.expected)
		}
	}
}

func TestWatcher_DetectsFileModification(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "prd.json")
	if err := os.WriteFile(tmpFile, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Log the paths
	absPath, _ := filepath.Abs(tmpFile)
	resolvedPath, _ := filepath.EvalSymlinks(absPath)
	t.Logf("tmpFile: %s", tmpFile)
	t.Logf("absPath: %s", absPath)
	t.Logf("resolvedPath: %s", resolvedPath)

	// Add file to watcher
	if err := w.AddFile(tmpFile); err != nil {
		t.Fatalf("AddFile() returned error: %v", err)
	}

	// Log watched files
	t.Logf("Watched files: %v", w.WatchedFiles())

	// Start watcher in goroutine
	go w.Start()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(tmpFile, []byte(`{"name": "updated"}`), 0644); err != nil {
		t.Fatalf("Failed to modify temp file: %v", err)
	}

	// Wait for event with timeout, checking both events and errors
	timeout := time.After(2 * time.Second)
	for {
		select {
		case event := <-w.Events():
			t.Logf("Received event: %+v", event)
			if event.FilePath != resolvedPath {
				t.Errorf("Expected FilePath %q, got %q", resolvedPath, event.FilePath)
			}
			if event.Op != OpModify {
				t.Errorf("Expected OpModify, got %v", event.Op)
			}
			return
		case err := <-w.Errors():
			t.Logf("Received error: %v", err)
		case <-timeout:
			t.Errorf("Timed out waiting for file change event. Watched files: %v", w.WatchedFiles())
			return
		}
	}
}

func TestWatcher_IgnoresUnwatchedFiles(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()

	// Create and watch one file
	watchedFile := filepath.Join(tmpDir, "watched.json")
	if err := os.WriteFile(watchedFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	w.AddFile(watchedFile)

	// Create an unwatched file in the same directory
	unwatchedFile := filepath.Join(tmpDir, "unwatched.json")
	if err := os.WriteFile(unwatchedFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Start watcher
	go w.Start()
	time.Sleep(100 * time.Millisecond)

	// Modify the unwatched file
	if err := os.WriteFile(unwatchedFile, []byte(`{"updated": true}`), 0644); err != nil {
		t.Fatalf("Failed to modify temp file: %v", err)
	}

	// Should not receive any event
	select {
	case event := <-w.Events():
		t.Errorf("Received unexpected event for unwatched file: %+v", event)
	case <-time.After(500 * time.Millisecond):
		// Expected - no event for unwatched file
	}
}

func TestWatcher_Stop(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Start watcher
	go w.Start()
	time.Sleep(50 * time.Millisecond)

	// Stop should not error
	if err := w.Stop(); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

func TestWatcher_Events(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	events := w.Events()
	if events == nil {
		t.Error("Events() returned nil channel")
	}
}

func TestWatcher_Errors(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	errors := w.Errors()
	if errors == nil {
		t.Error("Errors() returned nil channel")
	}
}

func TestWatchDirectory(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()

	// Create subdirectories
	subDir := filepath.Join(tmpDir, "project1")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Watch the directory tree
	if err := w.WatchDirectory(tmpDir); err != nil {
		t.Errorf("WatchDirectory() returned error: %v", err)
	}

	// Verify directories are being tracked
	w.mu.RLock()
	watchingNewDirs := w.watchNewDirs
	dirCount := len(w.directories)
	w.mu.RUnlock()

	if !watchingNewDirs {
		t.Error("WatchDirectory() did not enable new directory watching")
	}

	// Should watch at least 2 directories (root and subdirectory)
	if dirCount < 2 {
		t.Errorf("Expected at least 2 directories to be watched, got %d", dirCount)
	}
}

func TestWatcher_DetectsNewPrdJsonFile(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()

	// Watch the directory tree (before the file exists)
	if err := w.WatchDirectory(tmpDir); err != nil {
		t.Fatalf("WatchDirectory() returned error: %v", err)
	}

	// Start watcher
	go w.Start()
	time.Sleep(100 * time.Millisecond)

	// Create a new prd.json file
	newFile := filepath.Join(tmpDir, "prd.json")
	if err := os.WriteFile(newFile, []byte(`{"name": "new project"}`), 0644); err != nil {
		t.Fatalf("Failed to create prd.json: %v", err)
	}

	// Wait for event
	select {
	case event := <-w.Events():
		absPath, _ := filepath.Abs(newFile)
		resolvedPath, _ := filepath.EvalSymlinks(absPath)
		if event.FilePath != resolvedPath {
			t.Errorf("Expected FilePath %q, got %q", resolvedPath, event.FilePath)
		}
		if event.Op != OpCreate {
			t.Errorf("Expected OpCreate, got %v", event.Op)
		}
	case <-time.After(3 * time.Second):
		t.Error("Timed out waiting for new file event")
	}
}

func TestWatcher_DetectsNewSubdirectoryWithPrdJson(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()

	// Watch the directory tree
	if err := w.WatchDirectory(tmpDir); err != nil {
		t.Fatalf("WatchDirectory() returned error: %v", err)
	}

	// Start watcher
	go w.Start()
	time.Sleep(100 * time.Millisecond)

	// Create a new subdirectory with a prd.json file
	newSubDir := filepath.Join(tmpDir, "newproject")
	if err := os.MkdirAll(newSubDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Give time for directory watch to be set up
	time.Sleep(200 * time.Millisecond)

	// Create prd.json in the new subdirectory
	newFile := filepath.Join(newSubDir, "prd.json")
	if err := os.WriteFile(newFile, []byte(`{"name": "new subproject"}`), 0644); err != nil {
		t.Fatalf("Failed to create prd.json: %v", err)
	}

	// Wait for event
	select {
	case event := <-w.Events():
		absPath, _ := filepath.Abs(newFile)
		resolvedPath, _ := filepath.EvalSymlinks(absPath)
		if event.FilePath != resolvedPath {
			t.Errorf("Expected FilePath %q, got %q", resolvedPath, event.FilePath)
		}
		if event.Op != OpCreate {
			t.Errorf("Expected OpCreate, got %v", event.Op)
		}
	case <-time.After(3 * time.Second):
		t.Error("Timed out waiting for new file in subdirectory event")
	}
}

func TestWatcher_IgnoresNonPrdJsonFiles(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()

	// Watch the directory tree
	if err := w.WatchDirectory(tmpDir); err != nil {
		t.Fatalf("WatchDirectory() returned error: %v", err)
	}

	// Start watcher
	go w.Start()
	time.Sleep(100 * time.Millisecond)

	// Create a non-prd.json file
	otherFile := filepath.Join(tmpDir, "other.json")
	if err := os.WriteFile(otherFile, []byte(`{"name": "other file"}`), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Should NOT receive an event for non-prd.json files
	select {
	case event := <-w.Events():
		t.Errorf("Received unexpected event for non-prd.json file: %+v", event)
	case <-time.After(500 * time.Millisecond):
		// Expected - no event for non-prd.json files
	}
}

func TestWatcher_DetectsNewPrdJsonInExistingSubdirectory(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()

	// Create an existing subdirectory (before watching)
	existingSubDir := filepath.Join(tmpDir, "existing")
	if err := os.MkdirAll(existingSubDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Watch the directory tree
	if err := w.WatchDirectory(tmpDir); err != nil {
		t.Fatalf("WatchDirectory() returned error: %v", err)
	}

	// Start watcher
	go w.Start()
	time.Sleep(100 * time.Millisecond)

	// Create prd.json in the existing subdirectory
	newFile := filepath.Join(existingSubDir, "prd.json")
	if err := os.WriteFile(newFile, []byte(`{"name": "existing subdir project"}`), 0644); err != nil {
		t.Fatalf("Failed to create prd.json: %v", err)
	}

	// Wait for event
	select {
	case event := <-w.Events():
		absPath, _ := filepath.Abs(newFile)
		resolvedPath, _ := filepath.EvalSymlinks(absPath)
		if event.FilePath != resolvedPath {
			t.Errorf("Expected FilePath %q, got %q", resolvedPath, event.FilePath)
		}
		if event.Op != OpCreate {
			t.Errorf("Expected OpCreate, got %v", event.Op)
		}
	case <-time.After(3 * time.Second):
		t.Error("Timed out waiting for new file event in existing subdirectory")
	}
}
