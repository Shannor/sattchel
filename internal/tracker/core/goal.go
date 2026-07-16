package core

import (
	"errors"
	"strings"

	"golang.org/x/exp/slices"
)

// Goal the main building block of the tracker.
// Holds relationships to all main entities in the system
type Goal struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ProjectID   string     `json:"projectId"`
	Status      GoalStatus `json:"status"`
	Impact      Impact     `json:"impact"`
	Effort      Effort     `json:"effort"`
	Parent      *Link      `json:"parent"`
	Children    []Goal     `json:"children"`
	Member      *Member    `json:"member"`
}

// GoalStatus represents the status of a goal
type GoalStatus string

const (
	GoalInProgress GoalStatus = "in-progress"
	GoalCompleted  GoalStatus = "completed"
	GoalCancelled  GoalStatus = "cancelled"
	GoalOpen       GoalStatus = "open"
	GoalDraft      GoalStatus = "draft"
)

// Impact represents the expected outcome of a goal
type Impact string

const (
	LowImpact     Impact = "low"
	MediumImpact  Impact = "medium"
	HighImpact    Impact = "high"
	UnknownImpact Impact = "unknown"
)

func impactValue(imp Impact) int {
	switch imp {
	case HighImpact:
		return 3
	case MediumImpact:
		return 2
	case LowImpact:
		return 1
	case UnknownImpact:
		return 0
	default:
		return 0
	}
}

// Compare the "high" impact will be the best result with lower impacts ranking lower
func (i Impact) Compare(b Impact) int {
	return impactValue(b) - impactValue(i)
}

// Effort represents the effort required to complete a goal
type Effort string

const (
	LowEffort     Effort = "low"
	MediumEffort  Effort = "medium"
	HighEffort    Effort = "high"
	UnknownEffort Effort = "unknown"
)

// lowValue returns the priority value of an effort
func lowValue(effort Effort) int {
	switch effort {
	case LowEffort:
		return 3
	case MediumEffort:
		return 2
	case HighEffort:
		return 1
	case UnknownEffort:
		return 0
	default:
		return 0
	}
}

// Compare for an effort will return the one with a lower effort is "better"
func (e Effort) Compare(b Effort) int {
	return lowValue(b) - lowValue(e)
}

func NewGoal(projectID, name string, options GoalOptions) Goal {
	g := Goal{
		Name:        name,
		Description: options.Description,
		ProjectID:   projectID,
	}
	if options.Status != "" {
		g.Status = options.Status
	} else {
		g.Status = GoalDraft
	}
	if options.Impact != "" {
		g.Impact = options.Impact
	} else {
		g.Impact = UnknownImpact
	}
	if options.Effort != "" {
		g.Effort = options.Effort
	} else {
		g.Effort = UnknownEffort
	}
	if options.MemberID != "" {
		g.Member = &Member{ID: options.MemberID}
	}
	if options.ParentID != "" {
		g.Parent = &Link{TargetID: options.ParentID}
	}
	return g
}

func (g *Goal) AssignMember(m *Member) {
	g.Member = m
}

func (g *Goal) UnassignMember() {
	g.Member = nil
}

// Compare for a goal will first sort by higher impact, lower effort, required relationship, and finally name as backup
func (g *Goal) Compare(b Goal) int {
	if g.Impact.Compare(b.Impact) != 0 {
		return g.Impact.Compare(b.Impact)
	}
	if g.Effort.Compare(b.Effort) != 0 {
		return g.Effort.Compare(b.Effort)
	}
	if g.HasParent() && b.HasParent() {
		return g.Parent.Relationship.Compare(b.Parent.Relationship)
	}
	return strings.Compare(g.Name, b.Name)
}

func (g *Goal) HasMember() bool {
	return g.Member != nil && g.Member.ID != ""
}

func (g *Goal) HasParent() bool {
	return g.Parent != nil && g.Parent.TargetID != ""
}

func (g *Goal) IsRoot() bool {
	return g.Parent == nil || g.Parent.TargetID == ""
}

func (g *Goal) IsDoItNow() bool {
	return g.Impact == HighImpact && g.Effort == LowEffort
}

func (g *Goal) IsHonestWork() bool {
	return g.Impact == HighImpact && g.Effort == HighEffort
}

func (g *Goal) IsWhy() bool {
	return g.Impact == LowImpact && g.Effort == HighEffort
}

func (g *Goal) IsSnacking() bool {
	return g.Effort == LowEffort && g.Impact == LowImpact
}

func (g *Goal) AttachChild(child *Goal, rel LinkRelationship, desc string) error {
	if child.ID == "" {
		return errors.New("child goal ID is missing")
	}
	if child.ID == g.ID {
		return errors.New("cannot attach goal to itself")
	}

	parentLink := Link{
		TargetID:    g.ID,
		Description: desc,
	}
	if rel != "" {
		parentLink.Relationship = rel
	} else {
		parentLink.Relationship = LinkOptional
	}

	child.Parent = &parentLink

	if slices.ContainsFunc(g.Children, func(it Goal) bool { return it.ID == child.ID }) {
		return nil
	}
	g.Children = append(g.Children, *child)
	return nil
}

// DetachChild may need to revisit this. We don't want orphaned goals.
// So we may need to be replaced instead of removed
func (g *Goal) DetachChild(child *Goal) error {
	if child.ID == "" {
		return errors.New("child goal ID is missing")
	}
	if child.ID == g.ID {
		return errors.New("cannot remove goal from itself")
	}

	g.Children = slices.DeleteFunc(g.Children, func(l Goal) bool { return l.ID == child.ID })
	// TODO: Revisit this decision
	child.Parent = nil
	return nil
}

// GoalQuery optional query options
type GoalQuery struct {
	ParentID      string
	MemberIDs     []string
	Impacts       []Impact
	Efforts       []Effort
	Relationships []LinkRelationship
	Statuses      []GoalStatus
	MissingFields []string
}

type GoalOptions struct {
	MemberID         string
	ParentID         string
	Description      string
	Impact           Impact
	Effort           Effort
	LinkRelationship LinkRelationship
	Status           GoalStatus
}

// CanBeRoot returns if it's allowed for the incoming goal request to be the root of the project.
func (o *GoalOptions) CanBeRoot(p *Project) bool {
	if p == nil {
		return false
	}
	return p.RootGoalID == "" && o.ParentID == ""
}

// WantsRoot returns if the goal request wants to be the root of the project.
func (o *GoalOptions) WantsRoot() bool {
	return o.ParentID == ""
}

// IsOrphan returns if the goal request is an orphan
func (o *GoalOptions) IsOrphan(p *Project) bool {
	return o.ParentID == "" && !o.CanBeRoot(p)
}
