// help.go provides the help overlay component for PRDMonitor.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpOverlay represents the help popup that displays keyboard shortcuts.
type HelpOverlay struct {
	width  int
	height int
}

// NewHelpOverlay creates a new help overlay.
func NewHelpOverlay() *HelpOverlay {
	return &HelpOverlay{}
}

// SetSize sets the dimensions for the help overlay.
func (h *HelpOverlay) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// HelpShortcuts contains all the keyboard shortcuts to display.
var HelpShortcuts = []struct {
	Key         string
	Description string
}{
	{"↑/k", "Move up"},
	{"↓/j", "Move down"},
	{"←/h", "Move to left column"},
	{"→/l", "Move to right column"},
	{"Tab", "Cycle between columns"},
	{"Enter/Space", "Expand card details"},
	{"f", "Filter by project"},
	{"Esc", "Close overlay/filter"},
	{"?", "Toggle this help"},
	{"q/Ctrl+C", "Quit application"},
}

// View renders the help overlay.
func (h *HelpOverlay) View() string {
	// Calculate overlay dimensions
	overlayWidth := 45
	if h.width > 0 && overlayWidth > h.width-10 {
		overlayWidth = h.width - 10
	}

	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Align(lipgloss.Center).
		Width(overlayWidth - 4)

	// Shortcut key style
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Width(14)

	// Description style
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	// Build content
	var content strings.Builder
	content.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	content.WriteString("\n\n")

	for _, shortcut := range HelpShortcuts {
		content.WriteString(keyStyle.Render(shortcut.Key))
		content.WriteString(descStyle.Render(shortcut.Description))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true).
		Align(lipgloss.Center).
		Width(overlayWidth - 4)
	content.WriteString(footerStyle.Render("Press ? or Esc to close"))

	// Container style with border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(overlayWidth)

	return containerStyle.Render(content.String())
}
