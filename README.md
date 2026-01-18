# PRDMonitor

A terminal-based Kanban board that monitors directories for `prd.json` files and displays user stories in a beautiful TUI (Terminal User Interface).

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green)

## Overview

PRDMonitor provides a read-only dashboard for visualizing user stories across multiple projects. It automatically discovers `prd.json` files in your directory tree and displays them in a three-column Kanban board: **Incomplete**, **In Progress**, and **Complete**.

The board updates in real-time as you edit your PRD files, making it perfect for keeping track of project progress across multiple repositories.

## Features

- **Three-Column Kanban Board** - Stories organized by status (Incomplete, In Progress, Complete)
- **Live Updates** - File changes detected within 2-5 seconds via fsnotify
- **Multi-Project Support** - Aggregate stories from 20+ projects without performance issues
- **Project Filtering** - Filter the board by individual project
- **Keyboard Navigation** - Navigate with arrow keys, hjkl, or Tab
- **Scrollable Columns** - Handle large numbers of stories per column
- **Card Animations** - Visual feedback when cards move between columns
- **Smart Sorting** - Cards sorted by last modification date
- **Graceful Error Handling** - Malformed JSON files don't crash the app
- **Fast Discovery** - Parallel directory scanning with smart exclusions (node_modules, .git, etc.)

## How It Was Built

PRDMonitor was built with Claude Code (Claude Opus 4.5) in a single development session. The project demonstrates AI-assisted development, with Claude implementing features incrementally based on a structured PRD (Product Requirements Document).

### Tech Stack

- **Language**: [Go](https://go.dev/) 1.24
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) - A functional framework for building terminal apps
- **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal layouts
- **File Watching**: [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform file system notifications

### Development Timeline

The entire application was built in approximately **3 hours** (from first commit at 11:46 to final feature at 14:47 on January 18, 2026). This included:

- 24 feature implementations (PM-001 through PM-024)
- 3 bug fixes
- Comprehensive test coverage
- Documentation

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/mattlanham/prdmonitor.git
cd prdmonitor

# Build the binary
go build -o prdmonitor .

# Optional: Move to your PATH
sudo mv prdmonitor /usr/local/bin/
```

### Dependencies

All dependencies are managed via Go modules and will be automatically downloaded:

```bash
go mod download
```

## Usage

```bash
# Monitor current directory
prdmonitor

# Monitor a specific directory
prdmonitor /path/to/projects
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `←` `→` `h` `l` | Navigate between columns |
| `↑` `↓` `j` `k` | Navigate between cards |
| `Tab` | Cycle between columns |
| `f` | Open project filter |
| `?` | Show help overlay |
| `q` `Ctrl+C` | Quit application |

## PRD File Format

PRDMonitor looks for files named `prd.json` with the following schema:

```json
{
  "name": "Project Name",
  "branchName": "feature/branch-name",
  "userStories": [
    {
      "id": "PM-001",
      "title": "Story Title",
      "description": "Detailed description of the user story",
      "acceptanceCriteria": [
        "Criterion 1",
        "Criterion 2"
      ],
      "priority": 0,
      "status": "incomplete",
      "updatedAt": "2026-01-18T12:00:00Z"
    }
  ]
}
```

### Status Values

- `incomplete` - Displayed in the **Incomplete** column
- `in-progress` - Displayed in the **In Progress** column
- `complete` - Displayed in the **Complete** column

## Building for Multiple Platforms

Go makes cross-compilation straightforward:

```bash
# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o prdmonitor-darwin-arm64 .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o prdmonitor-darwin-amd64 .

# Linux (x86_64)
GOOS=linux GOARCH=amd64 go build -o prdmonitor-linux-amd64 .

# Linux (ARM64)
GOOS=linux GOARCH=arm64 go build -o prdmonitor-linux-arm64 .

# Windows (x86_64)
GOOS=windows GOARCH=amd64 go build -o prdmonitor-windows-amd64.exe .
```

## Project Structure

```
prdmonitor/
├── main.go              # Entry point, CLI argument parsing
├── internal/
│   ├── model/           # Data structures for PRD and user stories
│   ├── parser/          # JSON parsing and validation
│   ├── scanner/         # Recursive directory scanning
│   ├── watcher/         # File system watching with fsnotify
│   └── tui/             # Bubble Tea TUI components
│       ├── app.go       # Main application model
│       ├── board.go     # Kanban board view
│       ├── card.go      # User story card component
│       ├── column.go    # Column component (scrollable)
│       ├── filter.go    # Project filter overlay
│       └── help.go      # Help overlay
├── go.mod
├── go.sum
└── ralph/
    └── prd.json         # Example PRD file
```

## Running Tests

```bash
go test ./...
```

## License

MIT License - See [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [Charm](https://charm.sh/) libraries (Bubble Tea, Lip Gloss)
- Developed with [Claude Code](https://claude.ai/code) by Anthropic
