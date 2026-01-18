package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

	// Verify file is tracked
	absPath, _ := filepath.Abs(tmpFile)
	if !w.files[absPath] {
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
	if w.files[absPath] {
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

	// Add file to watcher
	if err := w.AddFile(tmpFile); err != nil {
		t.Fatalf("AddFile() returned error: %v", err)
	}

	// Start watcher in goroutine
	go w.Start()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(tmpFile, []byte(`{"name": "updated"}`), 0644); err != nil {
		t.Fatalf("Failed to modify temp file: %v", err)
	}

	// Wait for event with timeout
	select {
	case event := <-w.Events():
		absPath, _ := filepath.Abs(tmpFile)
		if event.FilePath != absPath {
			t.Errorf("Expected FilePath %q, got %q", absPath, event.FilePath)
		}
		if event.Op != OpModify {
			t.Errorf("Expected OpModify, got %v", event.Op)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for file change event")
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
