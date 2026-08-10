package core

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockTrackerRepository struct {
	transactionFunc   func(ctx context.Context, fn func(ctx context.Context) error) error
	createProjectFunc func(ctx context.Context, project *Project) (*Project, error)
	getProjectsFunc   func(ctx context.Context) ([]Project, error)
	getProjectFunc    func(ctx context.Context, projectID string) (*Project, error)
	updateProjectFunc func(ctx context.Context, project *Project) (*Project, error)
	deleteProjectFunc func(ctx context.Context, projectID string) error
	createGoalFunc    func(ctx context.Context, projectID string, goal *Goal) (*Goal, error)
	getGoalsFunc      func(ctx context.Context, projectID string) ([]Goal, error)
	getGoalFunc       func(ctx context.Context, goalID string) (*Goal, error)
	updateGoalFunc    func(ctx context.Context, goal *Goal) (*Goal, error)
	queryGoalsFunc    func(ctx context.Context, projectID string, query *GoalQuery) ([]Goal, error)
	createMemberFunc  func(ctx context.Context, member *Member) (*Member, error)
	getMemberFunc     func(ctx context.Context, memberID string) (*Member, error)
	getMembersFunc    func(ctx context.Context) ([]Member, error)
	updateMemberFunc  func(ctx context.Context, member *Member) (*Member, error)
	deleteMemberFunc  func(ctx context.Context, memberID string) error
	exportFunc        func(ctx context.Context, filepath string) error
	importFunc        func(ctx context.Context, filepath string) error
}

func (m *mockTrackerRepository) CreateProject(ctx context.Context, project *Project) (*Project, error) {
	if m.createProjectFunc != nil {
		return m.createProjectFunc(ctx, project)
	}
	return nil, nil
}

func (m *mockTrackerRepository) GetProjects(ctx context.Context) ([]Project, error) {
	if m.getProjectsFunc != nil {
		return m.getProjectsFunc(ctx)
	}
	return nil, nil
}

