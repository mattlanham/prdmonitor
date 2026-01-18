package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/model"
	"lanham/prdmonitor/internal/parser"
	"lanham/prdmonitor/internal/watcher"
)

func TestNewColumn(t *testing.T) {
	col := NewColumn("Test Column", "39")

	if col.title != "Test Column" {
		t.Errorf("expected title 'Test Column', got %q", col.title)
	}

	if len(col.cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(col.cards))
	}
}

func TestColumn_AddCard(t *testing.T) {
	col := NewColumn("Test", "39")

	card := &Card{
		ProjectName: "Project1",
		Story: model.UserStory{
			ID:    "US-001",
			Title: "Test Story",
		},
	}

	col.AddCard(card)

	if len(col.cards) != 1 {
		t.Errorf("expected 1 card, got %d", len(col.cards))
	}

	if col.cards[0].Story.ID != "US-001" {
		t.Errorf("expected card ID 'US-001', got %q", col.cards[0].Story.ID)
	}
}

func TestNewBoard_DistributesCardsByStatus(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Title: "Incomplete Story", Status: model.StatusIncomplete},
					{ID: "US-002", Title: "In Progress Story", Status: model.StatusInProgress},
					{ID: "US-003", Title: "Complete Story", Status: model.StatusComplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Check that board has 3 columns
	if len(board.columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(board.columns))
	}

	// Check column titles
	expectedTitles := []string{"Incomplete", "In Progress", "Complete"}
	for i, col := range board.columns {
		if col.title != expectedTitles[i] {
			t.Errorf("column %d: expected title %q, got %q", i, expectedTitles[i], col.title)
		}
	}

	// Check card distribution
	if len(board.columns[0].cards) != 1 {
		t.Errorf("Incomplete column: expected 1 card, got %d", len(board.columns[0].cards))
	}
	if len(board.columns[1].cards) != 1 {
		t.Errorf("In Progress column: expected 1 card, got %d", len(board.columns[1].cards))
	}
	if len(board.columns[2].cards) != 1 {
		t.Errorf("Complete column: expected 1 card, got %d", len(board.columns[2].cards))
	}
}

func TestNewBoard_UnknownStatusDefaultsToIncomplete(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Title: "Unknown Status", Status: "unknown-status"},
					{ID: "US-002", Title: "Empty Status", Status: ""},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Both cards should be in the Incomplete column
	if len(board.columns[0].cards) != 2 {
		t.Errorf("Incomplete column: expected 2 cards (unknown defaults), got %d", len(board.columns[0].cards))
	}
}

func TestNewBoard_MultipleProjects(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "Project1",
				UserStories: []model.UserStory{
					{ID: "P1-001", Title: "Story 1", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/project1/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "Project2",
				UserStories: []model.UserStory{
					{ID: "P2-001", Title: "Story 2", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/project2/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Both cards should be in the Incomplete column
	if len(board.columns[0].cards) != 2 {
		t.Errorf("Incomplete column: expected 2 cards from 2 projects, got %d", len(board.columns[0].cards))
	}

	// Verify project names are different
	projectNames := make(map[string]bool)
	for _, card := range board.columns[0].cards {
		projectNames[card.ProjectName] = true
	}
	if len(projectNames) != 2 {
		t.Error("expected cards from 2 different projects")
	}
}

func TestNewBoard_EmptyParseResults(t *testing.T) {
	board := NewBoard([]*parser.ParseResult{})

	if len(board.columns) != 3 {
		t.Errorf("expected 3 columns even with no data, got %d", len(board.columns))
	}

	for i, col := range board.columns {
		if len(col.cards) != 0 {
			t.Errorf("column %d: expected 0 cards, got %d", i, len(col.cards))
		}
	}
}

func TestColumn_View_EmptyColumn(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(30, 20)

	view := col.View()

	// Check that it contains the title
	if !strings.Contains(view, "Test") {
		t.Error("column view should contain the title")
	}

	// Check that it shows (0 items)
	if !strings.Contains(view, "(0 items)") {
		t.Error("column view should show item count")
	}

	// Check for empty placeholder
	if !strings.Contains(view, "No items") {
		t.Error("empty column should show 'No items' placeholder")
	}
}

func TestColumn_View_WithCards(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 30)

	col.AddCard(&Card{
		ProjectName: "MyProject",
		Story: model.UserStory{
			ID:          "US-001",
			Title:       "Test Title",
			Description: "Test Description",
		},
	})

	view := col.View()

	// Check that it contains card content
	if !strings.Contains(view, "US-001") {
		t.Error("column view should contain card ID")
	}

	if !strings.Contains(view, "(1 items)") {
		t.Error("column view should show correct item count")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 5, "hi"},
		{"", 10, ""},
		{"test", 4, "test"},
		{"testing", 3, "tes"},
		{"ab", 0, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestNewApp(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name:        "TestProject",
				UserStories: []model.UserStory{},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")

	if app.board == nil {
		t.Error("app.board should not be nil")
	}
}

func TestApp_Init(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")
	cmd := app.Init()

	// Without a watcher, Init returns nil
	if cmd != nil {
		t.Error("Init should return nil cmd when no watcher is set")
	}
}

func TestBoard_SetSize(t *testing.T) {
	board := NewBoard([]*parser.ParseResult{})
	board.SetSize(120, 40)

	if board.width != 120 {
		t.Errorf("expected width 120, got %d", board.width)
	}

	if board.height != 40 {
		t.Errorf("expected height 40, got %d", board.height)
	}

	// Check that columns were sized
	for _, col := range board.columns {
		if col.width == 0 {
			t.Error("column width should be set")
		}
	}
}

// Card component tests

func TestNewCard(t *testing.T) {
	story := model.UserStory{
		ID:          "US-001",
		Title:       "Test Story",
		Description: "A test description",
		Status:      model.StatusIncomplete,
	}

	card := NewCard("MyProject", story)

	if card.ProjectName != "MyProject" {
		t.Errorf("expected ProjectName 'MyProject', got %q", card.ProjectName)
	}

	if card.Story.ID != "US-001" {
		t.Errorf("expected Story.ID 'US-001', got %q", card.Story.ID)
	}
}

func TestCard_Render_ContainsProjectName(t *testing.T) {
	card := NewCard("MyProject", model.UserStory{
		ID:          "US-001",
		Title:       "Test Title",
		Description: "Test Description",
	})

	view := card.Render(40)

	if !strings.Contains(view, "MyProject") {
		t.Error("card should display project name")
	}
}

func TestCard_Render_ContainsStoryID(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:          "US-001",
		Title:       "Test Title",
		Description: "Test Description",
	})

	view := card.Render(40)

	if !strings.Contains(view, "US-001") {
		t.Error("card should display story ID prominently")
	}
}

func TestCard_Render_ContainsTitle(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:          "US-001",
		Title:       "Test Title",
		Description: "Test Description",
	})

	view := card.Render(40)

	if !strings.Contains(view, "Test Title") {
		t.Error("card should display title")
	}
}

func TestCard_Render_ContainsDescription(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:          "US-001",
		Title:       "Test Title",
		Description: "Test Description",
	})

	view := card.Render(40)

	if !strings.Contains(view, "Test Description") {
		t.Error("card should display description")
	}
}

func TestCard_Render_TruncatesLongDescription(t *testing.T) {
	longDesc := "This is a very long description that should be truncated when the width is small"
	card := NewCard("Project", model.UserStory{
		ID:          "US-001",
		Title:       "Title",
		Description: longDesc,
	})

	view := card.Render(20) // Small width to force truncation

	// Should contain ellipsis if truncated
	if len(longDesc) > 16 && !strings.Contains(view, "...") {
		t.Error("long description should be truncated with ellipsis")
	}
}

func TestCard_Render_EmptyDescription(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:          "US-001",
		Title:       "Test Title",
		Description: "",
	})

	// Should not panic on empty description
	view := card.Render(40)

	if !strings.Contains(view, "US-001") {
		t.Error("card should still render with empty description")
	}
}

func TestCard_Render_HasBorder(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:    "US-001",
		Title: "Title",
	})

	view := card.Render(40)

	// Check for rounded border characters
	if !strings.ContainsAny(view, "╭╮╯╰│─") {
		t.Error("card should have box-drawing border characters")
	}
}

