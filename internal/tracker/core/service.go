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

	if options.IsOrphan(p) {
		return nil, errors.New("no orphan goals allowed")
	}

	if options.WantsRoot() && !options.CanBeRoot(p) {
		return nil, fmt.Errorf("project already has a root goal: %w", ErrInvalidRequest)
	}

	goalEntity := NewGoal(projectID, goalName, options)
	if options.MemberID != "" {
		member, err := s.repo.GetMember(ctx, options.MemberID)
		if err != nil {
			return nil, err
		}
		goalEntity.AssignMember(member)
	}

	newGoal, err := s.repo.CreateGoal(ctx, projectID, new(goalEntity))
	if err != nil {
		return nil, err
	}

	if options.WantsRoot() {
		p.SetRoot(*newGoal)
		_, err = s.repo.UpdateProject(ctx, p)
		if err != nil {
			return nil, err
		}
		return newGoal, nil
	}

	// Attaching goal to a parent goal
	parent, err := s.repo.GetGoal(ctx, options.ParentID)
	if err != nil {
		return nil, err
	}

	err = parent.AttachChild(newGoal, options.LinkRelationship, options.Description)
	if err != nil {
		return nil, err
	}

	_, err = s.repo.UpdateGoal(ctx, parent)
	if err != nil {
		return nil, err
	}

	result, err := s.repo.UpdateGoal(ctx, newGoal)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) ChangeParent(ctx context.Context, projectID string, goalID string, newParentID string, options GoalOptions) (*Goal, error) {
	_, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	child, err := s.repo.GetGoal(ctx, goalID)
	if err != nil {
		return nil, err
	}
	if child.HasParent() && child.Parent.TargetID == newParentID {
		return child, nil
	}
	// Make sure new parent exists before removing existing parent
	newParent, err := s.repo.GetGoal(ctx, newParentID)
	if err != nil {
		return nil, err
	}

	// Remove existing parent if there is one
	if child.HasParent() {
		oldParent, err := s.repo.GetGoal(ctx, child.Parent.TargetID)
		if err != nil {
			return nil, err
		}
		err = oldParent.DetachChild(child)
		if err != nil {
			return nil, err
		}
		_, err = s.repo.UpdateGoal(ctx, oldParent)
		if err != nil {
			return nil, err
		}
	}

	err = newParent.AttachChild(child, options.LinkRelationship, options.Description)
	if err != nil {
		return nil, err
	}
	_, err = s.repo.UpdateGoal(ctx, child)
	if err != nil {
		return nil, err
	}

	_, err = s.repo.UpdateGoal(ctx, newParent)
	if err != nil {
		return nil, err
	}

	return child, nil
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
		return strings.Compare(i.Name, j.Name)
	})
	return goals, nil
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
