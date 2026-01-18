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

// CardPosition represents the location of a card on the board.
type CardPosition struct {
	ProjectName string
	StoryID     string
	Status      string // The status/column where the card is located
}

// GetCardPositions returns the current position (status) of all cards on the board.
// The map key is "projectName:storyID" for efficient lookup.
func (b *Board) GetCardPositions() map[string]string {
	positions := make(map[string]string)
	statusNames := []string{"incomplete", "in-progress", "complete"}

	for i, col := range b.columns {
		status := statusNames[i]
		for _, card := range col.cards {
			key := card.ProjectName + ":" + card.Story.ID
			positions[key] = status
		}
	}
	return positions
}

// ApplyAnimations marks cards as animating if they moved from a different column.
// oldPositions is the result of GetCardPositions() from before the rebuild.
func (b *Board) ApplyAnimations(oldPositions map[string]string) {
	statusNames := []string{"incomplete", "in-progress", "complete"}

	for i, col := range b.columns {
		currentStatus := statusNames[i]
		for _, card := range col.cards {
			key := card.ProjectName + ":" + card.Story.ID
			if oldStatus, exists := oldPositions[key]; exists {
				// Card existed before - check if it moved
				if oldStatus != currentStatus {
					card.StartAnimation()
				}
			}
			// New cards don't animate - they just appear
		}
	}
}

// HasAnimatingCards returns true if any card on the board is currently animating.
func (b *Board) HasAnimatingCards() bool {
	for _, col := range b.columns {
		for _, card := range col.cards {
			if card.IsAnimating() {
				return true
			}
		}
	}
	return false
}

// ClearExpiredAnimations stops animations on cards whose animation duration has passed.
func (b *Board) ClearExpiredAnimations() {
	for _, col := range b.columns {
		for _, card := range col.cards {
			if card.Animating && !card.IsAnimating() {
				card.StopAnimation()
			}
		}
	}
}

// AllProjectsFilter is imported from filter.go - use this constant value for "no filter"
const allProjectsFilterValue = "All Projects"

// NewBoard creates a new Board from the parsed PRD results with no filter.
func NewBoard(parseResults []*parser.ParseResult) *Board {
	return NewBoardWithFilter(parseResults, allProjectsFilterValue)
}

// NewBoardWithFilter creates a new Board from the parsed PRD results with an optional project filter.
// If projectFilter is empty or "All Projects", all projects are shown.
// Cards are sorted by modification time (most recently changed first).
func NewBoardWithFilter(parseResults []*parser.ParseResult, projectFilter string) *Board {
	// Create the three columns
	incompleteCol := NewColumn("Incomplete", lipgloss.Color("203"))  // Red-ish
	inProgressCol := NewColumn("In Progress", lipgloss.Color("220")) // Yellow
	completeCol := NewColumn("Complete", lipgloss.Color("84"))       // Green

	// Distribute user stories to appropriate columns
	for _, result := range parseResults {
		projectName := result.PRD.Name

		// Skip projects that don't match the filter
		if projectFilter != "" && projectFilter != allProjectsFilterValue && projectName != projectFilter {
			continue
		}

		for _, story := range result.PRD.UserStories {
			// Use modification time from the parse result for sorting
			card := NewCardWithModTime(projectName, story, result.ModTime)

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

	// Sort cards within each column by modification time (most recently changed first)
	incompleteCol.SortByModTime()
	inProgressCol.SortByModTime()
	// Complete column uses updatedAt field (if present) for sorting, falls back to ModTime
	completeCol.SortByUpdatedAt()

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

	// Reserve space for header, footer, and padding:
	// - ASCII art header: ASCIIArtLines (3) + TopPadding + 1 margin = 5 lines (when width >= MinWidthForASCIIArt)
	// - Simple header: 1 line + TopPadding + 1 margin = 3 lines (narrow terminals)
	// - Footer: 2 lines (status bar + help message) + 1 margin = 3 lines
	// - Bottom padding: BottomPadding lines (to match top padding)
	// Total reserved for wide: 3 + TopPadding + 1 + 3 + BottomPadding = 9 lines
	// Total reserved for narrow: 1 + TopPadding + 1 + 3 + BottomPadding = 7 lines
	var reservedLines int
	if width >= MinWidthForASCIIArt {
		reservedLines = ASCIIArtLines + TopPadding + 1 + 3 + BottomPadding // ASCII art: 9 lines
	} else {
		reservedLines = 1 + TopPadding + 1 + 3 + BottomPadding // Simple header: 7 lines
	}
	availableHeight := height - reservedLines
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Distribute width evenly across columns with equal padding on both sides
	// Reserve: 4 for margins between columns + HorizontalPadding*2 for left and right padding
	reservedWidth := 4 + (HorizontalPadding * 2)
	columnWidth := (width - reservedWidth) / 3
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
