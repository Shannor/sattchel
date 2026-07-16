package core

import (
	"context"
)

// TrackerRepository represents the Tracker surface area required to implement the core domain
type TrackerRepository interface {
	UOW
	// CreateProject creates a new project
	CreateProject(ctx context.Context, project *Project) (*Project, error)
	// GetProjects returns all saved projects
	GetProjects(ctx context.Context) ([]Project, error)
	// GetProject returns a single project
	GetProject(ctx context.Context, projectID string) (*Project, error)
	// UpdateProject updates a project, will only update the fields that are not empty
	UpdateProject(ctx context.Context, project *Project) (*Project, error)

	// CreateGoal creates a new goal for a project
	CreateGoal(ctx context.Context, projectID string, goal *Goal) (*Goal, error)
	// GetGoals returns all goals for a project
	GetGoals(ctx context.Context, projectID string) ([]Goal, error)
	// GetGoal returns a single goal for a project
	GetGoal(ctx context.Context, goalID string) (*Goal, error)
	// UpdateGoal updates a goal, will only update the fields that are not empty
	UpdateGoal(ctx context.Context, goal *Goal) (*Goal, error)
	// QueryGoals method to search for particular goals based on criteria
	QueryGoals(ctx context.Context, projectID string, query *GoalQuery) ([]Goal, error)

	CreateMember(ctx context.Context, member *Member) (*Member, error)
	GetMember(ctx context.Context, memberID string) (*Member, error)
	GetMembers(ctx context.Context) ([]Member, error)
	UpdateMember(ctx context.Context, member *Member) (*Member, error)
	DeleteMember(ctx context.Context, memberID string) error
}

// UOW represents a unit of work. It will be used to group any writes
// that need to be batched together.
type UOW interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}
