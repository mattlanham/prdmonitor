// Package tui provides the terminal user interface for PRDMonitor.
package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/parser"
	"lanham/prdmonitor/internal/watcher"
)

// ReadOnlyMessage is displayed in the status bar to indicate view-only mode.
const ReadOnlyMessage = "VIEW-ONLY"

// EditHelpMessage is displayed to guide users on how to make changes.
const EditHelpMessage = "Edit prd.json files directly to make changes"

// FileChangedMsg is sent when a watched file changes.
type FileChangedMsg struct {
	FilePath string
	Op       watcher.Operation
}

// App is the main Bubble Tea model for the PRDMonitor application.
type App struct {
	board        *Board
	width        int
	height       int
	watcher      *watcher.Watcher
	parseResults []*parser.ParseResult
	rootDir      string
	showHelp     bool         // Whether to show the help overlay
	expandedCard *Card        // Currently expanded card (nil if none)
	helpOverlay  *HelpOverlay // Help overlay component
}

// NewApp creates a new App model with the given parsed PRD results.
func NewApp(parseResults []*parser.ParseResult, w *watcher.Watcher, rootDir string) *App {
	return &App{
		board:        NewBoard(parseResults),
		watcher:      w,
		parseResults: parseResults,
		rootDir:      rootDir,
		showHelp:     false,
		expandedCard: nil,
		helpOverlay:  NewHelpOverlay(),
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	if a.watcher != nil {
		return a.listenForFileChanges()
	}
	return nil
}

// listenForFileChanges returns a command that listens for file change events.
func (a *App) listenForFileChanges() tea.Cmd {
	return func() tea.Msg {
		select {
		case event := <-a.watcher.Events():
			return FileChangedMsg{
				FilePath: event.FilePath,
				Op:       event.Op,
			}
		case <-a.watcher.Errors():
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
			if a.showHelp {
				a.showHelp = false
				return a, nil
			}
			return a, nil
		}

		// Handle help overlay toggle
		if msg.String() == "?" {
			if a.expandedCard != nil {
				// Close expanded card first
				a.expandedCard = nil
			}
			a.showHelp = !a.showHelp
			return a, nil
		}

		// If help is showing, only allow closing it
		if a.showHelp {
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

	case FileChangedMsg:
		// Re-parse the changed file and update the board
		a.handleFileChange(msg)
		// Continue listening for more file changes
		return a, a.listenForFileChanges()
	}

	return a, nil
}

// handleFileChange processes a file change event and updates the board.
func (a *App) handleFileChange(msg FileChangedMsg) {
	switch msg.Op {
	case watcher.OpModify, watcher.OpCreate:
		// Re-parse the modified/created file
		result, err := parser.ParseFile(msg.FilePath)
		if err != nil {
			// Ignore parse errors (file may be temporarily invalid during save)
			return
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

		// Rebuild the board with updated data
		a.board = NewBoard(a.parseResults)
		a.board.SetSize(a.width, a.height)

	case watcher.OpDelete:
		// Remove the deleted file from parse results
		for i, pr := range a.parseResults {
			if pr.FilePath == msg.FilePath {
				a.parseResults = append(a.parseResults[:i], a.parseResults[i+1:]...)
				break
			}
		}

		// Rebuild the board
		a.board = NewBoard(a.parseResults)
		a.board.SetSize(a.width, a.height)
	}
}

// View implements tea.Model.
func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	header := headerStyle.Render("PRDMonitor - Kanban Board")

	// Board view
	boardView := a.board.View()

	// Status bar indicating view-only mode with navigation hints
	statusBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	statusBar := statusBarStyle.Render(ReadOnlyMessage + " | ↑↓←→/hjkl: navigate | Enter: expand | ?: help | q: quit")

	// Help message for editing
	helpMsgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	helpMsg := helpMsgStyle.Render(EditHelpMessage)

	// Combine status bar and help message
	footer := lipgloss.JoinVertical(lipgloss.Left, statusBar, helpMsg)

	// Base view
	baseView := lipgloss.JoinVertical(lipgloss.Left, header, boardView, footer)

	// Overlay help if showing
	if a.showHelp {
		return a.renderWithOverlay(baseView, a.helpOverlay.View())
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
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
