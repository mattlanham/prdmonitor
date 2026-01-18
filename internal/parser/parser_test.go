package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lanham/prdmonitor/internal/model"
)

func TestParseBytes_ValidJSON(t *testing.T) {
	validJSON := []byte(`{
		"name": "Test Project",
		"branchName": "feature/test",
		"userStories": [
			{
				"id": "TS-001",
				"title": "Test Story",
				"description": "A test user story",
				"acceptanceCriteria": ["Criteria 1", "Criteria 2"],
				"priority": 1,
				"status": "incomplete"
			}
		]
	}`)

	prd, err := ParseBytes(validJSON)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if prd.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got '%s'", prd.Name)
	}
	if prd.BranchName != "feature/test" {
		t.Errorf("expected branchName 'feature/test', got '%s'", prd.BranchName)
	}
	if len(prd.UserStories) != 1 {
		t.Fatalf("expected 1 user story, got %d", len(prd.UserStories))
	}

	story := prd.UserStories[0]
	if story.ID != "TS-001" {
		t.Errorf("expected story ID 'TS-001', got '%s'", story.ID)
	}
	if story.Title != "Test Story" {
		t.Errorf("expected story title 'Test Story', got '%s'", story.Title)
	}
	if story.Status != "incomplete" {
		t.Errorf("expected status 'incomplete', got '%s'", story.Status)
	}
}

func TestParseBytes_ExtractsProjectName(t *testing.T) {
	json := []byte(`{"name": "My Project", "userStories": []}`)

	prd, err := ParseBytes(json)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if prd.Name != "My Project" {
		t.Errorf("expected name 'My Project', got '%s'", prd.Name)
	}
}

func TestParseBytes_ExtractsUserStoryFields(t *testing.T) {
	json := []byte(`{
		"name": "Project",
		"userStories": [
			{
				"id": "US-100",
				"title": "User Story Title",
				"description": "Detailed description here",
				"acceptanceCriteria": ["AC1", "AC2", "AC3"],
				"priority": 5,
				"status": "in-progress"
			}
		]
	}`)

	prd, err := ParseBytes(json)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if len(prd.UserStories) != 1 {
		t.Fatalf("expected 1 user story, got %d", len(prd.UserStories))
	}

	story := prd.UserStories[0]
	if story.ID != "US-100" {
		t.Errorf("expected ID 'US-100', got '%s'", story.ID)
	}
	if story.Title != "User Story Title" {
		t.Errorf("expected title 'User Story Title', got '%s'", story.Title)
	}
	if story.Description != "Detailed description here" {
		t.Errorf("expected description 'Detailed description here', got '%s'", story.Description)
	}
	if len(story.AcceptanceCriteria) != 3 {
		t.Errorf("expected 3 acceptance criteria, got %d", len(story.AcceptanceCriteria))
	}
	if story.Priority != 5 {
		t.Errorf("expected priority 5, got %d", story.Priority)
	}
	if story.Status != "in-progress" {
		t.Errorf("expected status 'in-progress', got '%s'", story.Status)
	}
}

func TestParseBytes_MultipleUserStories(t *testing.T) {
	json := []byte(`{
		"name": "Multi Story Project",
		"userStories": [
			{"id": "MS-001", "title": "First", "status": "complete"},
			{"id": "MS-002", "title": "Second", "status": "in-progress"},
			{"id": "MS-003", "title": "Third", "status": "incomplete"}
		]
	}`)

	prd, err := ParseBytes(json)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if len(prd.UserStories) != 3 {
		t.Fatalf("expected 3 user stories, got %d", len(prd.UserStories))
	}

	expectedStatuses := []string{"complete", "in-progress", "incomplete"}
	for i, story := range prd.UserStories {
		if story.Status != expectedStatuses[i] {
			t.Errorf("story %d: expected status '%s', got '%s'", i, expectedStatuses[i], story.Status)
		}
	}
}

func TestParseBytes_EmptyUserStories(t *testing.T) {
	json := []byte(`{"name": "Empty Project", "userStories": []}`)

	prd, err := ParseBytes(json)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if len(prd.UserStories) != 0 {
		t.Errorf("expected 0 user stories, got %d", len(prd.UserStories))
	}
}

func TestParseBytes_MalformedJSON(t *testing.T) {
	testCases := []struct {
		name string
		json []byte
	}{
		{"invalid syntax", []byte(`{invalid json}`)},
		{"unclosed brace", []byte(`{"name": "test"`)},
		{"trailing comma", []byte(`{"name": "test",}`)},
		{"single quotes", []byte(`{'name': 'test'}`)},
		{"empty input", []byte(``)},
		{"just whitespace", []byte(`   `)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBytes(tc.json)
			if err == nil {
				t.Errorf("expected error for malformed JSON, got nil")
			}
		})
	}
}