func TestCard_RenderWithStyle(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:          "US-001",
		Title:       "Test Title",
		Description: "Description",
	})

	customStyle := CardStyle{
		BorderColor:      "255",
		ProjectColor:     "39",
		IDColor:          "205",
		TitleColor:       "255",
		DescriptionColor: "247",
	}

	// Should not panic with custom style
	view := card.RenderWithStyle(40, customStyle)

	if !strings.Contains(view, "US-001") {
		t.Error("card should render with custom style")
	}
}

func TestDefaultCardStyle(t *testing.T) {
	style := DefaultCardStyle()

	// Verify default colors are set
	if style.BorderColor == "" {
		t.Error("BorderColor should have a default value")
	}
	if style.ProjectColor == "" {
		t.Error("ProjectColor should have a default value")
	}
	if style.IDColor == "" {
		t.Error("IDColor should have a default value")
	}
	if style.TitleColor == "" {
		t.Error("TitleColor should have a default value")
	}
	if style.DescriptionColor == "" {
		t.Error("DescriptionColor should have a default value")
	}
}

func TestCard_Render_MinimumWidth(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:    "US-001",
		Title: "Title",
	})

	// Should not panic with very small width
	view := card.Render(5)

	if view == "" {
		t.Error("card should render even with minimum width")
	}
}

// Sorting tests for PM-006

func TestColumn_SortByPriority(t *testing.T) {
	col := NewColumn("Test", "39")

	// Add cards in reverse priority order (highest priority last)
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID:       "US-003",
			Title:    "Low Priority",
			Priority: 10,
		},
	})
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID:       "US-001",
			Title:    "High Priority",
			Priority: 0,
		},
	})
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID:       "US-002",
			Title:    "Medium Priority",
			Priority: 5,
		},
	})

	// Sort by priority
	col.SortByPriority()

	// Verify order: priority 0, 5, 10
	if len(col.cards) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(col.cards))
	}

	expectedIDs := []string{"US-001", "US-002", "US-003"}
	for i, expectedID := range expectedIDs {
		if col.cards[i].Story.ID != expectedID {
			t.Errorf("card %d: expected ID %q, got %q", i, expectedID, col.cards[i].Story.ID)
		}
	}
}

func TestColumn_SortByPriority_EmptyColumn(t *testing.T) {
	col := NewColumn("Test", "39")

	// Should not panic on empty column
	col.SortByPriority()

	if len(col.cards) != 0 {
		t.Error("empty column should remain empty after sort")
	}
}

func TestColumn_SortByPriority_SingleCard(t *testing.T) {
	col := NewColumn("Test", "39")

	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID:       "US-001",
			Priority: 5,
		},
	})

	col.SortByPriority()

	if len(col.cards) != 1 || col.cards[0].Story.ID != "US-001" {
		t.Error("single card should remain in place after sort")
	}
}

func TestColumn_SortByPriority_SamePriority(t *testing.T) {
	col := NewColumn("Test", "39")

	// Add cards with same priority
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID:       "US-001",
			Priority: 5,
		},
	})
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID:       "US-002",
			Priority: 5,
		},
	})

	col.SortByPriority()

	// Both cards should still be present
	if len(col.cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(col.cards))
	}

	// Stable sort maintains original order for equal priorities
	// (Go's sort.Slice is not stable, but the important thing is they're both present)
	priorities := []int{col.cards[0].Story.Priority, col.cards[1].Story.Priority}
	if priorities[0] != 5 || priorities[1] != 5 {
		t.Error("cards with same priority should both be present")
	}
}

func TestNewBoard_SortsCardsByPriority(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-003", Title: "Low Priority", Status: model.StatusIncomplete, Priority: 10},
					{ID: "US-001", Title: "High Priority", Status: model.StatusIncomplete, Priority: 0},
					{ID: "US-002", Title: "Medium Priority", Status: model.StatusIncomplete, Priority: 5},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Check that Incomplete column has cards sorted by priority
	incompleteCol := board.columns[0]
	if len(incompleteCol.cards) != 3 {
		t.Fatalf("expected 3 cards in Incomplete column, got %d", len(incompleteCol.cards))
	}

	// Verify order: priority 0, 5, 10
	expectedOrder := []struct {
		id       string
		priority int
	}{
		{"US-001", 0},
		{"US-002", 5},
		{"US-003", 10},
	}

	for i, expected := range expectedOrder {
		card := incompleteCol.cards[i]
		if card.Story.ID != expected.id {
			t.Errorf("card %d: expected ID %q, got %q", i, expected.id, card.Story.ID)
		}
		if card.Story.Priority != expected.priority {
			t.Errorf("card %d: expected priority %d, got %d", i, expected.priority, card.Story.Priority)
		}
	}
}

func TestNewBoard_SortsAcrossMultipleProjects(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "ProjectA",
				UserStories: []model.UserStory{
					{ID: "A-001", Title: "A Story", Status: model.StatusInProgress, Priority: 5},
				},
			},
			FilePath: "/test/a/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "ProjectB",
				UserStories: []model.UserStory{
					{ID: "B-001", Title: "B Story", Status: model.StatusInProgress, Priority: 1},
				},
			},
			FilePath: "/test/b/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Check that In Progress column has cards sorted by priority across projects
	inProgressCol := board.columns[1]
	if len(inProgressCol.cards) != 2 {
		t.Fatalf("expected 2 cards in In Progress column, got %d", len(inProgressCol.cards))
	}

	// B-001 (priority 1) should come before A-001 (priority 5)
	if inProgressCol.cards[0].Story.ID != "B-001" {
		t.Errorf("expected first card to be B-001 (priority 1), got %s (priority %d)",
			inProgressCol.cards[0].Story.ID, inProgressCol.cards[0].Story.Priority)
	}
	if inProgressCol.cards[1].Story.ID != "A-001" {
		t.Errorf("expected second card to be A-001 (priority 5), got %s (priority %d)",
			inProgressCol.cards[1].Story.ID, inProgressCol.cards[1].Story.Priority)
	}
}

// Read-only interface tests for PM-009

func TestReadOnlyMessage_Constant(t *testing.T) {
	// Verify the read-only message constant is set
	if ReadOnlyMessage == "" {
		t.Error("ReadOnlyMessage constant should not be empty")
	}

	// Should indicate view-only mode
	if !strings.Contains(strings.ToLower(ReadOnlyMessage), "view") && !strings.Contains(strings.ToLower(ReadOnlyMessage), "only") {
		t.Error("ReadOnlyMessage should indicate view-only mode")
	}
}

func TestEditHelpMessage_Constant(t *testing.T) {
	// Verify the edit help message constant is set
	if EditHelpMessage == "" {
		t.Error("EditHelpMessage constant should not be empty")
	}

	// Should mention editing prd.json files
	if !strings.Contains(strings.ToLower(EditHelpMessage), "prd.json") {
		t.Error("EditHelpMessage should mention prd.json files")
	}

	// Should suggest editing directly
	if !strings.Contains(strings.ToLower(EditHelpMessage), "edit") {
		t.Error("EditHelpMessage should suggest editing")
	}
}

func TestApp_View_ContainsReadOnlyIndicator(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")
	app.width = 100
	app.height = 40
	app.board.SetSize(100, 40)

	view := app.View()

	// Check that the view contains the read-only indicator
	if !strings.Contains(view, ReadOnlyMessage) {
		t.Error("App view should contain the ReadOnlyMessage indicator")
	}
}

func TestApp_View_ContainsEditHelpMessage(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")
	app.width = 100
	app.height = 40
	app.board.SetSize(100, 40)

	view := app.View()

	// Check that the view contains the edit help message
	if !strings.Contains(view, EditHelpMessage) {
		t.Error("App view should contain the EditHelpMessage")
	}
}

func TestApp_Update_IgnoresEditKeyBindings(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")

	// List of keybindings that should be ignored (no action taken)
	editKeys := []string{"e", "i", "d", "x", "enter", "backspace", "delete"}

	for _, key := range editKeys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if key == "enter" {
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		} else if key == "backspace" {
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		} else if key == "delete" {
			msg = tea.KeyMsg{Type: tea.KeyDelete}
		}

		_, cmd := app.Update(msg)

		// Edit keybindings should return nil command (no action)
		if cmd != nil {
			t.Errorf("key '%s' should be ignored (return nil cmd) in read-only mode", key)
		}
	}
}

