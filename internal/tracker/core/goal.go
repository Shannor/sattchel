package core

import (
	"errors"

	"golang.org/x/exp/slices"
)

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

type GoalStatus string

const (
	GoalInProgress GoalStatus = "in-progress"
	GoalCompleted  GoalStatus = "completed"
	GoalCancelled  GoalStatus = "cancelled"
	GoalOpen       GoalStatus = "open"
	GoalDraft      GoalStatus = "draft"
)

type Impact string

const (
	LowImpact     Impact = "low"
	MediumImpact  Impact = "medium"
	HighImpact    Impact = "high"
	UnknownImpact Impact = "unknown"
)

type Effort string

const (
	XSmallEffort  Effort = "xs"
	SmallEffort   Effort = "s"
	MediumEffort  Effort = "m"
	LargeEffort   Effort = "l"
	XLargeEffort  Effort = "xl"
	UnknownEffort Effort = "unknown"
)

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

func (g *Goal) HasMember() bool {
	return g.Member != nil && g.Member.ID != ""
}

func (g *Goal) HasParent() bool {
	return g.Parent != nil && g.Parent.TargetID != ""
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

	if slices.ContainsFunc(g.Children, func(it Goal) bool { return it.ID == child.ID }) {
		return nil
	}
	g.Children = append(g.Children, *child)
	child.Parent = &parentLink
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
