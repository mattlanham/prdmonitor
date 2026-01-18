// board.go provides the Kanban board view with three columns.
package tui

import (
	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/model"
	"lanham/prdmonitor/internal/parser"
)

// Board represents the Kanban board with three columns.
type Board struct {
	columns []*Column
	width   int
	height  int
}

// NewBoard creates a new Board from the parsed PRD results.
func NewBoard(parseResults []*parser.ParseResult) *Board {
	// Create the three columns
	incompleteCol := NewColumn("Incomplete", lipgloss.Color("203"))  // Red-ish
	inProgressCol := NewColumn("In Progress", lipgloss.Color("220")) // Yellow
	completeCol := NewColumn("Complete", lipgloss.Color("84"))       // Green

	// Distribute user stories to appropriate columns
	for _, result := range parseResults {
		projectName := result.PRD.Name
		for _, story := range result.PRD.UserStories {
			card := &Card{
				ProjectName: projectName,
				Story:       story,
			}

			switch story.Status {
			case model.StatusInProgress:
				inProgressCol.AddCard(card)
			case model.StatusComplete:
				completeCol.AddCard(card)
			default:
				// Default to incomplete for unknown statuses
				incompleteCol.AddCard(card)
			}
		}
	}

	return &Board{
		columns: []*Column{incompleteCol, inProgressCol, completeCol},
	}
}

// SetSize updates the board dimensions and distributes width to columns.
func (b *Board) SetSize(width, height int) {
	b.width = width
	b.height = height

	// Reserve space for header and footer (about 4 lines)
	availableHeight := height - 4
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Distribute width evenly across columns with some padding
	columnWidth := (width - 4) / 3 // 4 for margins between columns
	if columnWidth < 20 {
		columnWidth = 20
	}

	for _, col := range b.columns {
		col.SetSize(columnWidth, availableHeight)
	}
}

// View renders the board with all three columns side by side.
func (b *Board) View() string {
	if b.width == 0 {
		return ""
	}

	// Render each column
	views := make([]string, len(b.columns))
	for i, col := range b.columns {
		views[i] = col.View()
	}

	// Join columns horizontally with a small gap
	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}

// Card represents a user story card on the board.
type Card struct {
	ProjectName string
	Story       model.UserStory
}
