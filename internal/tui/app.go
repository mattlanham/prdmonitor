// Package tui provides the terminal user interface for PRDMonitor.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/parser"
	"lanham/prdmonitor/internal/watcher"
)

// ASCIIArtTitle is the ASCII art representation of "PRD Monitor".
// Uses a compact, legible style to leave more space for the Kanban board.
var ASCIIArtTitle = []string{
	"    ____  ____  ____     __  ___            _ __            ",
	"   / __ \\/ __ \\/ __ \\   /  |/  /___  ____  (_) /_____  _____",
	"  / /_/ / /_/ / / / /  / /|_/ / __ \\/ __ \\/ / __/ __ \\/ ___/",
	" / ____/ _, _/ /_/ /  / /  / / /_/ / / / / / /_/ /_/ / /    ",
	"/_/   /_/ |_/_____/  /_/  /_/\\____/_/ /_/_/\\__/\\____/_/     ",
	"                                                            ",
}

// ASCIIArtWidth is the visual width of the ASCII art title in rune characters.
// Note: Byte length varies due to Unicode box-drawing characters.
const ASCIIArtWidth = 30

// MinWidthForASCIIArt is the minimum terminal width to display the ASCII art.
// Below this width, a simple text header is shown instead.
const MinWidthForASCIIArt = 40

// Padding constants for equal spacing on all sides of the content area.
// These ensure a balanced and visually consistent layout.
const (
	HorizontalPadding = 2 // Left and right padding (equal)
	TopPadding        = 1 // Top padding (from header PaddingTop) - visible spacing above logo
	BottomPadding     = 1 // Bottom padding (equal to top for balance)
)

// ASCIIArtLines is the number of lines in the ASCII art logo.
// Used for height calculations in board.go.
const ASCIIArtLines = 7

// ReadOnlyMessage is displayed in the status bar to indicate view-only mode.
const ReadOnlyMessage = "VIEW-ONLY"

// EditHelpMessage is displayed to guide users on how to make changes.
const EditHelpMessage = "Edit prd.json files directly to make changes"

// FilterIndicatorPrefix is the prefix for the filter indicator badge.
const FilterIndicatorPrefix = "Filtered: "

// FileChangedMsg is sent when a watched file changes.
type FileChangedMsg struct {
	FilePath string
	Op       watcher.Operation
}

// App is the main Bubble Tea model for the PRDMonitor application.
type App struct {
	board         *Board
	width         int
	height        int
	watcher       *watcher.Watcher
	parseResults  []*parser.ParseResult
	rootDir       string
	showHelp      bool           // Whether to show the help overlay
	expandedCard  *Card          // Currently expanded card (nil if none)
	helpOverlay   *HelpOverlay   // Help overlay component
	showFilter    bool           // Whether to show the filter overlay
	filterOverlay *FilterOverlay // Filter overlay component
}

// NewApp creates a new App model with the given parsed PRD results.
func NewApp(parseResults []*parser.ParseResult, w *watcher.Watcher, rootDir string) *App {
	// Extract unique project names
	projectNames := extractProjectNames(parseResults)

	return &App{
		board:         NewBoard(parseResults),
		watcher:       w,
		parseResults:  parseResults,
		rootDir:       rootDir,
		showHelp:      false,
		expandedCard:  nil,
		helpOverlay:   NewHelpOverlay(),
		showFilter:    false,
		filterOverlay: NewFilterOverlay(projectNames),
	}
}

// extractProjectNames extracts unique project names from parse results.
func extractProjectNames(parseResults []*parser.ParseResult) []string {
	seen := make(map[string]bool)
	var names []string
	for _, result := range parseResults {
		name := result.PRD.Name
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// rebuildBoard rebuilds the board with the current filter applied.
// Returns true if any cards moved between columns (animations were triggered).
func (a *App) rebuildBoard() bool {
	// Capture current card positions before rebuild
	oldPositions := a.board.GetCardPositions()

	// Rebuild the board with current filter state
	a.board = NewBoardWithFilterState(a.parseResults, a.filterOverlay.FilterState())
	a.board.SetSize(a.width, a.height)

	// Apply animations to cards that moved columns
	a.board.ApplyAnimations(oldPositions)

	return a.board.HasAnimatingCards()
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	if a.watcher != nil {
		return a.listenForFileChanges()
	}
	return nil
}

// WatcherStoppedMsg is sent when the file watcher has been stopped.
type WatcherStoppedMsg struct{}

// AnimationTickMsg is sent periodically to update card animations.
type AnimationTickMsg time.Time

// animationTickCmd returns a command that sends an AnimationTickMsg after a short delay.
func animationTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return AnimationTickMsg(t)
	})
}

