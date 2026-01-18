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
	scrollOffset int  // Index of the first visible card (for scrolling)
}

// NewColumn creates a new column with the given title and color.
func NewColumn(title string, color lipgloss.Color) *Column {
	return &Column{
		title:        title,
		cards:        make([]*Card, 0),
		color:        color,
		isSelected:   false,
		selectedCard: -1,
		scrollOffset: 0,
	}
}

// SetSelected sets the selection state of this column and which card is selected.
// It also ensures the selected card is visible by adjusting the scroll offset.
func (c *Column) SetSelected(isSelected bool, cardIndex int) {
	c.isSelected = isSelected
	if isSelected {
		c.selectedCard = cardIndex
		// Ensure selected card is visible
		c.ensureCardVisible(cardIndex)
	} else {
		c.selectedCard = -1
	}
}

// ensureCardVisible adjusts the scroll offset to make the given card index visible.
func (c *Column) ensureCardVisible(cardIndex int) {
	if len(c.cards) == 0 || cardIndex < 0 {
		return
	}

	visibleCount := c.visibleCardCount()
	if visibleCount <= 0 {
		return
	}

	// If card is above the visible area, scroll up
	if cardIndex < c.scrollOffset {
		c.scrollOffset = cardIndex
	}

	// If card is below the visible area, scroll down
	if cardIndex >= c.scrollOffset+visibleCount {
		c.scrollOffset = cardIndex - visibleCount + 1
	}

	// Clamp scroll offset
	c.clampScrollOffset()
}

// visibleCardCount returns the estimated number of cards that can be visible at once.
// This is an approximation based on column height and estimated card height.
func (c *Column) visibleCardCount() int {
	if c.height <= 0 {
		return 1
	}
	// Approximate card height: 5 lines (border + content) + 1 line gap
	// Header takes about 4 lines (title + count + spacing)
	estimatedCardHeight := 6
	availableHeight := c.height - 6 // Reserve space for title, count, borders
	if availableHeight <= 0 {
		return 1
	}
	count := availableHeight / estimatedCardHeight
	if count < 1 {
		return 1
	}
	return count
}

// clampScrollOffset ensures scrollOffset is within valid bounds.
func (c *Column) clampScrollOffset() {
	if c.scrollOffset < 0 {
		c.scrollOffset = 0
	}
	maxOffset := c.maxScrollOffset()
	if c.scrollOffset > maxOffset {
		c.scrollOffset = maxOffset
	}
}

// maxScrollOffset returns the maximum valid scroll offset.
func (c *Column) maxScrollOffset() int {
	visibleCount := c.visibleCardCount()
	maxOffset := len(c.cards) - visibleCount
	if maxOffset < 0 {
		return 0
	}
	return maxOffset
}

// ScrollUp scrolls the column up by one card.
func (c *Column) ScrollUp() {
	if c.scrollOffset > 0 {
		c.scrollOffset--
	}
}

// ScrollDown scrolls the column down by one card.
func (c *Column) ScrollDown() {
	if c.scrollOffset < c.maxScrollOffset() {
		c.scrollOffset++
	}
}

// ScrollOffset returns the current scroll offset.
func (c *Column) ScrollOffset() int {
	return c.scrollOffset
}

// SetScrollOffset sets the scroll offset directly.
func (c *Column) SetScrollOffset(offset int) {
	c.scrollOffset = offset
	c.clampScrollOffset()
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

// SortByModTime sorts cards by modification time (most recently changed first).
// Cards with more recent modification times appear at the top of the column.
func (c *Column) SortByModTime() {
	sort.Slice(c.cards, func(i, j int) bool {
		// Most recent first (descending order)
		return c.cards[i].ModTime.After(c.cards[j].ModTime)
	})
}

// SortByUpdatedAt sorts cards by their updatedAt time (most recently updated first).
// If a card has an updatedAt field set, that time is used; otherwise falls back to ModTime.
// This is intended for the completed column to show most recently completed stories at top.
func (c *Column) SortByUpdatedAt() {
	sort.Slice(c.cards, func(i, j int) bool {
		// Use GetCompletedSortTime which prefers updatedAt over ModTime
		return c.cards[i].GetCompletedSortTime().After(c.cards[j].GetCompletedSortTime())
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

	// Card count indicator with scroll position
	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Align(lipgloss.Center).
		Width(c.width - 4)

	countText := fmt.Sprintf("(%d items)", len(c.cards))
	if len(c.cards) > 0 && c.maxScrollOffset() > 0 {
		// Show scroll position indicator when scrollable
		visibleEnd := c.scrollOffset + c.visibleCardCount()
		if visibleEnd > len(c.cards) {
			visibleEnd = len(c.cards)
		}
		countText = fmt.Sprintf("(%d-%d of %d)", c.scrollOffset+1, visibleEnd, len(c.cards))
	}
	content.WriteString(countStyle.Render(countText))
	content.WriteString("\n")

	// Show scroll up indicator if not at top
	scrollIndicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Align(lipgloss.Center).
		Width(c.width - 4)

	if c.scrollOffset > 0 {
		content.WriteString(scrollIndicatorStyle.Render("▲ more above"))
	}
	content.WriteString("\n")

	if len(c.cards) == 0 {
		// Empty column placeholder
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Width(c.width - 4).
			Align(lipgloss.Center)

		content.WriteString(emptyStyle.Render("No items"))
	} else {
		// Render only visible cards based on scroll offset
		cardWidth := c.width - 6
		if cardWidth < 15 {
			cardWidth = 15
		}

		visibleCount := c.visibleCardCount()
		endIndex := c.scrollOffset + visibleCount
		if endIndex > len(c.cards) {
			endIndex = len(c.cards)
		}

		for i := c.scrollOffset; i < endIndex; i++ {
			card := c.cards[i]
			// Check if this card is selected
			isSelected := c.isSelected && i == c.selectedCard
			content.WriteString(card.RenderSelected(cardWidth, isSelected))
			if i < endIndex-1 {
				content.WriteString("\n")
			}
		}

		// Show scroll down indicator if not at bottom
		if c.scrollOffset < c.maxScrollOffset() {
			content.WriteString("\n")
			content.WriteString(scrollIndicatorStyle.Render("▼ more below"))
		}
	}

	return containerStyle.Render(content.String())
}
