// column.go provides the column component for the Kanban board.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Column represents a single column in the Kanban board.
type Column struct {
	title        string
	cards        []*Card
	color        lipgloss.Color
	width        int
	height       int
	isSelected   bool // Whether this column is currently selected
	selectedCard int  // Index of selected card (-1 if none)
}

// NewColumn creates a new column with the given title and color.
func NewColumn(title string, color lipgloss.Color) *Column {
	return &Column{
		title:        title,
		cards:        make([]*Card, 0),
		color:        color,
		isSelected:   false,
		selectedCard: -1,
	}
}

// SetSelected sets the selection state of this column and which card is selected.
func (c *Column) SetSelected(isSelected bool, cardIndex int) {
	c.isSelected = isSelected
	if isSelected {
		c.selectedCard = cardIndex
	} else {
		c.selectedCard = -1
	}
}

// AddCard adds a card to this column.
func (c *Column) AddCard(card *Card) {
	c.cards = append(c.cards, card)
}

// SortByPriority sorts cards by priority (lower priority values first).
// This ensures cards with priority 0 appear at the top, followed by 1, 2, etc.
func (c *Column) SortByPriority() {
	sort.Slice(c.cards, func(i, j int) bool {
		return c.cards[i].Story.Priority < c.cards[j].Story.Priority
	})
}

// SetSize updates the column dimensions.
func (c *Column) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// View renders the column with its title and cards.
func (c *Column) View() string {
	// Column container style
	containerStyle := lipgloss.NewStyle().
		Width(c.width).
		Height(c.height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(c.color).
		Padding(0, 1)

	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(c.color).
		Width(c.width - 4).
		Align(lipgloss.Center).
		MarginBottom(1)

	// Build the column content
	var content strings.Builder
	content.WriteString(titleStyle.Render(c.title))
	content.WriteString("\n")

	// Card count indicator
	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Align(lipgloss.Center).
		Width(c.width - 4)

	content.WriteString(countStyle.Render(fmt.Sprintf("(%d items)", len(c.cards))))
	content.WriteString("\n\n")

	if len(c.cards) == 0 {
		// Empty column placeholder
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Width(c.width - 4).
			Align(lipgloss.Center)

		content.WriteString(emptyStyle.Render("No items"))
	} else {
		// Render cards using the Card component
		cardWidth := c.width - 6
		if cardWidth < 15 {
			cardWidth = 15
		}

		for i, card := range c.cards {
			// Check if this card is selected
			isSelected := c.isSelected && i == c.selectedCard
			content.WriteString(card.RenderSelected(cardWidth, isSelected))
			if i < len(c.cards)-1 {
				content.WriteString("\n")
			}
		}
	}

	return containerStyle.Render(content.String())
}
