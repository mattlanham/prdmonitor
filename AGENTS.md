# AGENTS.md - AI Agent Guidance for PRDMonitor

## Project Overview

PRDMonitor is a terminal-based Kanban board application written in Go. It monitors directories for `prd.json` files and displays user stories in a read-only TUI with three columns: Incomplete, In Progress, and Complete.

## Tech Stack

- **Language**: Go (1.21+)
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (preferred) or tview
- **File Watching**: [fsnotify](https://github.com/fsnotify/fsnotify)
- **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss) for terminal styling

## Project Structure

```
prdmonitor/
├── main.go              # Entry point, CLI argument parsing
├── internal/
│   ├── scanner/         # Directory scanning and prd.json discovery
│   │   └── scanner.go
│   ├── parser/          # JSON parsing and validation
│   │   └── parser.go
│   ├── watcher/         # File system watching with fsnotify
│   │   └── watcher.go
│   ├── model/           # Data structures for PRD and user stories
│   │   └── model.go
│   └── tui/             # Bubble Tea TUI components
│       ├── app.go       # Main application model
│       ├── board.go     # Kanban board view
│       ├── card.go      # User story card component
│       ├── column.go    # Column component (scrollable)
│       └── help.go      # Help overlay
├── go.mod
├── go.sum
└── ralph/
    └── prd.json         # Example PRD file
```

## Data Structures

### prd.json Schema

```go
type PRD struct {
    Name       string      `json:"name"`
    BranchName string      `json:"branchName"`
    UserStories []UserStory `json:"userStories"`
}

type UserStory struct {
    ID                 string   `json:"id"`
    Title              string   `json:"title"`
    Description        string   `json:"description"`
    AcceptanceCriteria []string `json:"acceptanceCriteria"`
    Priority           int      `json:"priority"`
    Status             string   `json:"status"` // "incomplete", "in-progress", "complete"
}
```

### Status Values

- `incomplete` → Incomplete column
- `in-progress` → In Progress column
- `complete` → Complete column
- Unknown values → Default to Incomplete column

## Implementation Priorities

Follow user story priorities (0 = highest):

1. **PM-001**: CLI argument parsing for folder selection
2. **PM-002**: Recursive directory scanning for prd.json files
3. **PM-003**: JSON parsing and validation
4. **PM-004**: TUI with Kanban board layout (3 columns)
5. **PM-005**: User story card display
6. **PM-006**: Status-based card sorting
7. **PM-007**: File change detection with fsnotify
8. **PM-008**: New file detection
9. **PM-009**: Read-only interface
10. **PM-010**: File deletion handling
11. **PM-011**: Multi-project aggregation
12. **PM-012**: Keyboard navigation
13. **PM-013**: Scrollable columns
14. **PM-014**: Graceful exit and cleanup

## Key Requirements

### CLI Behavior

- Accept optional folder path as first argument: `prdmonitor /path/to/projects`
- Default to current directory if no argument provided
- Validate folder exists and is readable
- Print errors to stderr

### TUI Requirements

- Three columns: Incomplete, In Progress, Complete
- Cards show: project name, story ID, title, description (truncated)
- Use box-drawing characters for card borders
- Adapt layout to terminal width
- Display placeholder message in empty columns
- Status bar indicating view-only mode

### Keyboard Shortcuts

- Arrow keys / `hjkl`: Navigate between cards and columns
- `Enter` / `Space`: Expand card to show full description
- `Tab`: Cycle between columns
- `q` / `Ctrl+C`: Quit application
- `?`: Show help overlay

### File Watching

- Monitor all discovered prd.json files
- Detect file changes, creations, and deletions
- Update TUI within 2-5 seconds of changes
- Keep TUI responsive during file watching

## Development Guidelines

### Error Handling

- Handle malformed JSON gracefully with error logging
- Invalid folders should print clear error messages to stderr
- File watcher errors should not crash the application

### Performance

- Support at least 20 projects without performance issues
- Use goroutines for file watching
- Batch TUI updates to avoid flickering

### Code Style

- Follow standard Go conventions
- Use `internal/` for private packages
- Keep main.go minimal - delegate to internal packages
- Write idiomatic Bubble Tea code with Model-Update-View pattern

### Testing

- Unit test JSON parsing with valid and malformed inputs
- Test scanner finds prd.json files correctly
- Test status mapping to columns

## Build & Run

```bash
# Build
go build -o prdmonitor .

# Run with current directory
./prdmonitor

# Run with specific folder
./prdmonitor /path/to/projects
```

## Dependencies

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/fsnotify/fsnotify
```