func (m *mockTrackerRepository) GetProject(ctx context.Context, projectID string) (*Project, error) {
	if m.getProjectFunc != nil {
		return m.getProjectFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *mockTrackerRepository) UpdateProject(ctx context.Context, project *Project) (*Project, error) {
	if m.updateProjectFunc != nil {
		return m.updateProjectFunc(ctx, project)
	}
	return nil, nil
}

func (m *mockTrackerRepository) DeleteProject(ctx context.Context, projectID string) error {
	if m.deleteProjectFunc != nil {
		return m.deleteProjectFunc(ctx, projectID)
	}
	return nil
}

func (m *mockTrackerRepository) CreateGoal(ctx context.Context, projectID string, goal *Goal) (*Goal, error) {
	if m.createGoalFunc != nil {
		return m.createGoalFunc(ctx, projectID, goal)
	}
	return nil, nil
}

func (m *mockTrackerRepository) GetGoals(ctx context.Context, projectID string) ([]Goal, error) {
	if m.getGoalsFunc != nil {
		return m.getGoalsFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *mockTrackerRepository) GetGoal(ctx context.Context, goalID string) (*Goal, error) {
	if m.getGoalFunc != nil {
		return m.getGoalFunc(ctx, goalID)
	}
	return nil, nil
}

func (m *mockTrackerRepository) UpdateGoal(ctx context.Context, goal *Goal) (*Goal, error) {
	if m.updateGoalFunc != nil {
		return m.updateGoalFunc(ctx, goal)
	}
	return nil, nil
}

func (m *mockTrackerRepository) QueryGoals(ctx context.Context, projectID string, query *GoalQuery) ([]Goal, error) {
	if m.queryGoalsFunc != nil {
		return m.queryGoalsFunc(ctx, projectID, query)
	}
	return nil, nil
}

func (m *mockTrackerRepository) CreateMember(ctx context.Context, member *Member) (*Member, error) {
	if m.createMemberFunc != nil {
		return m.createMemberFunc(ctx, member)
	}
	return nil, nil
}

func (m *mockTrackerRepository) GetMember(ctx context.Context, memberID string) (*Member, error) {
	if m.getMemberFunc != nil {
		return m.getMemberFunc(ctx, memberID)
	}
	return nil, nil
}

func (m *mockTrackerRepository) GetMembers(ctx context.Context) ([]Member, error) {
	if m.getMembersFunc != nil {
		return m.getMembersFunc(ctx)
	}
	return nil, nil
}

func (m *mockTrackerRepository) UpdateMember(ctx context.Context, member *Member) (*Member, error) {
	if m.updateMemberFunc != nil {
		return m.updateMemberFunc(ctx, member)
	}
	return nil, nil
}

func (m *mockTrackerRepository) DeleteMember(ctx context.Context, memberID string) error {
	if m.deleteMemberFunc != nil {
		return m.deleteMemberFunc(ctx, memberID)
	}
	return nil
}

func (m *mockTrackerRepository) Export(ctx context.Context, filepath string) error {
	if m.exportFunc != nil {
		return m.exportFunc(ctx, filepath)
	}
	return nil
}

func (m *mockTrackerRepository) Import(ctx context.Context, filepath string) error {
	if m.importFunc != nil {
		return m.importFunc(ctx, filepath)
	}
	return nil
}

func (m *mockTrackerRepository) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.transactionFunc != nil {
		return m.transactionFunc(ctx, fn)
	}
	return fn(ctx)
}

func TestServiceCreateProject(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		repo := &mockTrackerRepository{
			getProjectsFunc: func(ctx context.Context) ([]Project, error) {
				return []Project{{Label: "Existing Project"}}, nil
			},
			createProjectFunc: func(ctx context.Context, project *Project) (*Project, error) {
				project.ID = "p-123"
				return project, nil
			},
		}

		s := NewService(repo)
		p, err := s.CreateProject(context.Background(), "New Project", "A cool description")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if p.ID != "p-123" {
			t.Errorf("expected ID 'p-123', got %q", p.ID)
		}
		if p.Label != "New Project" {
			t.Errorf("expected Label 'New Project', got %q", p.Label)
		}
	})

	t.Run("empty name error", func(t *testing.T) {
		s := NewService(&mockTrackerRepository{})
		_, err := s.CreateProject(context.Background(), "", "")
		if !errors.Is(err, ErrInvalidName) {
			t.Errorf("expected error %v, got %v", ErrInvalidName, err)
		}
	})

	t.Run("duplicate name error", func(t *testing.T) {
		repo := &mockTrackerRepository{
			getProjectsFunc: func(ctx context.Context) ([]Project, error) {
				return []Project{{Label: "Existing Project"}}, nil
			},
		}
		s := NewService(repo)
		_, err := s.CreateProject(context.Background(), "  existing project  ", "")
		if !errors.Is(err, ErrProjectAlreadyExists) {
			t.Errorf("expected error %v, got %v", ErrProjectAlreadyExists, err)
		}
	})

	t.Run("repo errors propagated", func(t *testing.T) {
		expectedErr := errors.New("db error")
		repo := &mockTrackerRepository{
			getProjectsFunc: func(ctx context.Context) ([]Project, error) {
				return nil, expectedErr
			},
		}
		s := NewService(repo)
		_, err := s.CreateProject(context.Background(), "Project X", "")
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestServiceCreateGoal(t *testing.T) {
	t.Run("validation missing fields", func(t *testing.T) {
		s := NewService(&mockTrackerRepository{})

		_, err := s.CreateGoal(context.Background(), "", "Goal Name", GoalOptions{})
		if !errors.Is(err, ErrMissingRequiredFields) {
			t.Errorf("expected missing project ID error, got %v", err)
		}

		_, err = s.CreateGoal(context.Background(), "p-1", "", GoalOptions{})
		if !errors.Is(err, ErrMissingRequiredFields) {
			t.Errorf("expected missing goal name error, got %v", err)
		}
	})

	t.Run("no orphan goals allowed", func(t *testing.T) {
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, id string) (*Project, error) {
				return &Project{ID: "p-1", RootGoalID: "root-id"}, nil
			},
		}
		s := NewService(repo)

		_, err := s.CreateGoal(context.Background(), "p-1", "Orphan Goal", GoalOptions{})
		if err == nil || err.Error() != "no orphan goals allowed" {
			t.Errorf("expected orphan goal error, got %v", err)
		}
	})

	t.Run("already has root error", func(t *testing.T) {
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, id string) (*Project, error) {
				return &Project{ID: "p-1", RootGoalID: "root-id"}, nil
			},
		}
		s := NewService(repo)

		_, err := s.CreateGoal(context.Background(), "p-1", "Duplicate Root", GoalOptions{ParentID: ""})
		if err == nil || err.Error() != "no orphan goals allowed" {
			t.Errorf("expected 'no orphan goals allowed' error, got %v", err)
		}
	})

	t.Run("successful root goal creation", func(t *testing.T) {
		var updatedProject *Project
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, id string) (*Project, error) {
				return &Project{ID: "p-1", RootGoalID: ""}, nil
			},
			createGoalFunc: func(ctx context.Context, projectID string, goal *Goal) (*Goal, error) {
				goal.ID = "goal-root"
				return goal, nil
			},
			updateProjectFunc: func(ctx context.Context, p *Project) (*Project, error) {
				updatedProject = p
				return p, nil
			},
		}
		s := NewService(repo)

		g, err := s.CreateGoal(context.Background(), "p-1", "Root Goal", GoalOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if g.ID != "goal-root" {
			t.Errorf("expected goal ID 'goal-root', got %q", g.ID)
		}
		if updatedProject == nil || updatedProject.RootGoalID != "goal-root" {
			t.Errorf("expected project RootGoalID to be updated to 'goal-root'")
		}
	})

	t.Run("successful child goal creation with member", func(t *testing.T) {
		var updatedParent *Goal
		var updatedChild *Goal
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, id string) (*Project, error) {
				return &Project{ID: "p-1", RootGoalID: "goal-root"}, nil
			},
			getMemberFunc: func(ctx context.Context, id string) (*Member, error) {
				return &Member{ID: "m-1", Name: "Bob"}, nil
			},
			createGoalFunc: func(ctx context.Context, projectID string, goal *Goal) (*Goal, error) {
				goal.ID = "goal-child"
				return goal, nil
			},
			getGoalFunc: func(ctx context.Context, id string) (*Goal, error) {
				if id == "goal-root" {
					return &Goal{ID: "goal-root"}, nil
				}
				return nil, errors.New("not found")
			},
			updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
				if g.ID == "goal-root" {
					updatedParent = g
				} else if g.ID == "goal-child" {
					updatedChild = g
				}
				return g, nil
			},
		}
		s := NewService(repo)

		g, err := s.CreateGoal(context.Background(), "p-1", "Child Goal", GoalOptions{
			ParentID: "goal-root",
			MemberID: "m-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if g.ID != "goal-child" {
			t.Errorf("expected child ID 'goal-child', got %q", g.ID)
		}
		if g.Member == nil || g.Member.ID != "m-1" {
			t.Errorf("expected member ID 'm-1' assigned to child, got %v", g.Member)
		}

		if updatedParent == nil || len(updatedParent.Children) != 1 || updatedParent.Children[0].ID != "goal-child" {
			t.Errorf("expected parent to have attached child")
		}

		if updatedChild == nil || updatedChild.Parent == nil || updatedChild.Parent.TargetID != "goal-root" {
			t.Errorf("expected child to point to parent")
		}
	})
}