// listenForFileChanges returns a command that listens for file change events.
// It handles closed channels gracefully when the watcher is stopped.
func (a *App) listenForFileChanges() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-a.watcher.Events():
			if !ok {
				// Channel closed, watcher has been stopped
				return WatcherStoppedMsg{}
			}
			return FileChangedMsg{
				FilePath: event.FilePath,
				Op:       event.Op,
			}
		case _, ok := <-a.watcher.Errors():
			if !ok {
				// Channel closed, watcher has been stopped
				return WatcherStoppedMsg{}
			}
			// Ignore watcher errors for now, just continue listening
			return nil
		}
	}
}

// Update implements tea.Model.
// The board is read-only - cards cannot be moved or edited via keyboard or mouse.
// All edit-related keybindings are intentionally ignored.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle escape key to close overlays
		if msg.Type == tea.KeyEsc {
			if a.expandedCard != nil {
				a.expandedCard = nil
				return a, nil
			}
			if a.showFilter {
				a.showFilter = false
				return a, nil
			}
			if a.showHelp {
				a.showHelp = false
				return a, nil
			}
			return a, nil
		}

		// Handle help overlay toggle
		if msg.String() == "?" {
			if a.expandedCard != nil {
				a.expandedCard = nil
			}
			if a.showFilter {
				a.showFilter = false
			}
			a.showHelp = !a.showHelp
			return a, nil
		}

		// Handle filter overlay toggle
		if msg.String() == "f" {
			// Close other overlays first
			if a.expandedCard != nil {
				a.expandedCard = nil
			}
			if a.showHelp {
				a.showHelp = false
			}
			a.showFilter = !a.showFilter
			return a, nil
		}

		// If help is showing, only allow closing it
		if a.showHelp {
			return a, nil
		}

		// If filter is showing, handle filter navigation
		if a.showFilter {
			switch msg.Type {
			case tea.KeyEnter:
				// Close filter and apply
				a.showFilter = false
				a.rebuildBoard()
				return a, nil
			case tea.KeyUp:
				a.filterOverlay.MoveUp()
				return a, nil
			case tea.KeyDown:
				a.filterOverlay.MoveDown()
				return a, nil
			case tea.KeyTab:
				// Toggle between include/exclude mode
				a.filterOverlay.ToggleMode()
				return a, nil
			}
			switch msg.String() {
			case "k":
				a.filterOverlay.MoveUp()
				return a, nil
			case "j":
				a.filterOverlay.MoveDown()
				return a, nil
			case " ":
				// Toggle selection of current project
				a.filterOverlay.ToggleCurrentProject()
				return a, nil
			case "c":
				// Clear all selections
				a.filterOverlay.ClearSelection()
				return a, nil
			case "tab":
				// Toggle between include/exclude mode
				a.filterOverlay.ToggleMode()
				return a, nil
			}
			return a, nil
		}

		// If card is expanded, only allow closing it
		if a.expandedCard != nil {
			switch msg.Type {
			case tea.KeyEnter:
				a.expandedCard = nil
				return a, nil
			}
			switch msg.String() {
			case " ":
				a.expandedCard = nil
				return a, nil
			}
			return a, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit

		// Navigation: Arrow keys and hjkl
		case "up", "k":
			a.board.MoveUp()
			return a, nil
		case "down", "j":
			a.board.MoveDown()
			return a, nil
		case "left", "h":
			a.board.MoveLeft()
			return a, nil
		case "right", "l":
			a.board.MoveRight()
			return a, nil

		// Tab cycles between columns
		case "tab":
			a.board.CycleColumn()
			return a, nil

		// Enter or space expands the selected card
		case "enter", " ":
			selectedCard := a.board.SelectedCard()
			if selectedCard != nil {
				a.expandedCard = selectedCard
			}
			return a, nil

		// Explicitly ignore common edit keybindings to reinforce read-only mode
		// Users must edit prd.json files directly to make changes
		case "e", "i", "d", "x", "backspace", "delete":
			// No-op: board is view-only
			return a, nil
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.board.SetSize(msg.Width, msg.Height)
		a.helpOverlay.SetSize(msg.Width, msg.Height)
		a.filterOverlay.SetSize(msg.Width, msg.Height)

	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		if a.showHelp || a.expandedCard != nil {
			// Don't handle mouse when overlays are shown
			return a, nil
		}
		a.handleMouseEvent(msg)
		return a, nil

	case FileChangedMsg:
		// Re-parse the changed file and update the board
		hasAnimations := a.handleFileChange(msg)
		// Continue listening for more file changes
		// If animations are running, also start animation ticks
		if hasAnimations {
			return a, tea.Batch(a.listenForFileChanges(), animationTickCmd())
		}
		return a, a.listenForFileChanges()

	case AnimationTickMsg:
		// Update animations - clear expired ones
		a.board.ClearExpiredAnimations()
		// If animations are still running, continue ticking
		if a.board.HasAnimatingCards() {
			return a, animationTickCmd()
		}
		return a, nil

	case WatcherStoppedMsg:
		// Watcher has been stopped, no need to listen for more changes
		// This happens during graceful shutdown
		return a, nil
	}

	return a, nil
}

