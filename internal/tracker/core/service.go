package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Service struct {
	repo TrackerRepository
}

func NewService(repo TrackerRepository) *Service {
	return &Service{
		repo: repo,
	}
}

var (
	ErrInvalidRequest        = errors.New("invalid request")
	ErrInvalidName           = errors.New("invalid name")
	ErrProjectAlreadyExists  = errors.New("project already exists")
	ErrMissingRequiredFields = errors.New("missing required fields")
	ErrCannotMoveRoot        = errors.New("the root goal cannot be moved")
)

func (s *Service) CreateProject(ctx context.Context, name string, description string) (*Project, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	project := &Project{
		Label:       name,
		Description: description,
	}

	var result *Project
	err := s.repo.Transaction(ctx, func(txCtx context.Context) error {
		projects, err := s.repo.GetProjects(txCtx)
		if err != nil {
			return err
		}

		label := project.NormalizedLabel()
		for _, p := range projects {
			if p.NormalizedLabel() == label {
				return fmt.Errorf("project %s: %w", label, ErrProjectAlreadyExists)
			}
		}

		p, err := s.repo.CreateProject(txCtx, project)
		if err != nil {
			return err
		}
		result = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) CreateGoal(ctx context.Context, projectID string, goalName string, options GoalOptions) (*Goal, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w - project ID", ErrMissingRequiredFields)
	}
	if goalName == "" {
		return nil, fmt.Errorf("%w - goal name", ErrMissingRequiredFields)
	}

	var result *Goal
	err := s.repo.Transaction(ctx, func(txCtx context.Context) error {
		p, err := s.repo.GetProject(txCtx, projectID)
		if err != nil {
			return err
		}

		if options.IsOrphan(p) {
			return errors.New("no orphan goals allowed")
		}

		if options.WantsRoot() && !options.CanBeRoot(p) {
			return fmt.Errorf("project already has a root goal: %w", ErrInvalidRequest)
		}

		goalEntity := NewGoal(projectID, goalName, options)
		if options.MemberID != "" {
			member, err := s.repo.GetMember(txCtx, options.MemberID)
			if err != nil {
				return err
			}
			goalEntity.AssignMember(member)
		}

		newGoal, err := s.repo.CreateGoal(txCtx, projectID, new(goalEntity))
		if err != nil {
			return err
		}

		if options.WantsRoot() {
			p.SetRoot(*newGoal)
			_, err = s.repo.UpdateProject(txCtx, p)
			if err != nil {
				return err
			}
			result = newGoal
			return nil
		}

		// Attaching goal to a parent goal
		parent, err := s.repo.GetGoal(txCtx, options.ParentID)
		if err != nil {
			return err
		}

		err = parent.AttachChild(newGoal, options.LinkRelationship, options.Description)
		if err != nil {
			return err
		}

		_, err = s.repo.UpdateGoal(txCtx, parent)
		if err != nil {
			return err
		}

		res, err := s.repo.UpdateGoal(txCtx, newGoal)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) GetAllowedParents(ctx context.Context, projectID string, goalID string) ([]Goal, error) {
	goal, err := s.repo.GetGoal(ctx, goalID)
	if err != nil {
		return nil, err
	}
	if goal.IsRoot() {
		return nil, ErrCannotMoveRoot
	}

	goals, err := s.repo.GetGoals(ctx, projectID)
	if err != nil {
		return nil, err
	}
	results := make([]Goal, 0, len(goals))
	for _, g := range goals {
		if g.ID == goalID {
			continue
		}
		// Skip if goal is a child of the goal we're moving
		if g.HasParent() && g.Parent.TargetID == goal.ID {
			continue
		}
		results = append(results, g)
	}
	return results, nil
}

func (s *Service) ChangeParent(ctx context.Context, projectID string, goalID string, newParentID string, options GoalOptions) (*Goal, error) {
	var result *Goal
	err := s.repo.Transaction(ctx, func(txCtx context.Context) error {
		p, err := s.repo.GetProject(txCtx, projectID)
		if err != nil {
			return err
		}

		if goalID == p.RootGoalID {
			return ErrCannotMoveRoot
		}

		child, err := s.repo.GetGoal(txCtx, goalID)
		if err != nil {
			return err
		}
		if child.HasParent() && child.Parent.TargetID == newParentID {
			result = child
			return nil
		}
		// Make sure new parent exists before removing existing parent
		newParent, err := s.repo.GetGoal(txCtx, newParentID)
		if err != nil {
			return err
		}

		// Remove existing parent if there is one
		if child.HasParent() {
			oldParent, err := s.repo.GetGoal(txCtx, child.Parent.TargetID)
			if err != nil {
				return err
			}
			err = oldParent.DetachChild(child)
			if err != nil {
				return err
			}
			_, err = s.repo.UpdateGoal(txCtx, oldParent)
			if err != nil {
				return err
			}
		}

		err = newParent.AttachChild(child, options.LinkRelationship, options.Description)
		if err != nil {
			return err
		}
		_, err = s.repo.UpdateGoal(txCtx, child)
		if err != nil {
			return err
		}

		_, err = s.repo.UpdateGoal(txCtx, newParent)
		if err != nil {
			return err
		}

		result = child
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) AttachMember(ctx context.Context, projectID string, goalID string, memberID string) (*Goal, error) {
	var result *Goal
	err := s.repo.Transaction(ctx, func(txCtx context.Context) error {
		_, err := s.repo.GetProject(txCtx, projectID)
		if err != nil {
			return err
		}
		g, err := s.repo.GetGoal(txCtx, goalID)
		if err != nil {
			return err
		}
		member, err := s.repo.GetMember(txCtx, memberID)
		if err != nil {
			return err
		}

		g.AssignMember(member)
		updated, err := s.repo.UpdateGoal(txCtx, g)
		if err != nil {
			return err
		}
		result = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) GetProjects(ctx context.Context) ([]Project, error) {
	projects, err := s.repo.GetProjects(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(projects, func(i, j Project) int {
		return strings.Compare(i.Label, j.Label)
	})
	return projects, nil
}

func (s *Service) GetGoals(ctx context.Context, projectID string) ([]Goal, error) {
	goals, err := s.repo.GetGoals(ctx, projectID)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(goals, func(i, j Goal) int {
		return i.Compare(j)
	})
	return goals, nil
}

func (s *Service) QueryGoals(ctx context.Context, projectID string, query GoalQuery) ([]Goal, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w - project ID", ErrMissingRequiredFields)
	}
	goals, err := s.repo.QueryGoals(ctx, projectID, &query)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(goals, func(i, j Goal) int {
		return i.Compare(j)
	})
	return goals, nil
}

func (s *Service) GetRootGoal(ctx context.Context, projectID string) (*Goal, error) {
	goals, err := s.GetGoals(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(goals) == 0 {
		return nil, fmt.Errorf("no goals found")
	}

	for _, g := range goals {
		if g.IsRoot() {
			return &g, nil
		}
	}
	return nil, fmt.Errorf("no root goal found")
}

func (s *Service) GetGoal(ctx context.Context, goalID string) (*Goal, error) {
	return s.repo.GetGoal(ctx, goalID)
}

func (s *Service) CreateMember(ctx context.Context, name string, email string) (*Member, error) {
	if name == "" {
		return nil, fmt.Errorf("%w - name", ErrMissingRequiredFields)
	}
	member := &Member{
		Name:  name,
		Email: email,
	}
	return s.repo.CreateMember(ctx, member)
}

func (s *Service) GetMember(ctx context.Context, memberID string) (*Member, error) {
	if memberID == "" {
		return nil, fmt.Errorf("%w - member ID", ErrMissingRequiredFields)
	}
	return s.repo.GetMember(ctx, memberID)
}

func (s *Service) GetMembers(ctx context.Context) ([]Member, error) {
	return s.repo.GetMembers(ctx)
}

func (s *Service) UpdateMember(ctx context.Context, member *Member) (*Member, error) {
	if member == nil || member.ID == "" {
		return nil, fmt.Errorf("%w - member ID", ErrMissingRequiredFields)
	}
	return s.repo.UpdateMember(ctx, member)
}

func (s *Service) DeleteMember(ctx context.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("%w - member ID", ErrMissingRequiredFields)
	}
	return s.repo.DeleteMember(ctx, memberID)
}

func (s *Service) UpdateGoal(ctx context.Context, goalID string, name string, options GoalOptions) (*Goal, error) {
	if goalID == "" {
		return nil, fmt.Errorf("%w - goal ID", ErrMissingRequiredFields)
	}
	var result *Goal
	err := s.repo.Transaction(ctx, func(txCtx context.Context) error {
		goal, err := s.repo.GetGoal(txCtx, goalID)
		if err != nil {
			return err
		}
		if name != "" {
			goal.Name = name
		}
		if options.Description != "" {
			goal.Description = options.Description
		}
		if options.Status != "" {
			goal.Status = options.Status
		}
		if options.Impact != "" {
			goal.Impact = options.Impact
		}
		if options.Effort != "" {
			goal.Effort = options.Effort
		}
		if options.MemberID != "" {
			member, err := s.repo.GetMember(txCtx, options.MemberID)
			if err != nil {
				return err
			}
			goal.AssignMember(member)
		}
		res, err := s.repo.UpdateGoal(txCtx, goal)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