func TestServiceAttachMember(t *testing.T) {
	var updatedGoal *Goal
	repo := &mockTrackerRepository{
		getProjectFunc: func(ctx context.Context, id string) (*Project, error) {
			return &Project{ID: "p-1"}, nil
		},
		getGoalFunc: func(ctx context.Context, id string) (*Goal, error) {
			return &Goal{ID: "g-1"}, nil
		},
		getMemberFunc: func(ctx context.Context, id string) (*Member, error) {
			return &Member{ID: "m-1", Name: "Dave"}, nil
		},
		updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
			updatedGoal = g
			return g, nil
		},
	}
	s := NewService(repo)

	g, err := s.AttachMember(context.Background(), "p-1", "g-1", "m-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Member == nil || g.Member.ID != "m-1" {
		t.Errorf("expected member ID 'm-1', got %v", g.Member)
	}
	if updatedGoal == nil || updatedGoal.Member == nil || updatedGoal.Member.ID != "m-1" {
		t.Errorf("expected repository update to include assigned member")
	}
}

func TestServiceGetProjects(t *testing.T) {
	repo := &mockTrackerRepository{
		getProjectsFunc: func(ctx context.Context) ([]Project, error) {
			return []Project{
				{Label: "C Project"},
				{Label: "A Project"},
				{Label: "B Project"},
			}, nil
		},
	}
	s := NewService(repo)

	projects, err := s.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"A Project", "B Project", "C Project"}
	var got []string
	for _, p := range projects {
		got = append(got, p.Label)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("GetProjects() sorted order = %v; want %v", got, expected)
	}
}

