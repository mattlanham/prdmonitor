// card.go provides the user story card component for the Kanban board.
package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/model"
)

// Card represents a user story card on the board.
type Card struct {
	ProjectName string
	Story       model.UserStory
	ModTime     time.Time // Last modification time of the source prd.json file
	Animating   bool      // Whether this card is currently animating (just moved columns)
	AnimStart   time.Time // When the animation started
}

// CardStyle holds the styling configuration for a card.
type CardStyle struct {
	// Border style for the card container
	BorderColor lipgloss.Color
	// Project name color (appears at top)
	ProjectColor lipgloss.Color
	// Story ID color (prominent display)
	IDColor lipgloss.Color
	// Title color (bold/highlighted)
	TitleColor lipgloss.Color
	// Description color (subdued)
	DescriptionColor lipgloss.Color
}

// DefaultCardStyle returns the default styling for cards.
func DefaultCardStyle() CardStyle {
	return CardStyle{
		BorderColor:      lipgloss.Color("240"), // Gray border
		ProjectColor:     lipgloss.Color("39"),  // Cyan for project name
		IDColor:          lipgloss.Color("205"), // Pink for ID (prominent)
		TitleColor:       lipgloss.Color("255"), // White/bright for title
		DescriptionColor: lipgloss.Color("247"), // Light gray for description
	}
}

// AnimatingCardStyle returns the styling for a card that is animating (just moved columns).
// Uses a bright highlight border to draw attention to the card movement.
func AnimatingCardStyle() CardStyle {
	return CardStyle{
		BorderColor:      lipgloss.Color("220"), // Yellow/gold border for animation highlight
		ProjectColor:     lipgloss.Color("39"),  // Cyan for project name
		IDColor:          lipgloss.Color("205"), // Pink for ID (prominent)
		TitleColor:       lipgloss.Color("255"), // White/bright for title
		DescriptionColor: lipgloss.Color("247"), // Light gray for description
	}
}

// AnimationDuration is how long card movement animations last.
const AnimationDuration = 800 * time.Millisecond

// IsAnimating returns true if the card is currently animating.
func (c *Card) IsAnimating() bool {
	if !c.Animating {
		return false
	}
	// Check if animation has expired
	return time.Since(c.AnimStart) < AnimationDuration
}

// StartAnimation marks this card as animating (e.g., when it moves between columns).
func (c *Card) StartAnimation() {
	c.Animating = true
	c.AnimStart = time.Now()
}

// StopAnimation clears the animation state.
func (c *Card) StopAnimation() {
	c.Animating = false
	c.AnimStart = time.Time{}
}

// Render renders the card with the given width.
func (c *Card) Render(width int) string {
	return c.RenderWithStyle(width, DefaultCardStyle())
}

// RenderSelected renders the card with selection highlighting or animation.
func (c *Card) RenderSelected(width int, isSelected bool) string {
	style := DefaultCardStyle()
	if c.IsAnimating() {
		style = AnimatingCardStyle()
	} else if isSelected {
		style = SelectedCardStyle()
	}
	return c.RenderWithStyle(width, style)
}

// SelectedCardStyle returns the styling for a selected card.
func SelectedCardStyle() CardStyle {
	return CardStyle{
		BorderColor:      lipgloss.Color("39"),  // Cyan border for selection
		ProjectColor:     lipgloss.Color("39"),  // Cyan for project name
		IDColor:          lipgloss.Color("205"), // Pink for ID (prominent)
		TitleColor:       lipgloss.Color("255"), // White/bright for title
		DescriptionColor: lipgloss.Color("247"), // Light gray for description
	}
}

// RenderWithStyle renders the card with custom styling.
func (c *Card) RenderWithStyle(width int, style CardStyle) string {
	// Ensure minimum width for readable content
	if width < 10 {
		width = 10
	}

	// Calculate content width (accounting for border and padding)
	contentWidth := width - 4 // 2 for border, 2 for padding
	if contentWidth < 6 {
		contentWidth = 6
	}

	// Card container style with box-drawing borders
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.BorderColor).
		Width(width).
		Padding(0, 1)

	// Project name style - appears at top of card
	projectStyle := lipgloss.NewStyle().
		Foreground(style.ProjectColor).
		Bold(true)

	// ID style - prominent display
	idStyle := lipgloss.NewStyle().
		Foreground(style.IDColor).
		Bold(true)

	// Title style - bold/highlighted
	titleStyle := lipgloss.NewStyle().
		Foreground(style.TitleColor).
		Bold(true)

	// Description style - subdued
	descStyle := lipgloss.NewStyle().
		Foreground(style.DescriptionColor)

	// Build card content
	var content strings.Builder

	// Line 1: Project name (at top of card)
	projectName := truncateString(c.ProjectName, contentWidth)
	content.WriteString(projectStyle.Render(projectName))
	content.WriteString("\n")

	// Line 2: Story ID (prominently displayed)
	storyID := truncateString(c.Story.ID, contentWidth)
	content.WriteString(idStyle.Render(storyID))
	content.WriteString("\n")

	// Line 3: Title (bold/highlighted)
	title := truncateString(c.Story.Title, contentWidth)
	content.WriteString(titleStyle.Render(title))

	// Line 4: Description (truncated if needed)
	if c.Story.Description != "" {
		content.WriteString("\n")
		desc := c.Story.Description
		// Truncate description and add ellipsis if too long
		if len(desc) > contentWidth {
			desc = truncateString(desc, contentWidth)
		}
		content.WriteString(descStyle.Render(desc))
	}

	return cardStyle.Render(content.String())
}

