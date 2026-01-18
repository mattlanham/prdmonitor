// Package model defines data structures for PRD and user stories.
package model

// PRD represents a Product Requirements Document parsed from prd.json.
type PRD struct {
	Name        string      `json:"name"`
	BranchName  string      `json:"branchName"`
	UserStories []UserStory `json:"userStories"`
}

// UserStory represents a single user story within a PRD.
type UserStory struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
	Priority           int      `json:"priority"`
	Status             string   `json:"status"` // "incomplete", "in-progress", "complete"
}

// Status constants for user stories.
const (
	StatusIncomplete = "incomplete"
	StatusInProgress = "in-progress"
	StatusComplete   = "complete"
)