func TestServiceGetGoals(t *testing.T) {
	repo := &mockTrackerRepository{
		getGoalsFunc: func(ctx context.Context, pid string) ([]Goal, error) {
			return []Goal{
				{Name: "Beta"},
				{Name: "Alpha"},
				{Name: "Gamma"},
			}, nil
		},
	}
	s := NewService(repo)

	goals, err := s.GetGoals(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"Alpha", "Beta", "Gamma"}
	var got []string
	for _, g := range goals {
		got = append(got, g.Name)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("GetGoals() sorted order = %v; want %v", got, expected)
	}
}

func TestServiceChangeParent(t *testing.T) {
	t.Run("invalid project", func(t *testing.T) {
		expectedErr := errors.New("project not found")
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return nil, expectedErr
			},
		}
		s := NewService(repo)
		_, err := s.ChangeParent(context.Background(), "p-invalid", "g-1", "g-2", GoalOptions{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("cannot move root goal", func(t *testing.T) {
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID, RootGoalID: "g-root"}, nil
			},
		}
		s := NewService(repo)
		_, err := s.ChangeParent(context.Background(), "p-1", "g-root", "g-2", GoalOptions{})
		if !errors.Is(err, ErrCannotMoveRoot) {
			t.Errorf("expected error %v, got %v", ErrCannotMoveRoot, err)
		}
	})

	t.Run("invalid child goal", func(t *testing.T) {
		expectedErr := errors.New("child goal not found")
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				return nil, expectedErr
			},
		}
		s := NewService(repo)
		_, err := s.ChangeParent(context.Background(), "p-1", "g-invalid", "g-2", GoalOptions{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("invalid new parent goal", func(t *testing.T) {
		expectedErr := errors.New("new parent goal not found")
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				if goalID == "g-child" {
					return &Goal{ID: "g-child"}, nil
				}
				return nil, expectedErr
			},
		}
		s := NewService(repo)
		_, err := s.ChangeParent(context.Background(), "p-1", "g-child", "g-invalid", GoalOptions{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("successful parent change - no existing parent", func(t *testing.T) {
		var updatedChild *Goal
		var updatedNewParent *Goal
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				if goalID == "g-child" {
					return &Goal{ID: "g-child"}, nil
				}
				if goalID == "g-new-parent" {
					return &Goal{ID: "g-new-parent"}, nil
				}
				return nil, errors.New("not found")
			},
			updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
				if g.ID == "g-child" {
					updatedChild = g
				} else if g.ID == "g-new-parent" {
					updatedNewParent = g
				}
				return g, nil
			},
		}
		s := NewService(repo)
		g, err := s.ChangeParent(context.Background(), "p-1", "g-child", "g-new-parent", GoalOptions{
			LinkRelationship: LinkRequired,
			Description:      "Required dependency",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if g.ID != "g-child" {
			t.Errorf("expected returned goal to be g-child, got %q", g.ID)
		}
		if updatedChild == nil || updatedChild.Parent == nil || updatedChild.Parent.TargetID != "g-new-parent" {
			t.Errorf("expected child's parent to be updated to 'g-new-parent', got %v", updatedChild)
		}
		if updatedChild.Parent.Relationship != LinkRequired {
			t.Errorf("expected relationship to be 'required', got %v", updatedChild.Parent.Relationship)
		}
		if updatedChild.Parent.Description != "Required dependency" {
			t.Errorf("expected description to be 'Required dependency', got %q", updatedChild.Parent.Description)
		}
		if updatedNewParent == nil || len(updatedNewParent.Children) != 1 || updatedNewParent.Children[0].ID != "g-child" {
			t.Errorf("expected new parent to have attached child, got %v", updatedNewParent)
		}
	})

	t.Run("successful parent change - with existing parent", func(t *testing.T) {
		var updatedOldParent *Goal
		var updatedChild *Goal
		var updatedNewParent *Goal
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				if goalID == "g-child" {
					return &Goal{
						ID: "g-child",
						Parent: &Link{
							TargetID: "g-old-parent",
						},
					}, nil
				}
				if goalID == "g-old-parent" {
					return &Goal{
						ID: "g-old-parent",
						Children: []Goal{
							{ID: "g-child"},
						},
					}, nil
				}
				if goalID == "g-new-parent" {
					return &Goal{ID: "g-new-parent"}, nil
				}
				return nil, errors.New("not found")
			},
			updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
				if g.ID == "g-old-parent" {
					updatedOldParent = g
				} else if g.ID == "g-child" {
					updatedChild = g
				} else if g.ID == "g-new-parent" {
					updatedNewParent = g
				}
				return g, nil
			},
		}
		s := NewService(repo)
		g, err := s.ChangeParent(context.Background(), "p-1", "g-child", "g-new-parent", GoalOptions{
			LinkRelationship: LinkOptional,
			Description:      "Optional dependency",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if g.ID != "g-child" {
			t.Errorf("expected returned goal to be g-child, got %q", g.ID)
		}
		if updatedOldParent == nil || len(updatedOldParent.Children) != 0 {
			t.Errorf("expected old parent to have detached child, got %v", updatedOldParent)
		}
		if updatedChild == nil || updatedChild.Parent == nil || updatedChild.Parent.TargetID != "g-new-parent" {
			t.Errorf("expected child's parent to be updated to 'g-new-parent', got %v", updatedChild)
		}
		if updatedNewParent == nil || len(updatedNewParent.Children) != 1 || updatedNewParent.Children[0].ID != "g-child" {
			t.Errorf("expected new parent to have attached child, got %v", updatedNewParent)
		}
	})

	t.Run("error detaching/updating old parent", func(t *testing.T) {
		expectedErr := errors.New("update old parent error")
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				if goalID == "g-child" {
					return &Goal{
						ID: "g-child",
						Parent: &Link{
							TargetID: "g-old-parent",
						},
					}, nil
				}
				if goalID == "g-old-parent" {
					return &Goal{
						ID: "g-old-parent",
						Children: []Goal{
							{ID: "g-child"},
						},
					}, nil
				}
				if goalID == "g-new-parent" {
					return &Goal{ID: "g-new-parent"}, nil
				}
				return nil, errors.New("not found")
			},
			updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
				if g.ID == "g-old-parent" {
					return nil, expectedErr
				}
				return g, nil
			},
		}
		s := NewService(repo)
		_, err := s.ChangeParent(context.Background(), "p-1", "g-child", "g-new-parent", GoalOptions{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("error updating child", func(t *testing.T) {
		expectedErr := errors.New("update child error")
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				if goalID == "g-child" {
					return &Goal{ID: "g-child"}, nil
				}
				if goalID == "g-new-parent" {
					return &Goal{ID: "g-new-parent"}, nil
				}
				return nil, errors.New("not found")
			},
			updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
				if g.ID == "g-child" {
					return nil, expectedErr
				}
				return g, nil
			},
		}
		s := NewService(repo)
		_, err := s.ChangeParent(context.Background(), "p-1", "g-child", "g-new-parent", GoalOptions{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("error updating new parent", func(t *testing.T) {
		expectedErr := errors.New("update new parent error")
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				if goalID == "g-child" {
					return &Goal{ID: "g-child"}, nil
				}
				if goalID == "g-new-parent" {
					return &Goal{ID: "g-new-parent"}, nil
				}
				return nil, errors.New("not found")
			},
			updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
				if g.ID == "g-new-parent" {
					return nil, expectedErr
				}
				return g, nil
			},
		}
		s := NewService(repo)
		_, err := s.ChangeParent(context.Background(), "p-1", "g-child", "g-new-parent", GoalOptions{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("move to same parent does nothing and preserves parent link", func(t *testing.T) {
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				if goalID == "g-child" {
					return &Goal{
						ID: "g-child",
						Parent: &Link{
							TargetID: "g-parent",
						},
					}, nil
				}
				if goalID == "g-parent" {
					return &Goal{
						ID: "g-parent",
						Children: []Goal{
							{ID: "g-child"},
						},
					}, nil
				}
				return nil, errors.New("not found")
			},
			updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
				return g, nil
			},
		}
		s := NewService(repo)
		child, err := s.ChangeParent(context.Background(), "p-1", "g-child", "g-parent", GoalOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if child.Parent == nil || child.Parent.TargetID != "g-parent" {
			t.Errorf("expected parent link to remain 'g-parent', got %v", child.Parent)
		}
	})

	t.Run("moved child parent link inside parent Children slice is correct", func(t *testing.T) {
		var updatedParent *Goal
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, projectID string) (*Project, error) {
				return &Project{ID: projectID}, nil
			},
			getGoalFunc: func(ctx context.Context, goalID string) (*Goal, error) {
				if goalID == "g-child" {
					return &Goal{
						ID: "g-child",
					}, nil
				}
				if goalID == "g-new-parent" {
					return &Goal{
						ID: "g-new-parent",
					}, nil
				}
				return nil, errors.New("not found")
			},
			updateGoalFunc: func(ctx context.Context, g *Goal) (*Goal, error) {
				if g.ID == "g-new-parent" {
					updatedParent = g
				}
				return g, nil
			},
		}
		s := NewService(repo)
		_, err := s.ChangeParent(context.Background(), "p-1", "g-child", "g-new-parent", GoalOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updatedParent == nil {
			t.Fatal("expected parent to be updated")
		}
		if len(updatedParent.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(updatedParent.Children))
		}
		childInParent := updatedParent.Children[0]
		if childInParent.Parent == nil || childInParent.Parent.TargetID != "g-new-parent" {
			t.Errorf("expected child inside parent Children to have parent link target 'g-new-parent', got %v", childInParent.Parent)
		}
	})
}

