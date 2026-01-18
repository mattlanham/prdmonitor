package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
