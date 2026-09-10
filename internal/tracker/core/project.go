package core

import "strings"

type ProjectStatus string

const (
	ProjectInProgress ProjectStatus = "in-progress"
	ProjectComplete   ProjectStatus = "complete"
	ProjectDraft      ProjectStatus = "draft"
)

func (s ProjectStatus) IsValid() bool {
	switch s {
	case ProjectDraft, ProjectInProgress, ProjectComplete:
		return true
	default:
		return false
	}
}

// Project represents the base of an entire collection of Goals.
// This allows us to have multiple projects with different goals.
type Project struct {
	ID          string        `json:"id"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Status      ProjectStatus `json:"status"`
	// RootGoalID is the ID of the first goal in the project.
	// It will serve as the start of the tree and there will only be one.
	RootGoalID string `json:"rootGoalId"`
}

// NormalizedLabel returns the label without whitespace and in lowercase.
// Making it easier for comparisons.
func (p *Project) NormalizedLabel() string { return normalize(p.Label) }

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (p *Project) NormalizedStatus() ProjectStatus {
	if p.Status == "" {
		return ProjectDraft
	}
	return p.Status
}

func (p *Project) SetStatus(status ProjectStatus) {
	if status == "" {
		p.Status = ProjectDraft
		return
	}
	p.Status = status
}

func (p *Project) SetRoot(g Goal) {
	p.RootGoalID = g.ID
}
