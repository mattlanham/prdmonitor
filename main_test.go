package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseArgs_WithValidFolder(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	args := []string{"prdmonitor", tmpDir}
	config, err := ParseArgs(args)

	if err != nil {
		t.Errorf("ParseArgs() returned unexpected error: %v", err)
	}

	if config == nil {
		t.Fatal("ParseArgs() returned nil config")
	}

	if config.RootDir != tmpDir {
		t.Errorf("ParseArgs() RootDir = %q, want %q", config.RootDir, tmpDir)
	}
}

func TestParseArgs_DefaultToCurrentDirectory(t *testing.T) {
	// Get the current working directory to compare
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Only pass the program name (no folder argument)
	args := []string{"prdmonitor"}
	config, err := ParseArgs(args)

	if err != nil {
		t.Errorf("ParseArgs() returned unexpected error: %v", err)
	}

	if config == nil {
		t.Fatal("ParseArgs() returned nil config")
	}

	if config.RootDir != cwd {
		t.Errorf("ParseArgs() RootDir = %q, want %q (current directory)", config.RootDir, cwd)
	}
}

func TestParseArgs_NonExistentFolder(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist")

	args := []string{"prdmonitor", nonExistentPath}
	config, err := ParseArgs(args)

	if err == nil {
		t.Error("ParseArgs() expected error for non-existent folder, got nil")
	}

	if config != nil {
		t.Errorf("ParseArgs() expected nil config for non-existent folder, got %+v", config)
	}
}

func TestParseArgs_FileNotDirectory(t *testing.T) {
	// Create a temporary file (not a directory)
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := []string{"prdmonitor", tmpFile}
	config, err := ParseArgs(args)

	if err == nil {
		t.Error("ParseArgs() expected error for file (not directory), got nil")
	}

	if config != nil {
		t.Errorf("ParseArgs() expected nil config for file, got %+v", config)
	}
}

func TestValidateFolder_ValidDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	err := ValidateFolder(tmpDir)
	if err != nil {
		t.Errorf("ValidateFolder() returned unexpected error for valid directory: %v", err)
	}
}

func TestValidateFolder_NonExistent(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist")

	err := ValidateFolder(nonExistentPath)
	if err == nil {
		t.Error("ValidateFolder() expected error for non-existent path, got nil")
	}
}

func TestValidateFolder_NotADirectory(t *testing.T) {
	// Create a temporary file
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := ValidateFolder(tmpFile)
	if err == nil {
		t.Error("ValidateFolder() expected error for file path, got nil")
	}
}
