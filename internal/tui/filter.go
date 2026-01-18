// filter.go provides the project filter overlay component for PRDMonitor.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AllProjectsFilter is the special value that shows all projects.
const AllProjectsFilter = "All Projects"

// FilterOverlay represents the project filter popup.
type FilterOverlay struct {
	width         int
	height        int
	projects      []string // List of project names (first is always "All Projects")
	selectedIndex int      // Currently highlighted project index
}

// NewFilterOverlay creates a new filter overlay with the given project names.
func NewFilterOverlay(projectNames []string) *FilterOverlay {
	// Build list with "All Projects" at the top
	projects := []string{AllProjectsFilter}
	projects = append(projects, projectNames...)

	return &FilterOverlay{
		projects:      projects,
		selectedIndex: 0,
	}
}

// SetSize sets the dimensions for the filter overlay.
func (f *FilterOverlay) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// UpdateProjects updates the list of available projects.
func (f *FilterOverlay) UpdateProjects(projectNames []string) {
	// Preserve selection if possible
	currentSelection := ""
	if f.selectedIndex >= 0 && f.selectedIndex < len(f.projects) {
		currentSelection = f.projects[f.selectedIndex]
	}

	// Rebuild list
	f.projects = []string{AllProjectsFilter}
	f.projects = append(f.projects, projectNames...)

	// Try to restore selection
	f.selectedIndex = 0
	for i, p := range f.projects {
		if p == currentSelection {
			f.selectedIndex = i
			break
		}
	}
}

// Projects returns the list of available projects.
func (f *FilterOverlay) Projects() []string {
	return f.projects
}

// SelectedIndex returns the currently selected index.
func (f *FilterOverlay) SelectedIndex() int {
	return f.selectedIndex
}

// SelectedProject returns the currently selected project name.
func (f *FilterOverlay) SelectedProject() string {
	if f.selectedIndex >= 0 && f.selectedIndex < len(f.projects) {
		return f.projects[f.selectedIndex]
	}
	return AllProjectsFilter
}

// MoveUp moves the selection up.
func (f *FilterOverlay) MoveUp() {
	if f.selectedIndex > 0 {
		f.selectedIndex--
	}
}

// MoveDown moves the selection down.
func (f *FilterOverlay) MoveDown() {
	if f.selectedIndex < len(f.projects)-1 {
		f.selectedIndex++
	}
}

// SelectProject sets the selection to the given project name.
// Returns true if the project was found and selected.
func (f *FilterOverlay) SelectProject(projectName string) bool {
	for i, p := range f.projects {
		if p == projectName {
			f.selectedIndex = i
			return true
		}
	}
	return false
}

// View renders the filter overlay.
func (f *FilterOverlay) View() string {
	// Calculate overlay dimensions
	overlayWidth := 50
	if f.width > 0 && overlayWidth > f.width-10 {
		overlayWidth = f.width - 10
	}
	if overlayWidth < 30 {
		overlayWidth = 30
	}

	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Align(lipgloss.Center).
		Width(overlayWidth - 4)

	// Normal item style
	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Width(overlayWidth - 6)

	// Selected item style
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("39")).
		Bold(true).
		Width(overlayWidth - 6)

	// Build content
	var content strings.Builder
	content.WriteString(titleStyle.Render("Filter by Project"))
	content.WriteString("\n\n")

	// Render project list
	for i, project := range f.projects {
		displayName := project
		if len(displayName) > overlayWidth-8 {
			displayName = displayName[:overlayWidth-11] + "..."
		}

		if i == f.selectedIndex {
			content.WriteString("  ")
			content.WriteString(selectedStyle.Render(" " + displayName + " "))
		} else {
			content.WriteString("  ")
			content.WriteString(itemStyle.Render(" " + displayName))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true).
		Align(lipgloss.Center).
		Width(overlayWidth - 4)
	content.WriteString(footerStyle.Render("↑↓: navigate | Enter: select | Esc/f: close"))

	// Container style with border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(overlayWidth)

	return containerStyle.Render(content.String())
}
