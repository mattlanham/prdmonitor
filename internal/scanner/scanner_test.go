package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_FindsPrdJsonFiles(t *testing.T) {
	// Create a temp directory structure with prd.json files
	root := t.TempDir()

	// Create subdirectories with prd.json files
	projectA := filepath.Join(root, "projectA")
	projectB := filepath.Join(root, "projectB")
	nestedProject := filepath.Join(root, "nested", "deep", "projectC")

	for _, dir := range []string{projectA, projectB, nestedProject} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
		prdFile := filepath.Join(dir, "prd.json")
		if err := os.WriteFile(prdFile, []byte(`{"name": "test"}`), 0644); err != nil {
			t.Fatalf("failed to create prd.json in %s: %v", dir, err)
		}
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.Count() != 3 {
		t.Errorf("expected 3 prd.json files, got %d", result.Count())
	}
}

func TestScan_IgnoresOtherFiles(t *testing.T) {
	root := t.TempDir()

	// Create various files that should be ignored
	files := []string{
		"prd.json",           // This one should be found
		"PRD.json",           // Wrong case, should be ignored
		"prd.txt",            // Wrong extension
		"other.json",         // Wrong name
		"subdir/prd.json",    // This one should also be found
		"subdir/config.json", // Should be ignored
	}

	for _, f := range files {
		path := filepath.Join(root, f)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// Should find exactly 2 prd.json files (root and subdir)
	if result.Count() != 2 {
		t.Errorf("expected 2 prd.json files, got %d", result.Count())
		for _, f := range result.Files {
			t.Logf("found: %s", f)
		}
	}
}

func TestScan_EmptyDirectory(t *testing.T) {
	root := t.TempDir()

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.Count() != 0 {
		t.Errorf("expected 0 files in empty directory, got %d", result.Count())
	}

	if result.HasErrors() {
		t.Errorf("expected no errors for empty directory")
	}
}

func TestScan_NonExistentDirectory(t *testing.T) {
	_, err := Scan("/nonexistent/path/that/does/not/exist")

	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func TestScan_FileInsteadOfDirectory(t *testing.T) {
	// Create a temp file (not a directory)
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = Scan(tmpFile.Name())

	if err == nil {
		t.Error("expected error when scanning a file, got nil")
	}

	// Check it's the right error type
	if _, ok := err.(*NotADirectoryError); !ok {
		t.Errorf("expected NotADirectoryError, got %T: %v", err, err)
	}
}

func TestScan_ReturnsAbsolutePaths(t *testing.T) {
	root := t.TempDir()

	// Create a prd.json file
	prdFile := filepath.Join(root, "prd.json")
	if err := os.WriteFile(prdFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.Count() != 1 {
		t.Fatalf("expected 1 file, got %d", result.Count())
	}

	// Check the path is absolute
	if !filepath.IsAbs(result.Files[0]) {
		t.Errorf("expected absolute path, got: %s", result.Files[0])
	}
}

func TestScan_HandlesUnreadableDirectories(t *testing.T) {
	// Skip this test on Windows where permission handling differs
	if os.Getenv("GOOS") == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	root := t.TempDir()

	// Create a readable directory with a prd.json
	readableDir := filepath.Join(root, "readable")
	if err := os.MkdirAll(readableDir, 0755); err != nil {
		t.Fatalf("failed to create readable directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(readableDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	// Create an unreadable directory
	unreadableDir := filepath.Join(root, "unreadable")
	if err := os.MkdirAll(unreadableDir, 0000); err != nil {
		t.Fatalf("failed to create unreadable directory: %v", err)
	}
	// Ensure cleanup can access it
	defer os.Chmod(unreadableDir, 0755)

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// Should still find the prd.json in the readable directory
	if result.Count() != 1 {
		t.Errorf("expected 1 prd.json file, got %d", result.Count())
	}

	// Should have recorded an error for the unreadable directory
	if !result.HasErrors() {
		t.Errorf("expected errors for unreadable directory")
	}
}

func TestScanResult_Count(t *testing.T) {
	result := &ScanResult{
		Files: []string{"/a/prd.json", "/b/prd.json", "/c/prd.json"},
	}

	if result.Count() != 3 {
		t.Errorf("expected Count() to return 3, got %d", result.Count())
	}
}

func TestScanResult_HasErrors(t *testing.T) {
	resultNoErrors := &ScanResult{
		Files:  []string{"/a/prd.json"},
		Errors: []error{},
	}

	if resultNoErrors.HasErrors() {
		t.Error("expected HasErrors() to return false for empty errors")
	}

	resultWithErrors := &ScanResult{
		Files:  []string{"/a/prd.json"},
		Errors: []error{os.ErrPermission},
	}

	if !resultWithErrors.HasErrors() {
		t.Error("expected HasErrors() to return true when errors exist")
	}
}

func TestNotADirectoryError(t *testing.T) {
	err := &NotADirectoryError{Path: "/some/path"}
	expected := "path is not a directory: /some/path"

	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}
