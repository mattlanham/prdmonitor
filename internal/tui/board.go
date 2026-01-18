// board.go provides the Kanban board view with three columns.
package tui

import (
	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/model"
	"lanham/prdmonitor/internal/parser"
)

// Board represents the Kanban board with three columns.
type Board struct {
	columns      []*Column
	width        int
	height       int
	selectedCol  int // Currently selected column index (0-2)
	selectedCard int // Currently selected card index within the column
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
			card := NewCard(projectName, story)

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

	// Sort cards within each column by priority (lower priority values first)
	incompleteCol.SortByPriority()
	inProgressCol.SortByPriority()
	completeCol.SortByPriority()

	return &Board{
		columns:      []*Column{incompleteCol, inProgressCol, completeCol},
		selectedCol:  0,
		selectedCard: 0,
	}
}

// MoveLeft moves the selection to the previous column.
func (b *Board) MoveLeft() {
	if b.selectedCol > 0 {
		b.selectedCol--
		// Clamp selected card to valid range for new column
		b.clampSelectedCard()
	}
}

// MoveRight moves the selection to the next column.
func (b *Board) MoveRight() {
	if b.selectedCol < len(b.columns)-1 {
		b.selectedCol++
		// Clamp selected card to valid range for new column
		b.clampSelectedCard()
	}
}

// MoveUp moves the selection to the previous card in the current column.
func (b *Board) MoveUp() {
	if b.selectedCard > 0 {
		b.selectedCard--
		// Ensure the selected card is visible by triggering scroll adjustment
		b.ensureSelectedCardVisible()
	}
}

// MoveDown moves the selection to the next card in the current column.
func (b *Board) MoveDown() {
	col := b.columns[b.selectedCol]
	if b.selectedCard < len(col.cards)-1 {
		b.selectedCard++
		// Ensure the selected card is visible by triggering scroll adjustment
		b.ensureSelectedCardVisible()
	}
}

// ensureSelectedCardVisible ensures the currently selected card is visible
// by adjusting the scroll offset of the current column.
func (b *Board) ensureSelectedCardVisible() {
	if b.selectedCol >= 0 && b.selectedCol < len(b.columns) {
		b.columns[b.selectedCol].ensureCardVisible(b.selectedCard)
	}
}

// CycleColumn cycles to the next column (wraps around).
func (b *Board) CycleColumn() {
	b.selectedCol = (b.selectedCol + 1) % len(b.columns)
	b.clampSelectedCard()
}

// clampSelectedCard ensures selectedCard is within valid range for current column.
func (b *Board) clampSelectedCard() {
	col := b.columns[b.selectedCol]
	if len(col.cards) == 0 {
		b.selectedCard = 0
	} else if b.selectedCard >= len(col.cards) {
		b.selectedCard = len(col.cards) - 1
	}
}

// SelectedCard returns the currently selected card, or nil if none.
func (b *Board) SelectedCard() *Card {
	if b.selectedCol < 0 || b.selectedCol >= len(b.columns) {
		return nil
	}
	col := b.columns[b.selectedCol]
	if b.selectedCard < 0 || b.selectedCard >= len(col.cards) {
		return nil
	}
	return col.cards[b.selectedCard]
}

// SelectedColumn returns the index of the currently selected column.
func (b *Board) SelectedColumn() int {
	return b.selectedCol
}

// SelectedCardIndex returns the index of the currently selected card.
func (b *Board) SelectedCardIndex() int {
	return b.selectedCard
}

// ScrollColumnUp scrolls the specified column up by one card.
func (b *Board) ScrollColumnUp(colIndex int) {
	if colIndex >= 0 && colIndex < len(b.columns) {
		b.columns[colIndex].ScrollUp()
	}
}

// ScrollColumnDown scrolls the specified column down by one card.
func (b *Board) ScrollColumnDown(colIndex int) {
	if colIndex >= 0 && colIndex < len(b.columns) {
		b.columns[colIndex].ScrollDown()
	}
}

// SetSize updates the board dimensions and distributes width to columns.
func (b *Board) SetSize(width, height int) {
	b.width = width
	b.height = height

	// Reserve space for header (with top padding), margin, and footer (about 5 lines)
	availableHeight := height - 5
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

	// Set selection state on columns before rendering
	for i, col := range b.columns {
		col.SetSelected(i == b.selectedCol, b.selectedCard)
	}

	// Render each column
	views := make([]string, len(b.columns))
	for i, col := range b.columns {
		views[i] = col.View()
	}

	// Join columns horizontally with a small gap
	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}