// handleMouseEvent processes mouse events for scrolling.
func (a *App) handleMouseEvent(msg tea.MouseMsg) {
	// Determine which column was clicked based on X position
	colWidth := a.width / 3
	colIndex := msg.X / colWidth
	if colIndex < 0 {
		colIndex = 0
	}
	if colIndex > 2 {
		colIndex = 2
	}

	switch msg.Type {
	case tea.MouseWheelUp:
		// Scroll up in the target column
		a.board.ScrollColumnUp(colIndex)
	case tea.MouseWheelDown:
		// Scroll down in the target column
		a.board.ScrollColumnDown(colIndex)
	}
}

// handleFileChange processes a file change event and updates the board.
// Returns true if any cards moved columns (animations were triggered).
func (a *App) handleFileChange(msg FileChangedMsg) bool {
	switch msg.Op {
	case watcher.OpModify, watcher.OpCreate:
		// Re-parse the modified/created file
		result, err := parser.ParseFile(msg.FilePath)
		if err != nil {
			// Ignore parse errors (file may be temporarily invalid during save)
			return false
		}

		// Find and update the existing result or add new one
		found := false
		for i, pr := range a.parseResults {
			if pr.FilePath == msg.FilePath {
				a.parseResults[i] = result
				found = true
				break
			}
		}
		if !found {
			a.parseResults = append(a.parseResults, result)
		}

		// Update project list in filter overlay
		a.filterOverlay.UpdateProjects(extractProjectNames(a.parseResults))

		// Rebuild the board with current filter (returns true if animations started)
		return a.rebuildBoard()

	case watcher.OpDelete:
		// Remove the deleted file from parse results
		for i, pr := range a.parseResults {
			if pr.FilePath == msg.FilePath {
				a.parseResults = append(a.parseResults[:i], a.parseResults[i+1:]...)
				break
			}
		}

		// Update project list in filter overlay (also removes deleted projects from selection)
		a.filterOverlay.UpdateProjects(extractProjectNames(a.parseResults))

		// Rebuild the board with current filter (returns true if animations started)
		return a.rebuildBoard()
	}
	return false
}

