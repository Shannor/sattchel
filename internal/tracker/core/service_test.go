package core

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockTrackerRepository struct {
	createProjectFunc func(ctx context.Context, project *Project) (*Project, error)
	getProjectsFunc   func(ctx context.Context) ([]Project, error)
	getProjectFunc    func(ctx context.Context, projectID string) (*Project, error)
	updateProjectFunc func(ctx context.Context, project *Project) (*Project, error)
	createGoalFunc    func(ctx context.Context, projectID string, goal *Goal) (*Goal, error)
	getGoalsFunc      func(ctx context.Context, projectID string) ([]Goal, error)
	getGoalFunc       func(ctx context.Context, goalID string) (*Goal, error)
	updateGoalFunc    func(ctx context.Context, goal *Goal) (*Goal, error)
	createMemberFunc  func(ctx context.Context, member *Member) (*Member, error)
	getMemberFunc     func(ctx context.Context, memberID string) (*Member, error)
	getMembersFunc    func(ctx context.Context) ([]Member, error)
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
			LinkRelationship: LinkPreferred,
			Description:      "Preferred dependency",
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