func TestMemberCRUD(t *testing.T) {
	t.Run("CreateMember successful", func(t *testing.T) {
		repo := &mockTrackerRepository{
			createMemberFunc: func(ctx context.Context, m *Member) (*Member, error) {
				m.ID = "m-test"
				return m, nil
			},
		}
		s := NewService(repo)
		m, err := s.CreateMember(context.Background(), "Bob", "bob@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.ID != "m-test" || m.Name != "Bob" || m.Email != "bob@example.com" {
			t.Errorf("unexpected created member: %+v", m)
		}
	})

	t.Run("CreateMember missing name", func(t *testing.T) {
		s := NewService(&mockTrackerRepository{})
		_, err := s.CreateMember(context.Background(), "", "bob@example.com")
		if !errors.Is(err, ErrMissingRequiredFields) {
			t.Errorf("expected ErrMissingRequiredFields, got %v", err)
		}
	})

	t.Run("GetMember successful", func(t *testing.T) {
		repo := &mockTrackerRepository{
			getMemberFunc: func(ctx context.Context, id string) (*Member, error) {
				return &Member{ID: id, Name: "Bob"}, nil
			},
		}
		s := NewService(repo)
		m, err := s.GetMember(context.Background(), "m-test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.ID != "m-test" || m.Name != "Bob" {
			t.Errorf("unexpected member: %+v", m)
		}
	})

	t.Run("GetMembers successful", func(t *testing.T) {
		repo := &mockTrackerRepository{
			getMembersFunc: func(ctx context.Context) ([]Member, error) {
				return []Member{{ID: "m-1", Name: "Bob"}, {ID: "m-2", Name: "Alice"}}, nil
			},
		}
		s := NewService(repo)
		members, err := s.GetMembers(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(members) != 2 {
			t.Errorf("expected 2 members, got %d", len(members))
		}
	})

	t.Run("UpdateMember successful", func(t *testing.T) {
		repo := &mockTrackerRepository{
			updateMemberFunc: func(ctx context.Context, m *Member) (*Member, error) {
				return m, nil
			},
		}
		s := NewService(repo)
		m, err := s.UpdateMember(context.Background(), &Member{ID: "m-test", Name: "Bob Updated", Email: "bob2@example.com"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Name != "Bob Updated" || m.Email != "bob2@example.com" {
			t.Errorf("unexpected updated member: %+v", m)
		}
	})

	t.Run("DeleteMember successful", func(t *testing.T) {
		deletedID := ""
		repo := &mockTrackerRepository{
			deleteMemberFunc: func(ctx context.Context, id string) error {
				deletedID = id
				return nil
			},
		}
		s := NewService(repo)
		err := s.DeleteMember(context.Background(), "m-test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deletedID != "m-test" {
			t.Errorf("expected deleted ID 'm-test', got '%s'", deletedID)
		}
	})
}

func TestServiceUpdateGoal(t *testing.T) {
	t.Run("successful goal update", func(t *testing.T) {
		targetGoal := &Goal{ID: "g-test", Name: "Old Goal", Description: "Old Desc", Effort: UnknownEffort}
		repo := &mockTrackerRepository{
			getGoalFunc: func(ctx context.Context, id string) (*Goal, error) {
				return targetGoal, nil
			},
			getMemberFunc: func(ctx context.Context, id string) (*Member, error) {
				return &Member{ID: id, Name: "Assignee"}, nil
			},
			updateGoalFunc: func(ctx context.Context, goal *Goal) (*Goal, error) {
				return goal, nil
			},
		}
		s := NewService(repo)
		options := GoalOptions{
			Description: "New Desc",
			Effort:      LowEffort,
			MemberID:    "m-test",
		}
		updated, err := s.UpdateGoal(context.Background(), "g-test", "New Name", options)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "New Name" || updated.Description != "New Desc" || updated.Effort != LowEffort || updated.Member.ID != "m-test" {
			t.Errorf("unexpected updated goal: %+v", updated)
		}
	})
}

func TestServiceMergeProjects(t *testing.T) {
	t.Run("validation and self-merge errors", func(t *testing.T) {
		s := NewService(&mockTrackerRepository{})
		err := s.MergeProjects(context.Background(), "", "p-2", "")
		if !errors.Is(err, ErrMissingRequiredFields) {
			t.Errorf("expected ErrMissingRequiredFields, got %v", err)
		}
		err = s.MergeProjects(context.Background(), "p-1", "p-1", "")
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("expected ErrInvalidRequest, got %v", err)
		}
	})

	t.Run("successful merge with parent goal", func(t *testing.T) {
		sourceProj := &Project{ID: "p-src", Label: "Source Project", RootGoalID: "g-src-root"}
		mergeProj := &Project{ID: "p-mrg", Label: "Merge Project", RootGoalID: "g-mrg-root"}
		srcRootGoal := &Goal{ID: "g-src-root", ProjectID: "p-src", Name: "Src Root"}
		mrgRootGoal := &Goal{ID: "g-mrg-root", ProjectID: "p-mrg", Name: "Mrg Root", Status: GoalInProgress, Member: &Member{ID: "m-1"}}
		mrgChildGoal := &Goal{ID: "g-mrg-child", ProjectID: "p-mrg", Name: "Mrg Child", Status: GoalCompleted, Parent: &Link{TargetID: "g-mrg-root"}}

		goals := map[string]*Goal{
			"g-src-root":  srcRootGoal,
			"g-mrg-root":  mrgRootGoal,
			"g-mrg-child": mrgChildGoal,
		}

		var deletedProjectID string
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, id string) (*Project, error) {
				if id == "p-src" {
					return sourceProj, nil
				}
				if id == "p-mrg" {
					return mergeProj, nil
				}
				return nil, errors.New("not found")
			},
			getGoalFunc: func(ctx context.Context, id string) (*Goal, error) {
				if g, ok := goals[id]; ok {
					return g, nil
				}
				return nil, errors.New("not found")
			},
			getGoalsFunc: func(ctx context.Context, projectID string) ([]Goal, error) {
				var res []Goal
				for _, g := range goals {
					if g.ProjectID == projectID {
						res = append(res, *g)
					}
				}
				return res, nil
			},
			updateGoalFunc: func(ctx context.Context, goal *Goal) (*Goal, error) {
				goals[goal.ID] = goal
				return goal, nil
			},
			deleteProjectFunc: func(ctx context.Context, id string) error {
				deletedProjectID = id
				return nil
			},
		}

		s := NewService(repo)
		err := s.MergeProjects(context.Background(), "p-src", "p-mrg", "g-src-root")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify mergeRoot was attached as child of srcRootGoal
		updatedSrcRoot := goals["g-src-root"]
		updatedMrgRoot := goals["g-mrg-root"]
		updatedMrgChild := goals["g-mrg-child"]

		if len(updatedSrcRoot.Children) != 1 || updatedSrcRoot.Children[0].ID != "g-mrg-root" {
			t.Errorf("expected g-mrg-root to be child of g-src-root, got: %+v", updatedSrcRoot.Children)
		}
		if updatedMrgRoot.Parent == nil || updatedMrgRoot.Parent.TargetID != "g-src-root" {
			t.Errorf("expected parent of g-mrg-root to be g-src-root, got parent: %+v", updatedMrgRoot.Parent)
		}

		// Verify project IDs updated
		if updatedMrgRoot.ProjectID != "p-src" {
			t.Errorf("expected g-mrg-root ProjectID to be p-src, got %q", updatedMrgRoot.ProjectID)
		}
		if updatedMrgChild.ProjectID != "p-src" {
			t.Errorf("expected g-mrg-child ProjectID to be p-src, got %q", updatedMrgChild.ProjectID)
		}

		// Verify status and member preserved
		if updatedMrgRoot.Status != GoalInProgress {
			t.Errorf("expected status of g-mrg-root to be in-progress, got %q", updatedMrgRoot.Status)
		}
		if updatedMrgChild.Status != GoalCompleted {
			t.Errorf("expected status of g-mrg-child to be completed, got %q", updatedMrgChild.Status)
		}
		if updatedMrgRoot.Member == nil || updatedMrgRoot.Member.ID != "m-1" {
			t.Errorf("expected member of g-mrg-root to be preserved, got %+v", updatedMrgRoot.Member)
		}

		// Verify deleted project ID
		if deletedProjectID != "p-mrg" {
			t.Errorf("expected project p-mrg to be deleted, got %q", deletedProjectID)
		}
	})
}