func TestApp_Update_QuitStillWorks(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")

	// 'q' should still quit
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd := app.Update(msg)

	// Quit should return a command
	if cmd == nil {
		t.Error("'q' key should still trigger quit in read-only mode")
	}
}

// File deletion handling tests for PM-010

func TestApp_HandleFileChange_Delete(t *testing.T) {
	// Set up with two projects
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "Project1",
				UserStories: []model.UserStory{
					{ID: "P1-001", Title: "Story 1", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/project1/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "Project2",
				UserStories: []model.UserStory{
					{ID: "P2-001", Title: "Story 2", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/project2/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")
	app.width = 100
	app.height = 40
	app.board.SetSize(100, 40)

	// Verify initial state - 2 cards in Incomplete column
	if len(app.board.columns[0].cards) != 2 {
		t.Fatalf("Expected 2 cards initially, got %d", len(app.board.columns[0].cards))
	}

	// Simulate file deletion event
	deleteMsg := FileChangedMsg{
		FilePath: "/test/project1/prd.json",
		Op:       watcher.OpDelete,
	}

	// Handle the deletion
	app.handleFileChange(deleteMsg)

	// Verify that cards from Project1 are removed
	if len(app.board.columns[0].cards) != 1 {
		t.Errorf("Expected 1 card after deletion, got %d", len(app.board.columns[0].cards))
	}

	// Verify the remaining card is from Project2
	if app.board.columns[0].cards[0].ProjectName != "Project2" {
		t.Errorf("Expected remaining card from Project2, got %s", app.board.columns[0].cards[0].ProjectName)
	}
}

func TestApp_HandleFileChange_DeleteLastProject(t *testing.T) {
	// Set up with one project
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "Project1",
				UserStories: []model.UserStory{
					{ID: "P1-001", Title: "Story 1", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/project1/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")
	app.width = 100
	app.height = 40
	app.board.SetSize(100, 40)

	// Verify initial state
	if len(app.board.columns[0].cards) != 1 {
		t.Fatalf("Expected 1 card initially, got %d", len(app.board.columns[0].cards))
	}

	// Simulate file deletion event
	deleteMsg := FileChangedMsg{
		FilePath: "/test/project1/prd.json",
		Op:       watcher.OpDelete,
	}

	// Handle the deletion
	app.handleFileChange(deleteMsg)

	// Verify all columns are empty after deleting the only project
	for i, col := range app.board.columns {
		if len(col.cards) != 0 {
			t.Errorf("Column %d: expected 0 cards after deleting last project, got %d", i, len(col.cards))
		}
	}
}

func TestApp_HandleFileChange_DeleteNonExistent(t *testing.T) {
	// Set up with one project
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "Project1",
				UserStories: []model.UserStory{
					{ID: "P1-001", Title: "Story 1", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/project1/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")
	app.width = 100
	app.height = 40
	app.board.SetSize(100, 40)

	// Simulate deletion of a non-existent file
	deleteMsg := FileChangedMsg{
		FilePath: "/test/nonexistent/prd.json",
		Op:       watcher.OpDelete,
	}

	// Handle the deletion - should not panic or affect existing cards
	app.handleFileChange(deleteMsg)

	// Verify existing card is still there
	if len(app.board.columns[0].cards) != 1 {
		t.Errorf("Expected 1 card to remain, got %d", len(app.board.columns[0].cards))
	}
}

func TestApp_HandleFileChange_DeleteMultipleStories(t *testing.T) {
	// Set up with one project with multiple stories in different columns
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "Project1",
				UserStories: []model.UserStory{
					{ID: "P1-001", Title: "Incomplete Story", Status: model.StatusIncomplete},
					{ID: "P1-002", Title: "In Progress Story", Status: model.StatusInProgress},
					{ID: "P1-003", Title: "Complete Story", Status: model.StatusComplete},
				},
			},
			FilePath: "/test/project1/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")
	app.width = 100
	app.height = 40
	app.board.SetSize(100, 40)

	// Verify initial state - one card in each column
	if len(app.board.columns[0].cards) != 1 {
		t.Fatalf("Expected 1 card in Incomplete, got %d", len(app.board.columns[0].cards))
	}
	if len(app.board.columns[1].cards) != 1 {
		t.Fatalf("Expected 1 card in In Progress, got %d", len(app.board.columns[1].cards))
	}
	if len(app.board.columns[2].cards) != 1 {
		t.Fatalf("Expected 1 card in Complete, got %d", len(app.board.columns[2].cards))
	}

	// Simulate file deletion event
	deleteMsg := FileChangedMsg{
		FilePath: "/test/project1/prd.json",
		Op:       watcher.OpDelete,
	}

	// Handle the deletion
	app.handleFileChange(deleteMsg)

	// Verify all cards from all columns are removed (no orphaned cards)
	for i, col := range app.board.columns {
		if len(col.cards) != 0 {
			t.Errorf("Column %d: expected 0 cards after deletion, got %d - orphaned cards remain", i, len(col.cards))
		}
	}
}

// Multi-project aggregation tests for PM-011

func TestNewBoard_Handles20PlusProjects(t *testing.T) {
	// Create 25 projects with 3 stories each (one per status)
	// This tests the requirement: "Board handles at least 20 projects without performance issues"
	numProjects := 25
	parseResults := make([]*parser.ParseResult, numProjects)

	for i := 0; i < numProjects; i++ {
		parseResults[i] = &parser.ParseResult{
			PRD: &model.PRD{
				Name: fmt.Sprintf("Project%d", i+1),
				UserStories: []model.UserStory{
					{ID: fmt.Sprintf("P%d-001", i+1), Title: "Incomplete Story", Status: model.StatusIncomplete, Priority: i},
					{ID: fmt.Sprintf("P%d-002", i+1), Title: "In Progress Story", Status: model.StatusInProgress, Priority: i},
					{ID: fmt.Sprintf("P%d-003", i+1), Title: "Complete Story", Status: model.StatusComplete, Priority: i},
				},
			},
			FilePath: fmt.Sprintf("/test/project%d/prd.json", i+1),
		}
	}

	// Board creation should not panic or fail
	board := NewBoard(parseResults)

	// Verify all columns have correct number of cards
	if len(board.columns[0].cards) != numProjects {
		t.Errorf("Incomplete column: expected %d cards, got %d", numProjects, len(board.columns[0].cards))
	}
	if len(board.columns[1].cards) != numProjects {
		t.Errorf("In Progress column: expected %d cards, got %d", numProjects, len(board.columns[1].cards))
	}
	if len(board.columns[2].cards) != numProjects {
		t.Errorf("Complete column: expected %d cards, got %d", numProjects, len(board.columns[2].cards))
	}

	// Verify total cards
	totalCards := len(board.columns[0].cards) + len(board.columns[1].cards) + len(board.columns[2].cards)
	expectedTotal := numProjects * 3
	if totalCards != expectedTotal {
		t.Errorf("Expected %d total cards across all columns, got %d", expectedTotal, totalCards)
	}
}

func TestNewBoard_MultiProjectAggregation_CardsDistinguishedByProjectName(t *testing.T) {
	// Create multiple projects and verify each card shows distinct project name
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "Frontend App",
				UserStories: []model.UserStory{
					{ID: "FE-001", Title: "Build UI", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/projects/frontend/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "Backend API",
				UserStories: []model.UserStory{
					{ID: "BE-001", Title: "Create endpoints", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/projects/backend/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "Mobile App",
				UserStories: []model.UserStory{
					{ID: "MOB-001", Title: "Setup React Native", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/projects/mobile/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// All 3 cards should be in the Incomplete column
	if len(board.columns[0].cards) != 3 {
		t.Fatalf("Expected 3 cards in Incomplete column, got %d", len(board.columns[0].cards))
	}

	// Collect all unique project names
	projectNames := make(map[string]bool)
	for _, card := range board.columns[0].cards {
		projectNames[card.ProjectName] = true
	}

	// Verify we have 3 distinct project names
	if len(projectNames) != 3 {
		t.Errorf("Expected 3 distinct project names, got %d", len(projectNames))
	}

	// Verify specific project names exist
	expectedNames := []string{"Frontend App", "Backend API", "Mobile App"}
	for _, name := range expectedNames {
		if !projectNames[name] {
			t.Errorf("Expected project name %q not found in cards", name)
		}
	}
}

func TestNewBoard_MultiProject_SortsByPriorityAcrossAllProjects(t *testing.T) {
	// Test that cards from different projects are sorted by priority regardless of source
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "ProjectC",
				UserStories: []model.UserStory{
					{ID: "C-001", Title: "Low Priority", Status: model.StatusIncomplete, Priority: 10},
				},
			},
			FilePath: "/projects/c/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "ProjectA",
				UserStories: []model.UserStory{
					{ID: "A-001", Title: "High Priority", Status: model.StatusIncomplete, Priority: 0},
				},
			},
			FilePath: "/projects/a/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "ProjectB",
				UserStories: []model.UserStory{
					{ID: "B-001", Title: "Medium Priority", Status: model.StatusIncomplete, Priority: 5},
				},
			},
			FilePath: "/projects/b/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Verify cards are sorted by priority, not by project order
	incompleteCol := board.columns[0]
	if len(incompleteCol.cards) != 3 {
		t.Fatalf("Expected 3 cards, got %d", len(incompleteCol.cards))
	}

	// Expected order by priority: A-001 (0), B-001 (5), C-001 (10)
	expectedOrder := []struct {
		id       string
		priority int
		project  string
	}{
		{"A-001", 0, "ProjectA"},
		{"B-001", 5, "ProjectB"},
		{"C-001", 10, "ProjectC"},
	}

	for i, expected := range expectedOrder {
		card := incompleteCol.cards[i]
		if card.Story.ID != expected.id {
			t.Errorf("Card %d: expected ID %q, got %q", i, expected.id, card.Story.ID)
		}
		if card.Story.Priority != expected.priority {
			t.Errorf("Card %d: expected priority %d, got %d", i, expected.priority, card.Story.Priority)
		}
		if card.ProjectName != expected.project {
			t.Errorf("Card %d: expected project %q, got %q", i, expected.project, card.ProjectName)
		}
	}
}

func TestBoard_View_With20PlusProjects(t *testing.T) {
	// Test that the board View() method handles 20+ projects without issues
	numProjects := 20
	parseResults := make([]*parser.ParseResult, numProjects)

	for i := 0; i < numProjects; i++ {
		parseResults[i] = &parser.ParseResult{
			PRD: &model.PRD{
				Name: fmt.Sprintf("Project%d", i+1),
				UserStories: []model.UserStory{
					{ID: fmt.Sprintf("P%d-001", i+1), Title: "Story", Status: model.StatusIncomplete},
				},
			},
			FilePath: fmt.Sprintf("/test/project%d/prd.json", i+1),
		}
	}

	board := NewBoard(parseResults)
	board.SetSize(120, 50)

	// View should not panic and should return a non-empty string
	view := board.View()

	if view == "" {
		t.Error("Board view should not be empty with 20+ projects")
	}

	// Verify some project names appear in the view
	if !strings.Contains(view, "Project1") {
		t.Error("Board view should contain at least one project name")
	}
}

func TestApp_Update_With20PlusProjects(t *testing.T) {
	// Test that the App handles 20+ projects without issues
	numProjects := 20
	parseResults := make([]*parser.ParseResult, numProjects)

	for i := 0; i < numProjects; i++ {
		parseResults[i] = &parser.ParseResult{
			PRD: &model.PRD{
				Name: fmt.Sprintf("Project%d", i+1),
				UserStories: []model.UserStory{
					{ID: fmt.Sprintf("P%d-001", i+1), Title: "Story", Status: model.StatusIncomplete},
				},
			},
			FilePath: fmt.Sprintf("/test/project%d/prd.json", i+1),
		}
	}

	app := NewApp(parseResults, nil, "")

	// Initialize should not fail
	cmd := app.Init()
	if cmd != nil {
		t.Error("Init should return nil without watcher")
	}

	// Window resize should handle 20+ projects
	newApp, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	updatedApp := newApp.(*App)

	// Verify board was sized correctly
	if updatedApp.board.width != 120 {
		t.Errorf("Expected width 120, got %d", updatedApp.board.width)
	}

	// View should render without issues
	view := updatedApp.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestMultiProject_CardsFromAllStatusesAggregated(t *testing.T) {
	// Test that cards from multiple projects with different statuses
	// are correctly aggregated into the appropriate columns
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "WebApp",
				UserStories: []model.UserStory{
					{ID: "WEB-001", Title: "Design", Status: model.StatusComplete},
					{ID: "WEB-002", Title: "Build", Status: model.StatusInProgress},
					{ID: "WEB-003", Title: "Test", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/projects/web/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "CLI Tool",
				UserStories: []model.UserStory{
					{ID: "CLI-001", Title: "Parse args", Status: model.StatusComplete},
					{ID: "CLI-002", Title: "Core logic", Status: model.StatusInProgress},
				},
			},
			FilePath: "/projects/cli/prd.json",
		},
		{
			PRD: &model.PRD{
				Name: "Database",
				UserStories: []model.UserStory{
					{ID: "DB-001", Title: "Schema", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/projects/db/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Incomplete column: WEB-003 + DB-001 = 2 cards
	if len(board.columns[0].cards) != 2 {
		t.Errorf("Incomplete column: expected 2 cards, got %d", len(board.columns[0].cards))
	}

	// In Progress column: WEB-002 + CLI-002 = 2 cards
	if len(board.columns[1].cards) != 2 {
		t.Errorf("In Progress column: expected 2 cards, got %d", len(board.columns[1].cards))
	}

	// Complete column: WEB-001 + CLI-001 = 2 cards
	if len(board.columns[2].cards) != 2 {
		t.Errorf("Complete column: expected 2 cards, got %d", len(board.columns[2].cards))
	}

	// Verify project names are preserved on cards
	incompleteProjects := make(map[string]bool)
	for _, card := range board.columns[0].cards {
		incompleteProjects[card.ProjectName] = true
	}
	if !incompleteProjects["WebApp"] || !incompleteProjects["Database"] {
		t.Error("Incomplete column should have cards from WebApp and Database")
	}
}

// Keyboard navigation tests for PM-012

func TestBoard_MoveRight(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete},
					{ID: "US-002", Status: model.StatusInProgress},
					{ID: "US-003", Status: model.StatusComplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Initially at column 0
	if board.SelectedColumn() != 0 {
		t.Errorf("expected initial column 0, got %d", board.SelectedColumn())
	}

	// Move right to column 1
	board.MoveRight()
	if board.SelectedColumn() != 1 {
		t.Errorf("expected column 1 after MoveRight, got %d", board.SelectedColumn())
	}

	// Move right to column 2
	board.MoveRight()
	if board.SelectedColumn() != 2 {
		t.Errorf("expected column 2 after MoveRight, got %d", board.SelectedColumn())
	}

	// Try to move right past the last column (should stay at 2)
	board.MoveRight()
	if board.SelectedColumn() != 2 {
		t.Errorf("expected column 2 (clamped), got %d", board.SelectedColumn())
	}
}

func TestBoard_MoveLeft(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Move to column 2 first
	board.MoveRight()
	board.MoveRight()

	// Move left to column 1
	board.MoveLeft()
	if board.SelectedColumn() != 1 {
		t.Errorf("expected column 1 after MoveLeft, got %d", board.SelectedColumn())
	}

	// Move left to column 0
	board.MoveLeft()
	if board.SelectedColumn() != 0 {
		t.Errorf("expected column 0 after MoveLeft, got %d", board.SelectedColumn())
	}

	// Try to move left past the first column (should stay at 0)
	board.MoveLeft()
	if board.SelectedColumn() != 0 {
		t.Errorf("expected column 0 (clamped), got %d", board.SelectedColumn())
	}
}

func TestBoard_MoveUpDown(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete, Priority: 0},
					{ID: "US-002", Status: model.StatusIncomplete, Priority: 1},
					{ID: "US-003", Status: model.StatusIncomplete, Priority: 2},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Initially at card 0
	if board.SelectedCardIndex() != 0 {
		t.Errorf("expected initial card 0, got %d", board.SelectedCardIndex())
	}

	// Move down
	board.MoveDown()
	if board.SelectedCardIndex() != 1 {
		t.Errorf("expected card 1 after MoveDown, got %d", board.SelectedCardIndex())
	}

	board.MoveDown()
	if board.SelectedCardIndex() != 2 {
		t.Errorf("expected card 2 after MoveDown, got %d", board.SelectedCardIndex())
	}

	// Try to move down past the last card (should stay at 2)
	board.MoveDown()
	if board.SelectedCardIndex() != 2 {
		t.Errorf("expected card 2 (clamped), got %d", board.SelectedCardIndex())
	}

	// Move up
	board.MoveUp()
	if board.SelectedCardIndex() != 1 {
		t.Errorf("expected card 1 after MoveUp, got %d", board.SelectedCardIndex())
	}

	board.MoveUp()
	if board.SelectedCardIndex() != 0 {
		t.Errorf("expected card 0 after MoveUp, got %d", board.SelectedCardIndex())
	}

	// Try to move up past the first card (should stay at 0)
	board.MoveUp()
	if board.SelectedCardIndex() != 0 {
		t.Errorf("expected card 0 (clamped), got %d", board.SelectedCardIndex())
	}
}

func TestBoard_CycleColumn(t *testing.T) {
	board := NewBoard([]*parser.ParseResult{})

	// Initially at column 0
	if board.SelectedColumn() != 0 {
		t.Errorf("expected initial column 0, got %d", board.SelectedColumn())
	}

	// Cycle through columns
	board.CycleColumn()
	if board.SelectedColumn() != 1 {
		t.Errorf("expected column 1, got %d", board.SelectedColumn())
	}

	board.CycleColumn()
	if board.SelectedColumn() != 2 {
		t.Errorf("expected column 2, got %d", board.SelectedColumn())
	}

	// Should wrap around to column 0
	board.CycleColumn()
	if board.SelectedColumn() != 0 {
		t.Errorf("expected column 0 (wrapped), got %d", board.SelectedColumn())
	}
}

func TestBoard_SelectedCard(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Title: "First Story", Status: model.StatusIncomplete},
					{ID: "US-002", Title: "Second Story", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// First card should be selected
	card := board.SelectedCard()
	if card == nil {
		t.Fatal("SelectedCard should not be nil")
	}
	if card.Story.ID != "US-001" {
		t.Errorf("expected US-001, got %s", card.Story.ID)
	}

	// Move down and check
	board.MoveDown()
	card = board.SelectedCard()
	if card == nil {
		t.Fatal("SelectedCard should not be nil after MoveDown")
	}
	if card.Story.ID != "US-002" {
		t.Errorf("expected US-002, got %s", card.Story.ID)
	}
}

func TestBoard_SelectedCard_EmptyColumn(t *testing.T) {
	// Create board with empty Incomplete column (only complete stories)
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusComplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Initial column (Incomplete) is empty
	card := board.SelectedCard()
	if card != nil {
		t.Error("SelectedCard should be nil for empty column")
	}

	// Move to Complete column (has a card)
	board.MoveRight()
	board.MoveRight()
	card = board.SelectedCard()
	if card == nil {
		t.Error("SelectedCard should not be nil in Complete column")
	}
}

func TestBoard_NavigationClampsCardIndex(t *testing.T) {
	// Create board where columns have different numbers of cards
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete, Priority: 0},
					{ID: "US-002", Status: model.StatusIncomplete, Priority: 1},
					{ID: "US-003", Status: model.StatusIncomplete, Priority: 2},
					{ID: "US-004", Status: model.StatusInProgress}, // Only 1 card in this column
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Navigate to last card in Incomplete column (card index 2)
	board.MoveDown()
	board.MoveDown()
	if board.SelectedCardIndex() != 2 {
		t.Errorf("expected card 2, got %d", board.SelectedCardIndex())
	}

	// Move right to In Progress column (only 1 card)
	// Card index should be clamped to 0
	board.MoveRight()
	if board.SelectedCardIndex() != 0 {
		t.Errorf("expected card index clamped to 0, got %d", board.SelectedCardIndex())
	}
}

func TestApp_Update_ArrowKeyNavigation(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete},
					{ID: "US-002", Status: model.StatusInProgress},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")

	// Test arrow key navigation
	tests := []struct {
		key         string
		expectedCol int
		expectedRow int
	}{
		{"right", 1, 0},
		{"left", 0, 0},
		{"down", 0, 0}, // Only 1 card in Incomplete, stays at 0
	}

	for _, tt := range tests {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
		if tt.key == "up" {
			msg = tea.KeyMsg{Type: tea.KeyUp}
		} else if tt.key == "down" {
			msg = tea.KeyMsg{Type: tea.KeyDown}
		} else if tt.key == "left" {
			msg = tea.KeyMsg{Type: tea.KeyLeft}
		} else if tt.key == "right" {
			msg = tea.KeyMsg{Type: tea.KeyRight}
		}

		newModel, _ := app.Update(msg)
		app = newModel.(*App)
	}
}

func TestApp_Update_HjklNavigation(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete, Priority: 0},
					{ID: "US-002", Status: model.StatusIncomplete, Priority: 1},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")

	// Test j (down)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	newModel, _ := app.Update(msg)
	app = newModel.(*App)
	if app.board.SelectedCardIndex() != 1 {
		t.Errorf("expected card 1 after 'j', got %d", app.board.SelectedCardIndex())
	}

	// Test k (up)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	newModel, _ = app.Update(msg)
	app = newModel.(*App)
	if app.board.SelectedCardIndex() != 0 {
		t.Errorf("expected card 0 after 'k', got %d", app.board.SelectedCardIndex())
	}

	// Test l (right)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	newModel, _ = app.Update(msg)
	app = newModel.(*App)
	if app.board.SelectedColumn() != 1 {
		t.Errorf("expected column 1 after 'l', got %d", app.board.SelectedColumn())
	}

	// Test h (left)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}
	newModel, _ = app.Update(msg)
	app = newModel.(*App)
	if app.board.SelectedColumn() != 0 {
		t.Errorf("expected column 0 after 'h', got %d", app.board.SelectedColumn())
	}
}

func TestApp_Update_TabCyclesColumns(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")

	// Test Tab cycles through columns
	msg := tea.KeyMsg{Type: tea.KeyTab}

	newModel, _ := app.Update(msg)
	app = newModel.(*App)
	if app.board.SelectedColumn() != 1 {
		t.Errorf("expected column 1 after Tab, got %d", app.board.SelectedColumn())
	}

	newModel, _ = app.Update(msg)
	app = newModel.(*App)
	if app.board.SelectedColumn() != 2 {
		t.Errorf("expected column 2 after Tab, got %d", app.board.SelectedColumn())
	}

	newModel, _ = app.Update(msg)
	app = newModel.(*App)
	if app.board.SelectedColumn() != 0 {
		t.Errorf("expected column 0 after Tab (wrap), got %d", app.board.SelectedColumn())
	}
}

func TestApp_Update_EnterExpandsCard(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Title: "Test Story", Description: "Full description here", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")

	// Initially no expanded card
	if app.expandedCard != nil {
		t.Error("expandedCard should be nil initially")
	}

	// Press Enter to expand
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	if app.expandedCard == nil {
		t.Error("expandedCard should not be nil after Enter")
	}
	if app.expandedCard.Story.ID != "US-001" {
		t.Errorf("expected expanded card US-001, got %s", app.expandedCard.Story.ID)
	}
}

func TestApp_Update_SpaceExpandsCard(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Title: "Test Story", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")

	// Press Space to expand
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}
	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	if app.expandedCard == nil {
		t.Error("expandedCard should not be nil after Space")
	}
}

func TestApp_Update_EscClosesExpandedCard(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")

	// Expand a card first
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	if app.expandedCard == nil {
		t.Fatal("expandedCard should be set")
	}

	// Press Esc to close
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = app.Update(msg)
	app = newModel.(*App)

	if app.expandedCard != nil {
		t.Error("expandedCard should be nil after Esc")
	}
}

func TestApp_Update_QuestionMarkTogglesHelp(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")

	// Initially help is not shown
	if app.showHelp {
		t.Error("showHelp should be false initially")
	}

	// Press ? to show help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	if !app.showHelp {
		t.Error("showHelp should be true after pressing ?")
	}

	// Press ? again to hide help
	newModel, _ = app.Update(msg)
	app = newModel.(*App)

	if app.showHelp {
		t.Error("showHelp should be false after pressing ? again")
	}
}

func TestApp_Update_EscClosesHelp(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")

	// Show help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	if !app.showHelp {
		t.Fatal("showHelp should be true")
	}

	// Press Esc to close help
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = app.Update(msg)
	app = newModel.(*App)

	if app.showHelp {
		t.Error("showHelp should be false after Esc")
	}
}

func TestApp_Update_HelpBlocksNavigation(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete},
					{ID: "US-002", Status: model.StatusInProgress},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")
	initialCol := app.board.SelectedColumn()

	// Show help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	// Try to navigate while help is showing
	msg = tea.KeyMsg{Type: tea.KeyRight}
	newModel, _ = app.Update(msg)
	app = newModel.(*App)

	// Column should not have changed
	if app.board.SelectedColumn() != initialCol {
		t.Error("navigation should be blocked while help is showing")
	}
}

func TestApp_View_ShowsHelpOverlay(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")
	app.width = 100
	app.height = 40
	app.board.SetSize(100, 40)
	app.helpOverlay.SetSize(100, 40)

	// Show help
	app.showHelp = true

	view := app.View()

	// Check for help content
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("help view should contain 'Keyboard Shortcuts' title")
	}
}

func TestApp_View_ShowsExpandedCard(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Title: "Test Story", Description: "A detailed description", Status: model.StatusIncomplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")
	app.width = 100
	app.height = 40
	app.board.SetSize(100, 40)

	// Expand the card
	app.expandedCard = app.board.SelectedCard()

	view := app.View()

	// Check for expanded card content
	if !strings.Contains(view, "US-001") {
		t.Error("expanded view should contain story ID")
	}
	if !strings.Contains(view, "A detailed description") {
		t.Error("expanded view should contain full description")
	}
}

// Help overlay tests

func TestNewHelpOverlay(t *testing.T) {
	h := NewHelpOverlay()
	if h == nil {
		t.Error("NewHelpOverlay should not return nil")
	}
}

func TestHelpOverlay_SetSize(t *testing.T) {
	h := NewHelpOverlay()
	h.SetSize(100, 50)

	if h.width != 100 {
		t.Errorf("expected width 100, got %d", h.width)
	}
	if h.height != 50 {
		t.Errorf("expected height 50, got %d", h.height)
	}
}

func TestHelpOverlay_View(t *testing.T) {
	h := NewHelpOverlay()
	h.SetSize(100, 50)

	view := h.View()

	// Check for title
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("help overlay should contain title")
	}

	// Check for some shortcuts (case-insensitive check)
	viewLower := strings.ToLower(view)
	if !strings.Contains(viewLower, "quit") {
		t.Error("help overlay should mention quit")
	}

	// Check for close hint
	if !strings.Contains(viewLower, "close") || !strings.Contains(view, "Esc") {
		t.Error("help overlay should mention how to close")
	}
}

func TestHelpShortcuts_Contains_AllRequiredKeys(t *testing.T) {
	requiredKeys := []string{"↑", "↓", "←", "→", "Tab", "Enter", "Esc", "?", "q"}

	for _, required := range requiredKeys {
		found := false
		for _, shortcut := range HelpShortcuts {
			if strings.Contains(shortcut.Key, required) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("HelpShortcuts should contain %q", required)
		}
	}
}

// Card expanded view tests

func TestCard_RenderExpanded(t *testing.T) {
	card := NewCard("TestProject", model.UserStory{
		ID:          "US-001",
		Title:       "Test Story Title",
		Description: "This is a detailed description of the story.",
		Status:      model.StatusInProgress,
		Priority:    2,
		AcceptanceCriteria: []string{
			"First criterion",
			"Second criterion",
		},
	})

	view := card.RenderExpanded(60)

	// Check all expected content
	if !strings.Contains(view, "TestProject") {
		t.Error("expanded card should contain project name")
	}
	if !strings.Contains(view, "US-001") {
		t.Error("expanded card should contain story ID")
	}
	if !strings.Contains(view, "Test Story Title") {
		t.Error("expanded card should contain title")
	}
	if !strings.Contains(view, "This is a detailed description") {
		t.Error("expanded card should contain full description")
	}
	if !strings.Contains(view, "in-progress") {
		t.Error("expanded card should show status")
	}
	if !strings.Contains(view, "First criterion") {
		t.Error("expanded card should show acceptance criteria")
	}
	if !strings.Contains(view, "Second criterion") {
		t.Error("expanded card should show all acceptance criteria")
	}
	if !strings.Contains(view, "Priority") {
		t.Error("expanded card should show priority")
	}
	if !strings.Contains(view, "Esc") || !strings.Contains(view, "close") {
		t.Error("expanded card should show close hint")
	}
}

func TestCard_RenderExpanded_EmptyAcceptanceCriteria(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:                 "US-001",
		Title:              "Title",
		Description:        "Description",
		Status:             model.StatusIncomplete,
		AcceptanceCriteria: []string{},
	})

	// Should not panic
	view := card.RenderExpanded(60)

	if view == "" {
		t.Error("expanded card should render even with empty acceptance criteria")
	}
}

func TestCard_RenderSelected(t *testing.T) {
	card := NewCard("Project", model.UserStory{
		ID:    "US-001",
		Title: "Title",
	})

	// Render not selected
	normalView := card.RenderSelected(40, false)

	// Render selected
	selectedView := card.RenderSelected(40, true)

	// Both should contain the content
	if !strings.Contains(normalView, "US-001") {
		t.Error("normal view should contain ID")
	}
	if !strings.Contains(selectedView, "US-001") {
		t.Error("selected view should contain ID")
	}

	// Selected view should have different styling (cyan border)
	// We can't easily test ANSI colors, but we can verify both render without errors
}

func TestSelectedCardStyle(t *testing.T) {
	style := SelectedCardStyle()

	// Verify colors are set
	if style.BorderColor == "" {
		t.Error("SelectedCardStyle should have BorderColor")
	}
	if style.ProjectColor == "" {
		t.Error("SelectedCardStyle should have ProjectColor")
	}

	// Border should be cyan (39)
	if style.BorderColor != lipgloss.Color("39") {
		t.Error("SelectedCardStyle border should be cyan (39)")
	}
}

// Column selection state tests

func TestColumn_SetSelected(t *testing.T) {
	col := NewColumn("Test", "39")
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID: "US-001",
		},
	})
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID: "US-002",
		},
	})

	// Initially not selected
	if col.isSelected {
		t.Error("column should not be selected initially")
	}

	// Set selected with card 1
	col.SetSelected(true, 1)

	if !col.isSelected {
		t.Error("column should be selected")
	}
	if col.selectedCard != 1 {
		t.Errorf("selectedCard should be 1, got %d", col.selectedCard)
	}

	// Deselect
	col.SetSelected(false, 0)

	if col.isSelected {
		t.Error("column should not be selected after deselect")
	}
	if col.selectedCard != -1 {
		t.Errorf("selectedCard should be -1 when not selected, got %d", col.selectedCard)
	}
}

