package core

import (
	"context"
	"errors"
	"fmt"
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
)

func (s *Service) CreateProject(ctx context.Context, name string, description string) (*Project, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	project := &Project{
		Label:       name,
		Description: description,
	}

	projects, err := s.repo.GetProjects(ctx)
	if err != nil {
		return nil, err
	}

	label := project.NormalizedLabel()
	for _, p := range projects {
		if p.NormalizedLabel() == label {
			return nil, fmt.Errorf("project %s: %w", label, ErrProjectAlreadyExists)
		}
	}

	p, err := s.repo.CreateProject(ctx, project)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) CreateGoal(ctx context.Context, projectID string, goalName string, options GoalOptions) (*Goal, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w - project ID", ErrMissingRequiredFields)
	}
	if goalName == "" {
		return nil, fmt.Errorf("%w - goal name", ErrMissingRequiredFields)
	}

	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	newGoal := NewGoal(projectID, goalName, options)
	if options.MemberID != "" {
		member, err := s.repo.GetMember(ctx, options.MemberID)
		if err != nil {
			return nil, err
		}
		newGoal.AssignMember(member)
	}

	if options.WantsRoot() {
		if !options.CanBeRoot(p) {
			return nil, fmt.Errorf("project already has a root goal: %w", ErrInvalidRequest)
		}
		g, err := s.repo.CreateGoal(ctx, projectID, new(newGoal))
		if err != nil {
			return nil, err
		}
		p.SetRoot(*g)
		_, err = s.repo.UpdateProject(ctx, p)
		if err != nil {
			return nil, err
		}
		return g, nil
	}

	if options.ParentID == "" {
		return nil, errors.New("no orphan goals allowed")
	}
	// Attaching goal to a parent goal
	parent, err := s.repo.GetGoal(ctx, options.ParentID)
	if err != nil {
		return nil, err
	}

	child, err := s.repo.CreateGoal(ctx, projectID, new(newGoal))
	if err != nil {
		return nil, err
	}

	err = parent.AttachChild(child, options.LinkRelationship, options.Description)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.UpdateGoal(ctx, parent)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (s *Service) AttachMember(ctx context.Context, projectID string, goalID string, memberID string) (*Goal, error) {
	_, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.GetGoal(ctx, goalID)
	if err != nil {
		return nil, err
	}
	member, err := s.repo.GetMember(ctx, memberID)
	if err != nil {
		return nil, err
	}

	g.AssignMember(member)
	updated, err := s.repo.UpdateGoal(ctx, g)
	if err != nil {
		return nil, err
	}
	return updated, nil
}