// View implements tea.Model.
func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Header with ASCII art title or fallback text for narrow terminals
	header := a.renderHeader()

	// Filter indicator (shown when filter is active)
	// Always reserve space for the filter indicator to prevent layout shifts
	filterIndicator := a.renderFilterIndicator()
	indicatorStyle := lipgloss.NewStyle().
		Width(a.width - 4).
		Align(lipgloss.Center).
		MarginBottom(0)

	// Always render the filter indicator space, even when empty
	// This keeps the logo position fixed when the badge appears/disappears
	if filterIndicator != "" {
		filterIndicator = indicatorStyle.Render(filterIndicator)
	} else {
		// Reserve space with a single space to maintain consistent layout
		// This ensures the line height is preserved even when no badge is shown
		filterIndicator = indicatorStyle.Render(" ")
	}

	// Board view
	boardView := a.board.View()

	// Status bar indicating view-only mode with navigation hints
	statusBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	statusBar := statusBarStyle.Render("↑↓←→/hjkl: navigate | Enter: expand | f: filter | ?: help | q: quit")

	// Combine status bar and help message
	footer := lipgloss.JoinVertical(lipgloss.Left, statusBar)

	// Base view with equal padding on all sides
	// Apply padding to the entire content area for breathing room from terminal edge
	// Left and right padding are equal (HorizontalPadding), bottom matches top (from header style)
	paddedStyle := lipgloss.NewStyle().
		PaddingLeft(HorizontalPadding).
		PaddingRight(HorizontalPadding).
		PaddingBottom(BottomPadding)

	// Build content with filter indicator space always present between header and board
	var contentParts []string
	contentParts = append(contentParts, header, filterIndicator, boardView, footer)

	baseView := paddedStyle.Render(lipgloss.JoinVertical(lipgloss.Left, contentParts...))

	// Overlay help if showing
	if a.showHelp {
		return a.renderWithOverlay(baseView, a.helpOverlay.View())
	}

	// Overlay filter if showing
	if a.showFilter {
		return a.renderWithOverlay(baseView, a.filterOverlay.View())
	}

	// Overlay expanded card if showing
	if a.expandedCard != nil {
		expandedWidth := a.width - 20
		if expandedWidth < 50 {
			expandedWidth = 50
		}
		if expandedWidth > 80 {
			expandedWidth = 80
		}
		return a.renderWithOverlay(baseView, a.expandedCard.RenderExpanded(expandedWidth))
	}

	return baseView
}

// renderHeader renders the header with ASCII art title or fallback text.
func (a *App) renderHeader() string {
	// Use ASCII art if terminal is wide enough
	if a.width >= MinWidthForASCIIArt {
		return a.renderASCIIArtHeader()
	}

	// Fallback to simple text header for narrow terminals
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		PaddingTop(TopPadding).
		MarginBottom(1)

	return headerStyle.Render("PRD Monitor")
}

// renderASCIIArtHeader renders the ASCII art title header.
func (a *App) renderASCIIArtHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		PaddingTop(TopPadding).
		MarginBottom(1)

	// Join the ASCII art lines
	artLines := strings.Join(ASCIIArtTitle, "\n")

	// Center the ASCII art if the terminal is wider than the art
	if a.width > ASCIIArtWidth {
		titleStyle = titleStyle.Width(a.width - 4).Align(lipgloss.Center)
	}

	return titleStyle.Render(artLines)
}

// renderFilterIndicator renders the filter indicator badge when a filter is active.
// Returns empty string when no filter is active (i.e., showing all projects).
func (a *App) renderFilterIndicator() string {
	fs := a.filterOverlay.FilterState()

	// Don't show indicator when no projects are selected
	if !fs.IsFiltering() {
		return ""
	}

	// Create a prominent badge style with background color
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).   // Black text
		Background(lipgloss.Color("220")). // Yellow background for visibility
		Bold(true).
		Padding(0, 1)

	// Build the filter description
	count := len(fs.SelectedProjects)
	var filterText string
	if fs.Mode == FilterModeInclude {
		if count == 1 {
			// Show the project name if only one selected
			for p := range fs.SelectedProjects {
				filterText = FilterIndicatorPrefix + p
			}
		} else {
			filterText = FilterIndicatorPrefix + fmt.Sprintf("%d projects", count)
		}
	} else {
		// Exclude mode
		if count == 1 {
			for p := range fs.SelectedProjects {
				filterText = "Excluding: " + p
			}
		} else {
			filterText = fmt.Sprintf("Excluding: %d projects", count)
		}
	}

	return badgeStyle.Render(filterText)
}

// IsFilterActive returns true when a project filter is active.
func (a *App) IsFilterActive() bool {
	return a.filterOverlay.FilterState().IsFiltering()
}

// renderWithOverlay renders an overlay centered on top of the base view.
func (a *App) renderWithOverlay(base, overlay string) string {
	// Create a dimmed version of the base (we just show the overlay on top)
	// For simplicity, we center the overlay on the screen

	overlayStyle := lipgloss.NewStyle().
		Width(a.width).
		Height(a.height).
		Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(overlay)
}

// Run starts the Bubble Tea program without file watching.
func Run(parseResults []*parser.ParseResult) error {
	return RunWithWatcher(parseResults, nil, "")
}

// RunWithWatcher starts the Bubble Tea program with optional file watching.
func RunWithWatcher(parseResults []*parser.ParseResult, w *watcher.Watcher, rootDir string) error {
	app := NewApp(parseResults, w, rootDir)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