// Helper function tests

func TestWrapText(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		contains []string
	}{
		{"Hello world", 20, []string{"Hello world"}},
		{"Hello world this is a test", 10, []string{"Hello", "world", "this"}},
		{"", 10, []string{""}},
		{"Short", 0, []string{"Short"}},
	}

	for _, tt := range tests {
		result := wrapText(tt.input, tt.width)
		for _, expected := range tt.contains {
			if !strings.Contains(result, expected) && expected != "" {
				t.Errorf("wrapText(%q, %d) should contain %q, got %q", tt.input, tt.width, expected, result)
			}
		}
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-5, "-5"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{0, 0, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// Scrollable columns tests for PM-013

func TestColumn_ScrollOffset_Initial(t *testing.T) {
	col := NewColumn("Test", "39")

	// Initial scroll offset should be 0
	if col.ScrollOffset() != 0 {
		t.Errorf("expected initial scroll offset 0, got %d", col.ScrollOffset())
	}
}

func TestColumn_VisibleCardCount(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 30) // Height of 30

	visibleCount := col.visibleCardCount()

	// Should have a positive visible count
	if visibleCount <= 0 {
		t.Errorf("expected positive visible card count, got %d", visibleCount)
	}
}

func TestColumn_ScrollUp(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 20)

	// Add many cards to enable scrolling
	for i := 0; i < 20; i++ {
		col.AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	// Set scroll offset to 5
	col.SetScrollOffset(5)

	if col.ScrollOffset() != 5 {
		t.Errorf("expected scroll offset 5, got %d", col.ScrollOffset())
	}

	// Scroll up
	col.ScrollUp()

	if col.ScrollOffset() != 4 {
		t.Errorf("expected scroll offset 4 after ScrollUp, got %d", col.ScrollOffset())
	}

	// Scroll up to 0
	col.SetScrollOffset(0)
	col.ScrollUp()

	// Should stay at 0
	if col.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0 (clamped at top), got %d", col.ScrollOffset())
	}
}

