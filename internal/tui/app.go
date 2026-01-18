// Package tui provides the terminal user interface for PRDMonitor.
package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lanham/prdmonitor/internal/parser"
)

// App is the main Bubble Tea model for the PRDMonitor application.
type App struct {
	board  *Board
	width  int
	height int
}

// NewApp creates a new App model with the given parsed PRD results.
func NewApp(parseResults []*parser.ParseResult) *App {
	return &App{
		board: NewBoard(parseResults),
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return nil
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
	}

	return a, nil
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

// Run starts the Bubble Tea program.
func Run(parseResults []*parser.ParseResult) error {
	app := NewApp(parseResults)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