func TestParseBytes_MissingFields(t *testing.T) {
	// JSON with missing optional fields should still parse
	json := []byte(`{"name": "Minimal"}`)

	prd, err := ParseBytes(json)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if prd.Name != "Minimal" {
		t.Errorf("expected name 'Minimal', got '%s'", prd.Name)
	}
	if prd.UserStories != nil && len(prd.UserStories) != 0 {
		t.Errorf("expected nil or empty userStories, got %v", prd.UserStories)
	}
}

func TestParseFile_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "prd.json")

	content := []byte(`{
		"name": "File Test Project",
		"branchName": "main",
		"userStories": [
			{"id": "FT-001", "title": "File Test Story", "status": "complete"}
		]
	}`)

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if result.FilePath != filePath {
		t.Errorf("expected filePath '%s', got '%s'", filePath, result.FilePath)
	}
	if result.PRD.Name != "File Test Project" {
		t.Errorf("expected name 'File Test Project', got '%s'", result.PRD.Name)
	}
}

func TestParseFile_NonExistentFile(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/prd.json")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected ParseError, got %T", err)
	}
}

func TestParseFile_MalformedFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "malformed.json")

	if err := os.WriteFile(filePath, []byte(`{invalid}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := ParseFile(filePath)
	if err == nil {
		t.Error("expected error for malformed file, got nil")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected ParseError, got %T", err)
	}
	if parseErr.FilePath != filePath {
		t.Errorf("expected filePath '%s', got '%s'", filePath, parseErr.FilePath)
	}
}

func TestParseError_Error(t *testing.T) {
	parseErr := &ParseError{
		FilePath: "/path/to/file.json",
		Err:      errors.New("some error"),
	}

	expected := "failed to parse /path/to/file.json: some error"
	if parseErr.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, parseErr.Error())
	}
}

func TestParseError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	parseErr := &ParseError{
		FilePath: "/path/to/file.json",
		Err:      innerErr,
	}

	if parseErr.Unwrap() != innerErr {
		t.Error("Unwrap did not return the inner error")
	}
}

func TestParseFiles_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create two valid prd.json files
	file1 := filepath.Join(tempDir, "project1", "prd.json")
	file2 := filepath.Join(tempDir, "project2", "prd.json")

	os.MkdirAll(filepath.Dir(file1), 0755)
	os.MkdirAll(filepath.Dir(file2), 0755)

	content1 := []byte(`{"name": "Project 1", "userStories": []}`)
	content2 := []byte(`{"name": "Project 2", "userStories": []}`)

	os.WriteFile(file1, content1, 0644)
	os.WriteFile(file2, content2, 0644)

	results, errors := ParseFiles([]string{file1, file2})

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].PRD.Name != "Project 1" {
		t.Errorf("expected 'Project 1', got '%s'", results[0].PRD.Name)
	}
	if results[1].PRD.Name != "Project 2" {
		t.Errorf("expected 'Project 2', got '%s'", results[1].PRD.Name)
	}
}

func TestParseFiles_WithErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Create one valid and one malformed file
	validFile := filepath.Join(tempDir, "valid.json")
	malformedFile := filepath.Join(tempDir, "malformed.json")

	os.WriteFile(validFile, []byte(`{"name": "Valid Project"}`), 0644)
	os.WriteFile(malformedFile, []byte(`{invalid}`), 0644)

	results, errs := ParseFiles([]string{validFile, malformedFile})

	if len(results) != 1 {
		t.Errorf("expected 1 valid result, got %d", len(results))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
	if results[0].PRD.Name != "Valid Project" {
		t.Errorf("expected 'Valid Project', got '%s'", results[0].PRD.Name)
	}
}

func TestParseFiles_NonExistentFiles(t *testing.T) {
	results, errs := ParseFiles([]string{"/nonexistent1.json", "/nonexistent2.json"})

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}
}

func TestParseFiles_EmptyList(t *testing.T) {
	results, errs := ParseFiles([]string{})

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseBytes_AllStatusValues(t *testing.T) {
	testCases := []struct {
		status   string
		expected string
	}{
		{model.StatusIncomplete, "incomplete"},
		{model.StatusInProgress, "in-progress"},
		{model.StatusComplete, "complete"},
		{"unknown", "unknown"}, // Unknown status is preserved as-is
	}

	for _, tc := range testCases {
		t.Run(tc.status, func(t *testing.T) {
			json := []byte(`{"name": "Test", "userStories": [{"id": "1", "status": "` + tc.status + `"}]}`)
			prd, err := ParseBytes(json)
			if err != nil {
				t.Fatalf("ParseBytes failed: %v", err)
			}
			if prd.UserStories[0].Status != tc.expected {
				t.Errorf("expected status '%s', got '%s'", tc.expected, prd.UserStories[0].Status)
			}
		})
	}
}