func TestColumn_ScrollDown(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 20)

	// Add many cards to enable scrolling
	for i := 0; i < 20; i++ {
		col.AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	// Initially at 0
	if col.ScrollOffset() != 0 {
		t.Errorf("expected initial scroll offset 0, got %d", col.ScrollOffset())
	}

	// Scroll down
	col.ScrollDown()

	if col.ScrollOffset() != 1 {
		t.Errorf("expected scroll offset 1 after ScrollDown, got %d", col.ScrollOffset())
	}

	// Scroll to max
	maxOffset := col.maxScrollOffset()
	col.SetScrollOffset(maxOffset)

	// Try to scroll down past max
	col.ScrollDown()

	// Should stay at max
	if col.ScrollOffset() != maxOffset {
		t.Errorf("expected scroll offset %d (clamped at max), got %d", maxOffset, col.ScrollOffset())
	}
}

func TestColumn_SetScrollOffset(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 20)

	// Add cards
	for i := 0; i < 20; i++ {
		col.AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	// Set valid offset
	col.SetScrollOffset(5)
	if col.ScrollOffset() != 5 {
		t.Errorf("expected scroll offset 5, got %d", col.ScrollOffset())
	}

	// Set negative offset (should be clamped to 0)
	col.SetScrollOffset(-10)
	if col.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0 (clamped), got %d", col.ScrollOffset())
	}

	// Set offset beyond max (should be clamped)
	col.SetScrollOffset(1000)
	maxOffset := col.maxScrollOffset()
	if col.ScrollOffset() != maxOffset {
		t.Errorf("expected scroll offset %d (clamped to max), got %d", maxOffset, col.ScrollOffset())
	}
}