func TestServiceSplitProject(t *testing.T) {
	t.Run("successful split on child goal", func(t *testing.T) {
		sourceProj := &Project{ID: "p-src", Label: "Source Project", RootGoalID: "g-src-root"}
		srcRootGoal := &Goal{ID: "g-src-root", ProjectID: "p-src", Name: "Src Root"}
		splitGoal := &Goal{ID: "g-split", ProjectID: "p-src", Name: "Split Goal", Status: GoalInProgress, Member: &Member{ID: "m-2"}, Parent: &Link{TargetID: "g-src-root"}}
		splitChild := &Goal{ID: "g-child", ProjectID: "p-src", Name: "Split Child", Status: GoalCompleted, Parent: &Link{TargetID: "g-split"}}

		// Root has child
		srcRootGoal.Children = append(srcRootGoal.Children, *splitGoal)

		goals := map[string]*Goal{
			"g-src-root": srcRootGoal,
			"g-split":    splitGoal,
			"g-child":    splitChild,
		}

		var createdProject *Project
		repo := &mockTrackerRepository{
			getProjectFunc: func(ctx context.Context, id string) (*Project, error) {
				if id == "p-src" {
					return sourceProj, nil
				}
				return nil, errors.New("not found")
			},
			getProjectsFunc: func(ctx context.Context) ([]Project, error) {
				return []Project{*sourceProj}, nil
			},
			createProjectFunc: func(ctx context.Context, p *Project) (*Project, error) {
				p.ID = "p-new"
				createdProject = p
				return p, nil
			},
			updateProjectFunc: func(ctx context.Context, p *Project) (*Project, error) {
				return p, nil
			},
			getGoalFunc: func(ctx context.Context, id string) (*Goal, error) {
				if g, ok := goals[id]; ok {
					return g, nil
				}
				return nil, errors.New("not found")
			},
			getGoalsFunc: func(ctx context.Context, projectID string) ([]Goal, error) {
				var res []Goal
				for _, g := range goals {
					if g.ProjectID == projectID {
						res = append(res, *g)
					}
				}
				return res, nil
			},
			updateGoalFunc: func(ctx context.Context, goal *Goal) (*Goal, error) {
				goals[goal.ID] = goal
				return goal, nil
			},
		}

		s := NewService(repo)
		newProj, err := s.SplitProject(context.Background(), "p-src", "g-split", "New Split Project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if createdProject == nil || createdProject.Label != "New Split Project" {
			t.Errorf("expected new project to be created with label 'New Split Project', got: %+v", createdProject)
		}

		if newProj.ID != "p-new" || newProj.Label != "New Split Project" {
			t.Errorf("unexpected created project: %+v", newProj)
		}

		// Check root goal of new project
		if newProj.RootGoalID != "g-split" {
			t.Errorf("expected root goal of new project to be g-split, got %q", newProj.RootGoalID)
		}

		updatedSrcRoot := goals["g-src-root"]
		updatedSplitGoal := goals["g-split"]
		updatedSplitChild := goals["g-child"]

		// Verify splitGoal was detached from srcRootGoal
		if len(updatedSrcRoot.Children) != 0 {
			t.Errorf("expected srcRootGoal to have 0 children, got %d", len(updatedSrcRoot.Children))
		}
		if updatedSplitGoal.Parent != nil {
			t.Errorf("expected splitGoal to have no parent, got %+v", updatedSplitGoal.Parent)
		}

		// Verify project IDs updated
		if updatedSplitGoal.ProjectID != "p-new" {
			t.Errorf("expected splitGoal ProjectID to be p-new, got %q", updatedSplitGoal.ProjectID)
		}
		if updatedSplitChild.ProjectID != "p-new" {
			t.Errorf("expected splitChild ProjectID to be p-new, got %q", updatedSplitChild.ProjectID)
		}

		// Verify status and member preserved
		if updatedSplitGoal.Status != GoalInProgress {
			t.Errorf("expected status of splitGoal to be in-progress, got %q", updatedSplitGoal.Status)
		}
		if updatedSplitChild.Status != GoalCompleted {
			t.Errorf("expected status of splitChild to be completed, got %q", updatedSplitChild.Status)
		}
		if updatedSplitGoal.Member == nil || updatedSplitGoal.Member.ID != "m-2" {
			t.Errorf("expected member of splitGoal to be preserved, got %+v", updatedSplitGoal.Member)
		}
	})
}
