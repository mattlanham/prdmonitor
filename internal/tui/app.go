// Package tui provides the terminal user interface for PRDMonitor.
package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/parser"
	"lanham/prdmonitor/internal/watcher"
)

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
}

// NewApp creates a new App model with the given parsed PRD results.
func NewApp(parseResults []*parser.ParseResult, w *watcher.Watcher, rootDir string) *App {
	return &App{
		board:        NewBoard(parseResults),
		watcher:      w,
		parseResults: parseResults,
		rootDir:      rootDir,
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
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.board.SetSize(msg.Width, msg.Height)

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

	// Footer with status bar
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	footer := footerStyle.Render("View-only mode | q: quit | ?: help")

	return lipgloss.JoinVertical(lipgloss.Left, header, boardView, footer)
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
