// filter.go provides the project filter overlay component for PRDMonitor.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AllProjectsFilter is the special value that shows all projects.
const AllProjectsFilter = "All Projects"

// FilterMode determines how selected projects are filtered.
type FilterMode int

const (
	// FilterModeInclude shows only selected projects.
	FilterModeInclude FilterMode = iota
	// FilterModeExclude shows all projects except selected ones.
	FilterModeExclude
)

// FilterState represents the current filter configuration.
type FilterState struct {
	SelectedProjects map[string]bool // Projects that are selected (checked)
	Mode             FilterMode      // Include or exclude mode
}

// NewFilterState creates a new empty filter state in include mode.
func NewFilterState() *FilterState {
	return &FilterState{
		SelectedProjects: make(map[string]bool),
		Mode:             FilterModeInclude,
	}
}

// IsFiltering returns true if any filtering is active.
func (fs *FilterState) IsFiltering() bool {
	return len(fs.SelectedProjects) > 0
}

// ShouldShowProject returns true if the project should be displayed based on current filter.
func (fs *FilterState) ShouldShowProject(projectName string) bool {
	if !fs.IsFiltering() {
		return true // No filter active, show all
	}

	isSelected := fs.SelectedProjects[projectName]

	if fs.Mode == FilterModeInclude {
		return isSelected // Only show if selected
	}
	// Exclude mode: show if NOT selected
	return !isSelected
}

// ToggleProject toggles the selection state of a project.
func (fs *FilterState) ToggleProject(projectName string) {
	if fs.SelectedProjects[projectName] {
		delete(fs.SelectedProjects, projectName)
	} else {
		fs.SelectedProjects[projectName] = true
	}
}

// Clear removes all selected projects.
func (fs *FilterState) Clear() {
	fs.SelectedProjects = make(map[string]bool)
}

// ToggleMode switches between include and exclude mode.
func (fs *FilterState) ToggleMode() {
	if fs.Mode == FilterModeInclude {
		fs.Mode = FilterModeExclude
	} else {
		fs.Mode = FilterModeInclude
	}
}

// FilterOverlay represents the project filter popup.
type FilterOverlay struct {
	width         int
	height        int
	projects      []string     // List of project names (actual projects only, no "All Projects")
	cursorIndex   int          // Currently highlighted project index
	filterState   *FilterState // Current filter configuration
}

// NewFilterOverlay creates a new filter overlay with the given project names.
func NewFilterOverlay(projectNames []string) *FilterOverlay {
	return &FilterOverlay{
		projects:    projectNames,
		cursorIndex: 0,
		filterState: NewFilterState(),
	}
}

// SetSize sets the dimensions for the filter overlay.
func (f *FilterOverlay) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// UpdateProjects updates the list of available projects.
func (f *FilterOverlay) UpdateProjects(projectNames []string) {
	f.projects = projectNames

	// Clamp cursor to valid range
	if f.cursorIndex >= len(f.projects) {
		f.cursorIndex = len(f.projects) - 1
	}
	if f.cursorIndex < 0 {
		f.cursorIndex = 0
	}

	// Remove any selected projects that no longer exist
	validProjects := make(map[string]bool)
	for _, p := range projectNames {
		validProjects[p] = true
	}
	for p := range f.filterState.SelectedProjects {
		if !validProjects[p] {
			delete(f.filterState.SelectedProjects, p)
		}
	}
}

// Projects returns the list of available projects.
func (f *FilterOverlay) Projects() []string {
	return f.projects
}

// SelectedIndex returns the currently selected index.
// Deprecated: Use CursorIndex instead.
func (f *FilterOverlay) SelectedIndex() int {
	return f.cursorIndex
}

// CursorIndex returns the current cursor position.
func (f *FilterOverlay) CursorIndex() int {
	return f.cursorIndex
}

// SelectedProject returns the currently selected project name.
// Deprecated: Use FilterState for multi-select support.
func (f *FilterOverlay) SelectedProject() string {
	// For backwards compatibility: if exactly one project is selected in include mode,
	// return it. Otherwise return AllProjectsFilter.
	if f.filterState.Mode == FilterModeInclude && len(f.filterState.SelectedProjects) == 1 {
		for p := range f.filterState.SelectedProjects {
			return p
		}
	}
	return AllProjectsFilter
}