func TestColumn_MaxScrollOffset(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 20)

	// With no cards, max offset is 0
	if col.maxScrollOffset() != 0 {
		t.Errorf("expected max offset 0 for empty column, got %d", col.maxScrollOffset())
	}

	// Add cards
	for i := 0; i < 20; i++ {
		col.AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	maxOffset := col.maxScrollOffset()
	visibleCount := col.visibleCardCount()
	expectedMax := 20 - visibleCount
	if expectedMax < 0 {
		expectedMax = 0
	}

	if maxOffset != expectedMax {
		t.Errorf("expected max offset %d, got %d", expectedMax, maxOffset)
	}
}

func TestColumn_EnsureCardVisible_ScrollsDown(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 20)

	// Add many cards
	for i := 0; i < 20; i++ {
		col.AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	// Start at top
	col.SetScrollOffset(0)

	// Select a card that's out of view
	visibleCount := col.visibleCardCount()
	targetCard := visibleCount + 2 // Card that should be below the visible area

	col.SetSelected(true, targetCard)

	// Scroll offset should have been adjusted to show the card
	if col.ScrollOffset() == 0 {
		t.Error("scroll offset should have changed to show selected card")
	}

	// Card should now be visible
	if targetCard < col.ScrollOffset() || targetCard >= col.ScrollOffset()+visibleCount {
		t.Errorf("card %d should be visible, scroll offset is %d, visible count is %d",
			targetCard, col.ScrollOffset(), visibleCount)
	}
}

