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
	ErrCannotDeleteRoot      = errors.New("the root goal cannot be deleted")
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

func (s *Service) DeleteProject(ctx context.Context, projectID string) error {
	if projectID == "" {
		return fmt.Errorf("%w - project ID is required", ErrMissingRequiredFields)
	}

	return s.repo.Transaction(ctx, func(txCtx context.Context) error {
		// Verify project exists
		project, err := s.repo.GetProject(txCtx, projectID)
		if err != nil {
			return err
		}
		if project == nil {
			return fmt.Errorf("project %s not found", projectID)
		}

		// Get all goals belonging to this project
		goals, err := s.repo.GetGoals(txCtx, projectID)
		if err != nil {
			return err
		}

		// Delete all goals belonging to the project
		for _, goal := range goals {
			err = s.repo.DeleteGoal(txCtx, goal.ID)
			if err != nil {
				return err
			}
		}

		// Delete the project itself
		err = s.repo.DeleteProject(txCtx, projectID)
		if err != nil {
			return err
		}

		return nil
	})
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

func (s *Service) GetProject(ctx context.Context, projectID string) (*Project, error) {
	return s.repo.GetProject(ctx, projectID)
}

func (s *Service) UpdateProject(ctx context.Context, id string, name string, description string) (*Project, error) {
	if id == "" {
		return nil, fmt.Errorf("%w - project ID", ErrMissingRequiredFields)
	}

	var result *Project
	err := s.repo.Transaction(ctx, func(txCtx context.Context) error {
		current, err := s.repo.GetProject(txCtx, id)
		if err != nil {
			return err
		}

		if name != "" {
			current.Label = name
		}
		if description != "" {
			current.Description = description
		}

		p, err := s.repo.UpdateProject(txCtx, current)
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

// MergeProjects merges the mergeProject into the sourceProject. All goals of the mergeProject
// are transferred under the specified parentGoalID of the sourceProject (or the sourceProject's root goal if parentGoalID is empty).
// The mergeProject is then deleted.
func (s *Service) MergeProjects(ctx context.Context, sourceProjectID string, mergeProjectID string, parentGoalID string) error {
	if sourceProjectID == "" || mergeProjectID == "" {
		return fmt.Errorf("%w - source and merge project IDs are required", ErrMissingRequiredFields)
	}
	if sourceProjectID == mergeProjectID {
		return fmt.Errorf("%w - cannot merge project with itself", ErrInvalidRequest)
	}

	return s.repo.Transaction(ctx, func(txCtx context.Context) error {
		sourceProj, err := s.repo.GetProject(txCtx, sourceProjectID)
		if err != nil {
			return err
		}
		mergeProj, err := s.repo.GetProject(txCtx, mergeProjectID)
		if err != nil {
			return err
		}

		mergeRootGoalID := mergeProj.RootGoalID
		if mergeRootGoalID != "" {
			mergeGoals, err := s.repo.GetGoals(txCtx, mergeProjectID)
			if err != nil {
				return err
			}

			var mergeRoot *Goal
			idx := slices.IndexFunc(mergeGoals, func(a Goal) bool {
				return a.ID == mergeRootGoalID
			})
			if idx == -1 {
				return fmt.Errorf("root goal of merge project not found in goals")
			}
			mergeRoot = &mergeGoals[idx]

			var parentGoalIDToUse string
			if parentGoalID != "" {
				parentGoalIDToUse = parentGoalID
			} else {
				parentGoalIDToUse = sourceProj.RootGoalID
			}

			if parentGoalIDToUse != "" {
				parentGoal, err := s.repo.GetGoal(txCtx, parentGoalIDToUse)
				if err != nil {
					return err
				}
				if parentGoal.ProjectID != sourceProjectID {
					return fmt.Errorf("%w - parent goal does not belong to the source project", ErrInvalidRequest)
				}

				err = parentGoal.AttachChild(mergeRoot, LinkOptional, "Merged from project: "+mergeProj.Label)
				if err != nil {
					return err
				}

				parentGoal, err = s.repo.UpdateGoal(txCtx, parentGoal)
				if err != nil {
					return err
				}
			} else {
				sourceProj.SetRoot(*mergeRoot)
				_, err = s.repo.UpdateProject(txCtx, sourceProj)
				if err != nil {
					return err
				}
			}

			for i := range mergeGoals {
				mergeGoals[i].ProjectID = sourceProjectID
				_, err = s.repo.UpdateGoal(txCtx, &mergeGoals[i])
				if err != nil {
					return err
				}
			}
		}

		err = s.repo.DeleteProject(txCtx, mergeProjectID)
		if err != nil {
			return err
		}

		return nil
	})
}

// SplitProject splits a sourceProject starting from a splitGoalID, either creating a new project with the provided newProjectName, or moving it to an existing targetProjectID.
// The splitGoalID and all its descendants are moved to the target project.
// If newProjectName is provided, a new project with that name is created first.
// If targetProjectID is provided, it moves the goal to that existing project.
// If targetParentGoalID is provided, the moved goal is attached under it. Otherwise, it is attached under the target project's root goal.
// TODO: Will come back to clean this up. It can be better organized.
func (s *Service) SplitProject(ctx context.Context, sourceProjectID string, targetProjectID string, splitGoalID string, targetParentGoalID string, newProjectName string) (*Project, error) {
	if sourceProjectID == "" || splitGoalID == "" {
		return nil, fmt.Errorf("%w - source project ID and split goal ID are required", ErrMissingRequiredFields)
	}
	if targetProjectID == "" && newProjectName == "" {
		return nil, fmt.Errorf("%w - target project ID or new project name is required", ErrMissingRequiredFields)
	}
	if targetProjectID != "" && sourceProjectID == targetProjectID {
		return nil, fmt.Errorf("%w - source and target projects must be different", ErrInvalidRequest)
	}

	var result *Project
	err := s.repo.Transaction(ctx, func(txCtx context.Context) error {
		var targetProjIDToUse string

		if newProjectName != "" {
			projects, err := s.repo.GetProjects(txCtx)
			if err != nil {
				return err
			}

			newProj := &Project{
				Label: newProjectName,
			}
			label := newProj.NormalizedLabel()
			for _, p := range projects {
				if p.NormalizedLabel() == label {
					return fmt.Errorf("project %s: %w", label, ErrProjectAlreadyExists)
				}
			}

			newProj, err = s.repo.CreateProject(txCtx, newProj)
			if err != nil {
				return err
			}
			targetProjIDToUse = newProj.ID
		} else {
			targetProjIDToUse = targetProjectID
		}

		sourceProj, err := s.repo.GetProject(txCtx, sourceProjectID)
		if err != nil {
			return err
		}

		targetProj, err := s.repo.GetProject(txCtx, targetProjIDToUse)
		if err != nil {
			return err
		}

		splitGoal, err := s.repo.GetGoal(txCtx, splitGoalID)
		if err != nil {
			return err
		}
		if splitGoal.ProjectID != sourceProjectID {
			return fmt.Errorf("%w - split goal does not belong to the source project", ErrInvalidRequest)
		}

		// Detach from current parent in source project
		if splitGoal.HasParent() {
			parentGoal, err := s.repo.GetGoal(txCtx, splitGoal.Parent.TargetID)
			if err != nil {
				return err
			}
			err = parentGoal.DetachChild(splitGoal)
			if err != nil {
				return err
			}
			_, err = s.repo.UpdateGoal(txCtx, parentGoal)
			if err != nil {
				return err
			}
			splitGoal.Parent = nil
		} else if splitGoalID == sourceProj.RootGoalID {
			sourceProj.RootGoalID = ""
			_, err = s.repo.UpdateProject(txCtx, sourceProj)
			if err != nil {
				return err
			}
		}

		// Attach to parent in target project
		var parentGoalIDToUse string
		if targetParentGoalID != "" {
			parentGoalIDToUse = targetParentGoalID
		} else {
			parentGoalIDToUse = targetProj.RootGoalID
		}

		if parentGoalIDToUse != "" {
			parentGoal, err := s.repo.GetGoal(txCtx, parentGoalIDToUse)
			if err != nil {
				return err
			}
			if parentGoal.ProjectID != targetProjIDToUse {
				return fmt.Errorf("%w - target parent goal does not belong to the target project", ErrInvalidRequest)
			}

			// Pre-set the ProjectID to target project so the copy attached to parentGoal has the right project ID
			splitGoal.ProjectID = targetProj.ID

			err = parentGoal.AttachChild(splitGoal, LinkOptional, "Moved from project: "+sourceProj.Label)
			if err != nil {
				return err
			}

			_, err = s.repo.UpdateGoal(txCtx, parentGoal)
			if err != nil {
				return err
			}
		} else {
			splitGoal.ProjectID = targetProj.ID
			targetProj.SetRoot(*splitGoal)
			targetProj, err = s.repo.UpdateProject(txCtx, targetProj)
			if err != nil {
				return err
			}
		}

		sourceGoals, err := s.repo.GetGoals(txCtx, sourceProjectID)
		if err != nil {
			return err
		}

		parentToChildren := make(map[string][]Goal)
		for _, g := range sourceGoals {
			if g.HasParent() {
				parentToChildren[g.Parent.TargetID] = append(parentToChildren[g.Parent.TargetID], g)
			}
		}

		var collectDescendants func(id string) []Goal
		collectDescendants = func(id string) []Goal {
			var list []Goal
			for _, child := range parentToChildren[id] {
				list = append(list, child)
				list = append(list, collectDescendants(child.ID)...)
			}
			return list
		}

		descendants := collectDescendants(splitGoalID)
		allMovedGoals := append([]Goal{*splitGoal}, descendants...)

		for _, g := range allMovedGoals {
			g.ProjectID = targetProj.ID
			for i := range g.Children {
				g.Children[i].ProjectID = targetProj.ID
			}
			_, err = s.repo.UpdateGoal(txCtx, &g)
			if err != nil {
				return err
			}
		}

		result = targetProj
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
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

func (s *Service) DeleteGoal(ctx context.Context, projectID string, goalID string) error {
	if projectID == "" {
		return fmt.Errorf("%w - project ID", ErrMissingRequiredFields)
	}
	if goalID == "" {
		return fmt.Errorf("%w - goal ID", ErrMissingRequiredFields)
	}

	return s.repo.Transaction(ctx, func(txCtx context.Context) error {
		project, err := s.repo.GetProject(txCtx, projectID)
		if err != nil {
			return err
		}

		goal, err := s.repo.GetGoal(txCtx, goalID)
		if err != nil {
			return err
		}
		if goal.ProjectID != projectID {
			return fmt.Errorf("goal %s does not belong to project %s: %w", goalID, projectID, ErrInvalidRequest)
		}
		if goalID == project.RootGoalID || goal.IsRoot() {
			return ErrCannotDeleteRoot
		}
		if len(goal.Children) > 0 {
			return fmt.Errorf("goal %s has children: %w", goalID, ErrInvalidRequest)
		}

		if goal.HasParent() {
			parent, err := s.repo.GetGoal(txCtx, goal.Parent.TargetID)
			if err != nil {
				return err
			}
			if err := parent.DetachChild(goal); err != nil {
				return err
			}
			if _, err := s.repo.UpdateGoal(txCtx, parent); err != nil {
				return err
			}
		}

		return s.repo.DeleteGoal(txCtx, goalID)
	})
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

func (s *Service) Export(ctx context.Context, filepath string) error {
	if filepath == "" {
		return fmt.Errorf("%w - filepath", ErrMissingRequiredFields)
	}
	return s.repo.Export(ctx, filepath)
}

func (s *Service) Import(ctx context.Context, filepath string) error {
	if filepath == "" {
		return fmt.Errorf("%w - filepath", ErrMissingRequiredFields)
	}
	return s.repo.Import(ctx, filepath)
}
