// column.go provides the column component for the Kanban board.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Column represents a single column in the Kanban board.
type Column struct {
	title  string
	cards  []*Card
	color  lipgloss.Color
	width  int
	height int
}

// NewColumn creates a new column with the given title and color.
func NewColumn(title string, color lipgloss.Color) *Column {
	return &Column{
		title: title,
		cards: make([]*Card, 0),
		color: color,
	}
}

// AddCard adds a card to this column.
func (c *Column) AddCard(card *Card) {
	c.cards = append(c.cards, card)
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
		// Render cards
		cardWidth := c.width - 6
		if cardWidth < 15 {
			cardWidth = 15
		}

		for i, card := range c.cards {
			content.WriteString(renderCard(card, cardWidth))
			if i < len(c.cards)-1 {
				content.WriteString("\n")
			}
		}
	}

	return containerStyle.Render(content.String())
}

// renderCard renders a single card with box-drawing borders.
func renderCard(card *Card, width int) string {
	// Card border style
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(width).
		Padding(0, 1)

	// Project name style
	projectStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	// ID style
	idStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	// Title style
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true)

	// Description style
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("247"))

	// Build card content
	var content strings.Builder

	// Project name
	content.WriteString(projectStyle.Render(truncate(card.ProjectName, width-4)))
	content.WriteString("\n")

	// Story ID
	content.WriteString(idStyle.Render(card.Story.ID))
	content.WriteString("\n")

	// Title
	content.WriteString(titleStyle.Render(truncate(card.Story.Title, width-4)))
	content.WriteString("\n")

	// Description (truncated)
	desc := truncate(card.Story.Description, width-4)
	if len(card.Story.Description) > width-4 {
		desc = desc + "..."
	}
	content.WriteString(descStyle.Render(desc))

	return cardStyle.Render(content.String())
}

// truncate shortens a string to the given max length.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