func TestColumn_EnsureCardVisible_ScrollsUp(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 20)

	// Add many cards
	for i := 0; i < 20; i++ {
		col.AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	// Start at bottom
	col.SetScrollOffset(10)

	// Select card 2 which is above the visible area
	col.SetSelected(true, 2)

	// Scroll offset should have been adjusted
	if col.ScrollOffset() > 2 {
		t.Errorf("expected scroll offset <= 2, got %d", col.ScrollOffset())
	}
}

func TestColumn_View_ShowsScrollIndicator_WhenScrollable(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 20)

	// Add many cards to make column scrollable
	for i := 0; i < 20; i++ {
		col.AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID:    fmt.Sprintf("US-%03d", i+1),
				Title: "Story",
			},
		})
	}

	// Scroll down a bit
	col.SetScrollOffset(3)

	view := col.View()

	// Should show scroll position indicator (X-Y of Z format)
	if !strings.Contains(view, "of") {
		t.Error("scrollable column should show scroll position indicator")
	}

	// Should show "more above" indicator
	if !strings.Contains(view, "above") && !strings.Contains(view, "▲") {
		t.Error("should show 'more above' indicator when scrolled down")
	}
}

func TestColumn_View_ShowsMoreBelowIndicator(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 20)

	// Add many cards
	for i := 0; i < 20; i++ {
		col.AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID:    fmt.Sprintf("US-%03d", i+1),
				Title: "Story",
			},
		})
	}

	// At top, there should be more below
	col.SetScrollOffset(0)

	view := col.View()

	if !strings.Contains(view, "below") && !strings.Contains(view, "▼") {
		t.Error("should show 'more below' indicator when not at bottom")
	}
}

func TestColumn_View_HidesScrollIndicators_WhenNoScrollNeeded(t *testing.T) {
	col := NewColumn("Test", "39")
	col.SetSize(40, 100) // Large height

	// Add just a few cards that all fit
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID:    "US-001",
			Title: "Story 1",
		},
	})
	col.AddCard(&Card{
		ProjectName: "Project",
		Story: model.UserStory{
			ID:    "US-002",
			Title: "Story 2",
		},
	})

	view := col.View()

	// Should not show scroll indicators
	if strings.Contains(view, "more above") || strings.Contains(view, "more below") {
		t.Error("should not show scroll indicators when all cards fit")
	}
}

func TestBoard_ScrollColumnUp(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name:        "TestProject",
				UserStories: []model.UserStory{},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Add many cards to the first column for testing
	for i := 0; i < 20; i++ {
		board.columns[0].AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	board.SetSize(120, 30)

	// Set initial scroll offset
	board.columns[0].SetScrollOffset(5)

	// Scroll column 0 up
	board.ScrollColumnUp(0)

	if board.columns[0].ScrollOffset() != 4 {
		t.Errorf("expected scroll offset 4, got %d", board.columns[0].ScrollOffset())
	}

	// Invalid column index should not panic
	board.ScrollColumnUp(-1)
	board.ScrollColumnUp(10)
}

func TestBoard_ScrollColumnDown(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name:        "TestProject",
				UserStories: []model.UserStory{},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Add many cards to the second column for testing
	for i := 0; i < 20; i++ {
		board.columns[1].AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	board.SetSize(120, 30)

	// Scroll column 1 down
	board.ScrollColumnDown(1)

	if board.columns[1].ScrollOffset() != 1 {
		t.Errorf("expected scroll offset 1, got %d", board.columns[1].ScrollOffset())
	}

	// Invalid column index should not panic
	board.ScrollColumnDown(-1)
	board.ScrollColumnDown(10)
}

func TestColumn_IndependentScrolling(t *testing.T) {
	// Create a board with cards in all columns
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name: "TestProject",
				UserStories: []model.UserStory{
					{ID: "US-001", Status: model.StatusIncomplete},
					{ID: "US-002", Status: model.StatusInProgress},
					{ID: "US-003", Status: model.StatusComplete},
				},
			},
			FilePath: "/test/prd.json",
		},
	}

	board := NewBoard(parseResults)

	// Add more cards to each column
	for i := 0; i < 10; i++ {
		board.columns[0].AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID:     fmt.Sprintf("INC-%03d", i+1),
				Status: model.StatusIncomplete,
			},
		})
		board.columns[1].AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID:     fmt.Sprintf("IP-%03d", i+1),
				Status: model.StatusInProgress,
			},
		})
		board.columns[2].AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID:     fmt.Sprintf("COMP-%03d", i+1),
				Status: model.StatusComplete,
			},
		})
	}

	board.SetSize(120, 20)

	// Scroll each column independently
	board.columns[0].SetScrollOffset(3)
	board.columns[1].SetScrollOffset(5)
	board.columns[2].SetScrollOffset(1)

	// Verify each column has independent scroll offset
	if board.columns[0].ScrollOffset() != 3 {
		t.Errorf("column 0: expected offset 3, got %d", board.columns[0].ScrollOffset())
	}
	if board.columns[1].ScrollOffset() != 5 {
		t.Errorf("column 1: expected offset 5, got %d", board.columns[1].ScrollOffset())
	}
	if board.columns[2].ScrollOffset() != 1 {
		t.Errorf("column 2: expected offset 1, got %d", board.columns[2].ScrollOffset())
	}
}

