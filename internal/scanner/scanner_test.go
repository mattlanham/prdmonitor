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

func TestShouldSkipDir_SkipsNodeModules(t *testing.T) {
	if !ShouldSkipDir("node_modules") {
		t.Error("expected node_modules to be skipped")
	}
}

func TestShouldSkipDir_SkipsGit(t *testing.T) {
	if !ShouldSkipDir(".git") {
		t.Error("expected .git to be skipped")
	}
}

func TestShouldSkipDir_SkipsVendor(t *testing.T) {
	if !ShouldSkipDir("vendor") {
		t.Error("expected vendor to be skipped")
	}
}

func TestShouldSkipDir_DoesNotSkipRegularDirs(t *testing.T) {
	regularDirs := []string{"src", "lib", "app", "projects", "myproject", "ralph"}
	for _, dir := range regularDirs {
		if ShouldSkipDir(dir) {
			t.Errorf("expected %s to NOT be skipped", dir)
		}
	}
}

func TestShouldSkipDir_AllSkipDirsInMap(t *testing.T) {
	// Verify all skip dirs in the map return true
	for dir := range SkipDirs {
		if !ShouldSkipDir(dir) {
			t.Errorf("expected %s to be skipped (it's in SkipDirs map)", dir)
		}
	}
}

func TestScan_SkipsNodeModules(t *testing.T) {
	root := t.TempDir()

	// Create a regular project with prd.json
	projectDir := filepath.Join(root, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	// Create node_modules with a prd.json that should be ignored
	nodeModulesDir := filepath.Join(root, "myproject", "node_modules", "some-package")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatalf("failed to create node_modules directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nodeModulesDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json in node_modules: %v", err)
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// Should only find the one prd.json in myproject, not the one in node_modules
	if result.Count() != 1 {
		t.Errorf("expected 1 prd.json file (node_modules should be skipped), got %d", result.Count())
		for _, f := range result.Files {
			t.Logf("found: %s", f)
		}
	}
}

func TestScan_SkipsGitDirectory(t *testing.T) {
	root := t.TempDir()

	// Create a regular project with prd.json
	projectDir := filepath.Join(root, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	// Create .git directory with a prd.json that should be ignored
	gitDir := filepath.Join(root, "myproject", ".git", "objects")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json in .git: %v", err)
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// Should only find the one prd.json in myproject, not the one in .git
	if result.Count() != 1 {
		t.Errorf("expected 1 prd.json file (.git should be skipped), got %d", result.Count())
		for _, f := range result.Files {
			t.Logf("found: %s", f)
		}
	}
}

func TestScan_SkipsMultipleLargeDirectories(t *testing.T) {
	root := t.TempDir()

	// Create a regular project with prd.json
	projectDir := filepath.Join(root, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	// Create multiple directories that should be skipped
	skipDirs := []string{"node_modules", ".git", "vendor", ".venv", "dist", "build"}
	for _, skipDir := range skipDirs {
		dirPath := filepath.Join(root, "myproject", skipDir, "subdir")
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create %s directory: %v", skipDir, err)
		}
		// Each has a prd.json that should be ignored
		if err := os.WriteFile(filepath.Join(dirPath, "prd.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create prd.json in %s: %v", skipDir, err)
		}
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// Should only find the one prd.json in myproject
	if result.Count() != 1 {
		t.Errorf("expected 1 prd.json file (skip directories should be skipped), got %d", result.Count())
		for _, f := range result.Files {
			t.Logf("found: %s", f)
		}
	}
}
