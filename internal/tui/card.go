// card.go provides the user story card component for the Kanban board.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/model"
)

// Card represents a user story card on the board.
type Card struct {
	ProjectName string
	Story       model.UserStory
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

// Render renders the card with the given width.
func (c *Card) Render(width int) string {
	return c.RenderWithStyle(width, DefaultCardStyle())
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
func NewCard(projectName string, story model.UserStory) *Card {
	return &Card{
		ProjectName: projectName,
		Story:       story,
	}
}