func TestApp_KeyboardNavigation_TriggersScrolling(t *testing.T) {
	// Create app with many cards in one column
	stories := make([]model.UserStory, 20)
	for i := 0; i < 20; i++ {
		stories[i] = model.UserStory{
			ID:       fmt.Sprintf("US-%03d", i+1),
			Status:   model.StatusIncomplete,
			Priority: i,
		}
	}

	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name:        "TestProject",
				UserStories: stories,
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")
	app.width = 120
	app.height = 30
	app.board.SetSize(120, 30)

	// Navigate down multiple times to trigger scrolling
	for i := 0; i < 10; i++ {
		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ := app.Update(msg)
		app = newModel.(*App)
	}

	// The scroll offset should have been adjusted to keep the selected card visible
	col := app.board.columns[app.board.SelectedColumn()]
	selectedCard := app.board.SelectedCardIndex()
	scrollOffset := col.ScrollOffset()
	visibleCount := col.visibleCardCount()

	// Selected card should be within visible range
	if selectedCard < scrollOffset || selectedCard >= scrollOffset+visibleCount {
		t.Errorf("selected card %d should be visible (offset=%d, visible=%d)",
			selectedCard, scrollOffset, visibleCount)
	}
}

func TestApp_MouseWheelScroll(t *testing.T) {
	parseResults := []*parser.ParseResult{
		{
			PRD: &model.PRD{
				Name:        "TestProject",
				UserStories: []model.UserStory{},
			},
			FilePath: "/test/prd.json",
		},
	}

	app := NewApp(parseResults, nil, "")
	app.width = 120
	app.height = 30
	app.board.SetSize(120, 30)

	// Add cards to first column
	for i := 0; i < 20; i++ {
		app.board.columns[0].AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	// Simulate mouse wheel down on first column (X position determines column)
	msg := tea.MouseMsg{
		X:    10, // First column
		Y:    10,
		Type: tea.MouseWheelDown,
	}

	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	// First column should have scrolled down
	if app.board.columns[0].ScrollOffset() != 1 {
		t.Errorf("expected scroll offset 1 after wheel down, got %d", app.board.columns[0].ScrollOffset())
	}

	// Simulate mouse wheel up
	msg = tea.MouseMsg{
		X:    10,
		Y:    10,
		Type: tea.MouseWheelUp,
	}

	newModel, _ = app.Update(msg)
	app = newModel.(*App)

	// Should have scrolled back up
	if app.board.columns[0].ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0 after wheel up, got %d", app.board.columns[0].ScrollOffset())
	}
}

func TestApp_MouseWheelScroll_BlockedByOverlay(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")
	app.width = 120
	app.height = 30
	app.board.SetSize(120, 30)

	// Show help overlay
	app.showHelp = true

	// Add cards to first column
	for i := 0; i < 20; i++ {
		app.board.columns[0].AddCard(&Card{
			ProjectName: "Project",
			Story: model.UserStory{
				ID: fmt.Sprintf("US-%03d", i+1),
			},
		})
	}

	initialOffset := app.board.columns[0].ScrollOffset()

	// Try to scroll with mouse wheel while overlay is shown
	msg := tea.MouseMsg{
		X:    10,
		Y:    10,
		Type: tea.MouseWheelDown,
	}

	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	// Should not have scrolled
	if app.board.columns[0].ScrollOffset() != initialOffset {
		t.Error("mouse scroll should be blocked when help overlay is shown")
	}
}

// Graceful exit and cleanup tests for PM-014

func TestApp_Update_QuitViaQKey(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := app.Update(msg)

	// Should return Quit command
	if cmd == nil {
		t.Error("pressing 'q' should return a Quit command")
	}

	// Verify it's a quit command by checking the command type
	// tea.Quit returns a special quit message
	quitMsg := cmd()
	if quitMsg != tea.Quit() {
		t.Error("command should be tea.Quit")
	}
}

func TestApp_Update_QuitViaCtrlC(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := app.Update(msg)

	// Should return Quit command
	if cmd == nil {
		t.Error("pressing Ctrl+C should return a Quit command")
	}
}

func TestApp_Update_WatcherStoppedMsg(t *testing.T) {
	app := NewApp([]*parser.ParseResult{}, nil, "")

	// Simulate receiving a WatcherStoppedMsg
	msg := WatcherStoppedMsg{}
	_, cmd := app.Update(msg)

	// Should return nil command (no more listening for file changes)
	if cmd != nil {
		t.Error("WatcherStoppedMsg should return nil command")
	}
}

func TestApp_ListenForFileChanges_HandlesClosedChannel(t *testing.T) {
	// Create a watcher, start it, then stop it
	w, err := watcher.New()
	if err != nil {
		t.Fatalf("watcher.New() returned error: %v", err)
	}

	app := NewApp([]*parser.ParseResult{}, w, "")

	// Start the watcher
	go w.Start()

	// Stop the watcher to close channels
	w.Stop()

	// Get the listen command
	cmd := app.listenForFileChanges()

	// Execute the command - it should return WatcherStoppedMsg, not block forever
	done := make(chan tea.Msg)
	go func() {
		done <- cmd()
	}()

	select {
	case msg := <-done:
		// Should be WatcherStoppedMsg
		if _, ok := msg.(WatcherStoppedMsg); !ok {
			t.Errorf("expected WatcherStoppedMsg when watcher is stopped, got %T", msg)
		}
	case <-time.After(1 * time.Second):
		t.Error("listenForFileChanges should not block when watcher is stopped")
	}
}

func TestApp_WithWatcher_GracefulShutdown(t *testing.T) {
	// Create a watcher
	w, err := watcher.New()
	if err != nil {
		t.Fatalf("watcher.New() returned error: %v", err)
	}

	// Create app with watcher
	app := NewApp([]*parser.ParseResult{}, w, "")

	// Start watcher
	go w.Start()

	// Get the initial command (should be listenForFileChanges)
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init() should return a command when watcher is present")
	}

	// Stop the watcher
	w.Stop()

	// Verify watcher is stopped
	if !w.Stopped() {
		t.Error("watcher should be stopped")
	}

	// App should handle the stopped state gracefully
	msg := WatcherStoppedMsg{}
	_, resultCmd := app.Update(msg)

	// Should return nil (no more listening)
	if resultCmd != nil {
		t.Error("after WatcherStoppedMsg, should return nil command")
	}
}

func TestApp_NoWatcher_Init(t *testing.T) {
	// App without watcher
	app := NewApp([]*parser.ParseResult{}, nil, "")

	// Init should return nil (no file watching)
	cmd := app.Init()
	if cmd != nil {
		t.Error("Init() should return nil when no watcher is present")
	}
}

func TestWatcherStoppedMsg_Type(t *testing.T) {
	// Verify WatcherStoppedMsg is a valid tea.Msg
	var msg tea.Msg = WatcherStoppedMsg{}

	// Should be the correct type
	if _, ok := msg.(WatcherStoppedMsg); !ok {
		t.Error("WatcherStoppedMsg should implement tea.Msg")
	}
}