// truncateString shortens a string to the given max length with ellipsis.
func truncateString(s string, maxLen int) string {
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

// NewCard creates a new Card with the given project name and user story.
// Deprecated: Use NewCardWithModTime instead to include modification time for sorting.
func NewCard(projectName string, story model.UserStory) *Card {
	return &Card{
		ProjectName: projectName,
		Story:       story,
		ModTime:     time.Time{}, // Zero time when not specified
	}
}

// NewCardWithModTime creates a new Card with the given project name, user story, and modification time.
func NewCardWithModTime(projectName string, story model.UserStory, modTime time.Time) *Card {
	return &Card{
		ProjectName: projectName,
		Story:       story,
		ModTime:     modTime,
	}
}

// RenderExpanded renders the card in expanded mode showing full details.
func (c *Card) RenderExpanded(width int) string {
	if width < 30 {
		width = 30
	}

	style := DefaultCardStyle()

	// Calculate content width (accounting for border and padding)
	contentWidth := width - 6 // 2 for border, 4 for padding
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Card container style with box-drawing borders
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")). // Cyan border for expanded view
		Width(width).
		Padding(1, 2)

	// Project name style
	projectStyle := lipgloss.NewStyle().
		Foreground(style.ProjectColor).
		Bold(true)

	// ID style
	idStyle := lipgloss.NewStyle().
		Foreground(style.IDColor).
		Bold(true)

	// Title style
	titleStyle := lipgloss.NewStyle().
		Foreground(style.TitleColor).
		Bold(true)

	// Description style
	descStyle := lipgloss.NewStyle().
		Foreground(style.DescriptionColor)

	// Label style for sections
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	// Status style
	statusStyle := lipgloss.NewStyle()
	switch c.Story.Status {
	case "complete":
		statusStyle = statusStyle.Foreground(lipgloss.Color("84")) // Green
	case "in-progress":
		statusStyle = statusStyle.Foreground(lipgloss.Color("220")) // Yellow
	default:
		statusStyle = statusStyle.Foreground(lipgloss.Color("203")) // Red
	}

	// Build card content
	var content strings.Builder

	// Header: Project name and Story ID
	content.WriteString(projectStyle.Render(c.ProjectName))
	content.WriteString("\n")
	content.WriteString(idStyle.Render(c.Story.ID))
	content.WriteString(" • ")
	content.WriteString(statusStyle.Render(c.Story.Status))
	content.WriteString("\n\n")

	// Title
	content.WriteString(titleStyle.Render(c.Story.Title))
	content.WriteString("\n\n")

	// Description (full, wrapped)
	if c.Story.Description != "" {
		content.WriteString(labelStyle.Render("Description:"))
		content.WriteString("\n")
		content.WriteString(descStyle.Render(wrapText(c.Story.Description, contentWidth)))
		content.WriteString("\n\n")
	}

	// Acceptance Criteria
	if len(c.Story.AcceptanceCriteria) > 0 {
		content.WriteString(labelStyle.Render("Acceptance Criteria:"))
		content.WriteString("\n")
		for i, criterion := range c.Story.AcceptanceCriteria {
			bullet := descStyle.Render("• " + wrapText(criterion, contentWidth-2))
			content.WriteString(bullet)
			if i < len(c.Story.AcceptanceCriteria)-1 {
				content.WriteString("\n")
			}
		}
		content.WriteString("\n\n")
	}

	// Priority
	priorityStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("247"))
	content.WriteString(priorityStyle.Render("Priority: "))
	content.WriteString(priorityStyle.Render(strings.Repeat("★", min(c.Story.Priority+1, 5))))
	content.WriteString(priorityStyle.Render(" (" + itoa(c.Story.Priority) + ")"))

	// Footer hint
	content.WriteString("\n\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
	content.WriteString(footerStyle.Render("Press Esc or Enter to close"))

	return cardStyle.Render(content.String())
}

// wrapText wraps text to fit within the given width.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	if len(text) <= width {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		if i > 0 {
			if lineLen+1+len(word) > width {
				result.WriteString("\n")
				lineLen = 0
			} else {
				result.WriteString(" ")
				lineLen++
			}
		}
		result.WriteString(word)
		lineLen += len(word)
	}

	return result.String()
}

// itoa converts an int to string (simple helper to avoid importing strconv).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// min returns the minimum of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
