package core

import "strings"

// Project represents the base of an entire collection of Goals.
// This allows us to have multiple projects with different goals.
type Project struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
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

func (p *Project) SetRoot(g Goal) {
	p.RootGoalID = g.ID
}
