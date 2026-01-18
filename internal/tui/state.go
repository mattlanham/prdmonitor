// state.go provides persistence for PRDMonitor application state.
package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// StateFileName is the name of the state file stored in the root directory.
const StateFileName = ".prdmonitor.state"

// PersistentState represents the application state that persists across sessions.
type PersistentState struct {
	// Filter contains the filter configuration.
	Filter *PersistentFilterState `json:"filter,omitempty"`
}

// PersistentFilterState represents the filter state for persistence.
type PersistentFilterState struct {
	SelectedProjects []string `json:"selectedProjects,omitempty"`
	Mode             string   `json:"mode,omitempty"` // "include" or "exclude"
}

// NewPersistentState creates an empty persistent state.
func NewPersistentState() *PersistentState {
	return &PersistentState{}
}

// LoadState loads the persistent state from the given root directory.
// Returns an empty state if the file doesn't exist or can't be read.
func LoadState(rootDir string) *PersistentState {
	statePath := filepath.Join(rootDir, StateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		// File doesn't exist or can't be read - return empty state
		return NewPersistentState()
	}

	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		// Invalid JSON - return empty state
		return NewPersistentState()
	}

	return &state
}

// Save saves the persistent state to the given root directory.
func (s *PersistentState) Save(rootDir string) error {
	statePath := filepath.Join(rootDir, StateFileName)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}

// SetFilterState updates the persistent state from a FilterState.
func (s *PersistentState) SetFilterState(fs *FilterState) {
	if fs == nil || !fs.IsFiltering() {
		s.Filter = nil
		return
	}

	// Convert selected projects map to slice
	projects := make([]string, 0, len(fs.SelectedProjects))
	for p := range fs.SelectedProjects {
		projects = append(projects, p)
	}

	mode := "include"
	if fs.Mode == FilterModeExclude {
		mode = "exclude"
	}

	s.Filter = &PersistentFilterState{
		SelectedProjects: projects,
		Mode:             mode,
	}
}

// GetFilterState converts the persistent filter state to a FilterState.
// Returns nil if no filter is stored.
func (s *PersistentState) GetFilterState() *FilterState {
	if s.Filter == nil || len(s.Filter.SelectedProjects) == 0 {
		return nil
	}

	fs := NewFilterState()

	// Convert slice back to map
	for _, p := range s.Filter.SelectedProjects {
		fs.SelectedProjects[p] = true
	}

	if s.Filter.Mode == "exclude" {
		fs.Mode = FilterModeExclude
	} else {
		fs.Mode = FilterModeInclude
	}

	return fs
}
