package scanner

import (
	"fmt"
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

// Tests for PM-018: Fast prd.json file discovery

func TestScanWithProgress_CallsCallback(t *testing.T) {
	root := t.TempDir()

	// Create several directories with prd.json files
	for i := 0; i < 5; i++ {
		dir := filepath.Join(root, fmt.Sprintf("project%d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "prd.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create prd.json: %v", err)
		}
	}

	// Track progress callbacks
	var progressCalls []ScanProgress
	callback := func(progress ScanProgress) {
		progressCalls = append(progressCalls, progress)
	}

	result, err := ScanWithProgress(root, callback)
	if err != nil {
		t.Fatalf("ScanWithProgress returned error: %v", err)
	}

	// Should find all 5 prd.json files
	if result.Count() != 5 {
		t.Errorf("expected 5 prd.json files, got %d", result.Count())
	}

	// Should have at least one progress call (the final one)
	if len(progressCalls) == 0 {
		t.Error("expected at least one progress callback")
	}

	// Final callback should have IsComplete = true
	lastCall := progressCalls[len(progressCalls)-1]
	if !lastCall.IsComplete {
		t.Error("expected final progress callback to have IsComplete = true")
	}

	// Final callback should report correct file count
	if lastCall.FilesFound != 5 {
		t.Errorf("expected final progress to report 5 files found, got %d", lastCall.FilesFound)
	}
}

func TestScanWithProgress_NilCallback(t *testing.T) {
	root := t.TempDir()

	// Create a prd.json file
	if err := os.WriteFile(filepath.Join(root, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	// Should work fine with nil callback
	result, err := ScanWithProgress(root, nil)
	if err != nil {
		t.Fatalf("ScanWithProgress with nil callback returned error: %v", err)
	}

	if result.Count() != 1 {
		t.Errorf("expected 1 prd.json file, got %d", result.Count())
	}
}

func TestScanProgress_Fields(t *testing.T) {
	progress := ScanProgress{
		DirsScanned: 42,
		FilesFound:  5,
		CurrentDir:  "/test/path",
		IsComplete:  false,
	}

	if progress.DirsScanned != 42 {
		t.Errorf("expected DirsScanned = 42, got %d", progress.DirsScanned)
	}
	if progress.FilesFound != 5 {
		t.Errorf("expected FilesFound = 5, got %d", progress.FilesFound)
	}
	if progress.CurrentDir != "/test/path" {
		t.Errorf("expected CurrentDir = /test/path, got %s", progress.CurrentDir)
	}
	if progress.IsComplete {
		t.Error("expected IsComplete = false")
	}
}

func TestParallelScan_FindsAllFiles(t *testing.T) {
	root := t.TempDir()

	// Create a directory tree with prd.json files at various depths
	paths := []string{
		"project1",
		"project2/subproject",
		"nested/deep/project3",
		"another/path/to/project4",
	}

	for _, p := range paths {
		dir := filepath.Join(root, p)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", p, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "prd.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create prd.json in %s: %v", p, err)
		}
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.Count() != 4 {
		t.Errorf("expected 4 prd.json files, got %d", result.Count())
		for _, f := range result.Files {
			t.Logf("found: %s", f)
		}
	}
}

func TestParallelScan_SkipsLargeDirectories(t *testing.T) {
	root := t.TempDir()

	// Create a valid project
	projectDir := filepath.Join(root, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	// Create directories that should be skipped with prd.json files
	skipDirs := []string{"node_modules", ".git", "vendor", "__pycache__", "dist"}
	for _, skipDir := range skipDirs {
		dir := filepath.Join(projectDir, skipDir, "deeply", "nested")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s directory: %v", skipDir, err)
		}
		// Add prd.json that should be ignored
		if err := os.WriteFile(filepath.Join(dir, "prd.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create prd.json in %s: %v", skipDir, err)
		}
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// Should only find the one prd.json in myproject
	if result.Count() != 1 {
		t.Errorf("expected 1 prd.json file (parallel scan should skip large dirs), got %d", result.Count())
		for _, f := range result.Files {
			t.Logf("found: %s", f)
		}
	}
}

func TestParallelScan_HandlesEmptyDirectory(t *testing.T) {
	root := t.TempDir()

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.Count() != 0 {
		t.Errorf("expected 0 files in empty directory, got %d", result.Count())
	}
}

func TestParallelScan_HandlesDeepNesting(t *testing.T) {
	root := t.TempDir()

	// Create a deeply nested directory structure
	deepPath := root
	for i := 0; i < 20; i++ {
		deepPath = filepath.Join(deepPath, fmt.Sprintf("level%d", i))
	}
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatalf("failed to create deep directory structure: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deepPath, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.Count() != 1 {
		t.Errorf("expected 1 prd.json file in deep nesting, got %d", result.Count())
	}
}

func TestParallelScan_HandlesManyDirectories(t *testing.T) {
	root := t.TempDir()

	// Create many directories to test parallelism
	numDirs := 100
	for i := 0; i < numDirs; i++ {
		dir := filepath.Join(root, fmt.Sprintf("project%d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		// Only some have prd.json
		if i%5 == 0 {
			if err := os.WriteFile(filepath.Join(dir, "prd.json"), []byte("{}"), 0644); err != nil {
				t.Fatalf("failed to create prd.json: %v", err)
			}
		}
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	expectedCount := numDirs / 5 // Every 5th directory has prd.json
	if result.Count() != expectedCount {
		t.Errorf("expected %d prd.json files, got %d", expectedCount, result.Count())
	}
}

func TestParallelScan_ProgressTracksDirsScanned(t *testing.T) {
	root := t.TempDir()

	// Create several directories
	for i := 0; i < 10; i++ {
		dir := filepath.Join(root, fmt.Sprintf("dir%d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	var finalProgress ScanProgress
	callback := func(progress ScanProgress) {
		if progress.IsComplete {
			finalProgress = progress
		}
	}

	_, err := ScanWithProgress(root, callback)
	if err != nil {
		t.Fatalf("ScanWithProgress returned error: %v", err)
	}

	// Should have scanned at least the root + 10 directories
	if finalProgress.DirsScanned < 11 {
		t.Errorf("expected at least 11 directories scanned, got %d", finalProgress.DirsScanned)
	}
}

func TestScan_Pycache_IsSkipped(t *testing.T) {
	root := t.TempDir()

	// Create a regular project with prd.json
	projectDir := filepath.Join(root, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json: %v", err)
	}

	// Create __pycache__ with a prd.json that should be ignored
	pycacheDir := filepath.Join(root, "myproject", "__pycache__", "subdir")
	if err := os.MkdirAll(pycacheDir, 0755); err != nil {
		t.Fatalf("failed to create __pycache__ directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pycacheDir, "prd.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create prd.json in __pycache__: %v", err)
	}

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.Count() != 1 {
		t.Errorf("expected 1 prd.json file (__pycache__ should be skipped), got %d", result.Count())
		for _, f := range result.Files {
			t.Logf("found: %s", f)
		}
	}
}

func TestShouldSkipDir_AllRequiredDirsInSkipList(t *testing.T) {
	// These are the directories mentioned in PM-018 acceptance criteria
	requiredSkipDirs := []string{"node_modules", ".git", "vendor", "__pycache__"}

	for _, dir := range requiredSkipDirs {
		if !ShouldSkipDir(dir) {
			t.Errorf("expected %s to be in skip list (required by PM-018 acceptance criteria)", dir)
		}
	}
}
