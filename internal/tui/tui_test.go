package tui

import (
	"strings"
	"testing"

	"lanham/prdmonitor/internal/model"
	"lanham/prdmonitor/internal/parser"
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

func TestTruncate(t *testing.T) {
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
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
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

	app := NewApp(parseResults)

	if app.board == nil {
		t.Error("app.board should not be nil")
	}
}

func TestApp_Init(t *testing.T) {
	app := NewApp([]*parser.ParseResult{})
	cmd := app.Init()

	if cmd != nil {
		t.Error("Init should return nil cmd")
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