// FilterState returns the current filter state.
func (f *FilterOverlay) FilterState() *FilterState {
	return f.filterState
}

// MoveUp moves the cursor up.
func (f *FilterOverlay) MoveUp() {
	if f.cursorIndex > 0 {
		f.cursorIndex--
	}
}

// MoveDown moves the cursor down.
func (f *FilterOverlay) MoveDown() {
	if f.cursorIndex < len(f.projects)-1 {
		f.cursorIndex++
	}
}

// ToggleCurrentProject toggles the selection of the project at the cursor.
func (f *FilterOverlay) ToggleCurrentProject() {
	if f.cursorIndex >= 0 && f.cursorIndex < len(f.projects) {
		f.filterState.ToggleProject(f.projects[f.cursorIndex])
	}
}

// ToggleMode switches between include and exclude mode.
func (f *FilterOverlay) ToggleMode() {
	f.filterState.ToggleMode()
}

// ClearSelection clears all selected projects.
func (f *FilterOverlay) ClearSelection() {
	f.filterState.Clear()
}

// SelectProject sets the selection to the given project name.
// Returns true if the project was found and selected.
// Deprecated: Use ToggleCurrentProject for multi-select.
func (f *FilterOverlay) SelectProject(projectName string) bool {
	for i, p := range f.projects {
		if p == projectName {
			f.cursorIndex = i
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

	// Mode indicator style
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Align(lipgloss.Center).
		Width(overlayWidth - 4)

	// Normal item style
	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Width(overlayWidth - 6)

	// Selected item style (cursor is on this item)
	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("39")).
		Bold(true).
		Width(overlayWidth - 6)

	// Checked indicator
	checkedBox := "[x]"
	uncheckedBox := "[ ]"

	// Build content
	var content strings.Builder
	content.WriteString(titleStyle.Render("Filter Projects"))
	content.WriteString("\n")

	// Mode indicator
	modeText := "Mode: INCLUDE selected"
	if f.filterState.Mode == FilterModeExclude {
		modeText = "Mode: EXCLUDE selected"
	}
	content.WriteString(modeStyle.Render(modeText))
	content.WriteString("\n\n")

	// Render project list with checkboxes
	for i, project := range f.projects {
		displayName := project
		if len(displayName) > overlayWidth-12 {
			displayName = displayName[:overlayWidth-15] + "..."
		}

		// Determine checkbox state
		checkbox := uncheckedBox
		if f.filterState.SelectedProjects[project] {
			checkbox = checkedBox
		}

		line := checkbox + " " + displayName

		if i == f.cursorIndex {
			content.WriteString("  ")
			content.WriteString(cursorStyle.Render(" " + line + " "))
		} else {
			content.WriteString("  ")
			content.WriteString(itemStyle.Render(" " + line))
		}
		content.WriteString("\n")
	}

	// Show count of selected projects
	content.WriteString("\n")
	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Align(lipgloss.Center).
		Width(overlayWidth - 4)
	selectedCount := len(f.filterState.SelectedProjects)
	if selectedCount == 0 {
		content.WriteString(countStyle.Render("No projects selected (showing all)"))
	} else {
		countText := "1 project selected"
		if selectedCount > 1 {
			countText = strings.Replace(countText, "1", string(rune('0'+selectedCount)), 1)
			if selectedCount >= 10 {
				countText = lipgloss.NewStyle().Render(string(rune('0'+selectedCount/10)) + string(rune('0'+selectedCount%10)) + " projects selected")
			}
			countText = strings.Replace(countText, "project", "projects", 1)
		}
		content.WriteString(countStyle.Render(countText))
	}

	content.WriteString("\n\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true).
		Align(lipgloss.Center).
		Width(overlayWidth - 4)
	content.WriteString(footerStyle.Render("Space: toggle | Tab: mode | c: clear | Esc: close"))

	// Container style with border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(overlayWidth)

	return containerStyle.Render(content.String())
}
