package core

import (
	"testing"
)

func TestNewGoal(t *testing.T) {
	opts := GoalOptions{
		Description: "A test goal description",
		Status:      GoalInProgress,
		Impact:      HighImpact,
		Effort:      LowEffort,
		MemberID:    "member-1",
		ParentID:    "parent-1",
	}

	g := NewGoal("proj-123", "Finish Unit Tests", opts)

	if g.ProjectID != "proj-123" {
		t.Errorf("expected ProjectID 'proj-123', got %q", g.ProjectID)
	}
	if g.Name != "Finish Unit Tests" {
		t.Errorf("expected Name 'Finish Unit Tests', got %q", g.Name)
	}
	if g.Description != "A test goal description" {
		t.Errorf("expected Description 'A test goal description', got %q", g.Description)
	}
	if g.Status != GoalInProgress {
		t.Errorf("expected Status %q, got %q", GoalInProgress, g.Status)
	}
	if g.Impact != HighImpact {
		t.Errorf("expected Impact %q, got %q", HighImpact, g.Impact)
	}
	if g.Effort != LowEffort {
		t.Errorf("expected Effort %q, got %q", LowEffort, g.Effort)
	}
	if g.Member == nil || g.Member.ID != "member-1" {
		t.Errorf("expected Member.ID 'member-1', got %v", g.Member)
	}
	if g.Parent == nil || g.Parent.TargetID != "parent-1" {
		t.Errorf("expected Parent.TargetID 'parent-1', got %v", g.Parent)
	}
}

func TestNewGoalDefaults(t *testing.T) {
	g := NewGoal("proj-123", "Finish Unit Tests", GoalOptions{})

	if g.Status != GoalDraft {
		t.Errorf("expected default Status %q, got %q", GoalDraft, g.Status)
	}
	if g.Impact != UnknownImpact {
		t.Errorf("expected default Impact %q, got %q", UnknownImpact, g.Impact)
	}
	if g.Effort != UnknownEffort {
		t.Errorf("expected default Effort %q, got %q", UnknownEffort, g.Effort)
	}
	if g.Member != nil {
		t.Errorf("expected nil Member, got %v", g.Member)
	}
	if g.Parent != nil {
		t.Errorf("expected nil Parent, got %v", g.Parent)
	}
}

func TestAssignMember(t *testing.T) {
	g := Goal{}
	member := &Member{ID: "m-123", Name: "Alice"}
	g.AssignMember(member)

	if g.Member != member {
		t.Errorf("expected member %v, got %v", member, g.Member)
	}
}

func TestHasMember(t *testing.T) {
	tests := []struct {
		name   string
		member *Member
		want   bool
	}{
		{"nil member", nil, false},
		{"empty ID member", &Member{ID: ""}, false},
		{"valid member", &Member{ID: "m-1"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Goal{Member: tt.member}
			if got := g.HasMember(); got != tt.want {
				t.Errorf("HasMember() = %t; want %t", got, tt.want)
			}
		})
	}
}

func TestAttachChild(t *testing.T) {
	t.Run("successful attachment", func(t *testing.T) {
		parent := Goal{ID: "parent-id"}
		child := Goal{ID: "child-id"}

		err := parent.AttachChild(&child, LinkRequired, "Requires completing child first")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(parent.Children) != 1 {
			t.Errorf("expected 1 child, got %d", len(parent.Children))
		} else if parent.Children[0].ID != "child-id" {
			t.Errorf("expected child ID 'child-id', got %q", parent.Children[0].ID)
		}

		if child.Parent == nil {
			t.Fatalf("expected child to have a parent link set")
		}
		if child.Parent.TargetID != "parent-id" {
			t.Errorf("expected parent link target ID 'parent-id', got %q", child.Parent.TargetID)
		}
		if child.Parent.Relationship != LinkRequired {
			t.Errorf("expected link relationship %q, got %q", LinkRequired, child.Parent.Relationship)
		}
		if child.Parent.Description != "Requires completing child first" {
			t.Errorf("expected link description 'Requires completing child first', got %q", child.Parent.Description)
		}
	})

	t.Run("default link relationship", func(t *testing.T) {
		parent := Goal{ID: "parent-id"}
		child := Goal{ID: "child-id"}

		err := parent.AttachChild(&child, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if child.Parent.Relationship != LinkOptional {
			t.Errorf("expected default relationship %q, got %q", LinkOptional, child.Parent.Relationship)
		}
	})

	t.Run("missing child ID error", func(t *testing.T) {
		parent := Goal{ID: "parent-id"}
		child := Goal{ID: ""}

		err := parent.AttachChild(&child, "", "")
		if err == nil {
			t.Errorf("expected error for missing child ID, got nil")
		}
	})

	t.Run("cannot attach to itself error", func(t *testing.T) {
		parent := Goal{ID: "parent-id"}

		err := parent.AttachChild(&parent, "", "")
		if err == nil {
			t.Errorf("expected error for attaching itself, got nil")
		}
	})

	t.Run("duplicate child ignored", func(t *testing.T) {
		parent := Goal{ID: "parent-id"}
		child := Goal{ID: "child-id"}

		_ = parent.AttachChild(&child, "", "")
		err := parent.AttachChild(&child, "", "")
		if err != nil {
			t.Fatalf("unexpected error on duplicate attach: %v", err)
		}

		if len(parent.Children) != 1 {
			t.Errorf("expected only 1 child (duplicate ignored), got %d", len(parent.Children))
		}
	})
}

func TestRemoveChild(t *testing.T) {
	t.Run("successful removal", func(t *testing.T) {
		parent := Goal{ID: "parent-id"}
		child := Goal{ID: "child-id"}

		_ = parent.AttachChild(&child, "", "")
		err := parent.DetachChild(&child)
		if err != nil {
			t.Fatalf("unexpected error on remove: %v", err)
		}

		if len(parent.Children) != 0 {
			t.Errorf("expected 0 children after removal, got %d", len(parent.Children))
		}
		if child.Parent != nil {
			t.Errorf("expected child parent link to be nil, got %v", child.Parent)
		}
	})

	t.Run("missing child ID error", func(t *testing.T) {
		parent := Goal{ID: "parent-id"}
		child := Goal{ID: ""}

		err := parent.DetachChild(&child)
		if err == nil {
			t.Errorf("expected error for missing child ID, got nil")
		}
	})

	t.Run("cannot remove from itself error", func(t *testing.T) {
		parent := Goal{ID: "parent-id"}

		err := parent.DetachChild(&parent)
		if err == nil {
			t.Errorf("expected error for removing itself, got nil")
		}
	})
}

func TestGoalOptionsHelpers(t *testing.T) {
	t.Run("CanBeRoot", func(t *testing.T) {
		tests := []struct {
			name    string
			options GoalOptions
			project *Project
			want    bool
		}{
			{"nil project", GoalOptions{}, nil, false},
			{"project has root", GoalOptions{}, &Project{RootGoalID: "root-1"}, false},
			{"options has parent", GoalOptions{ParentID: "parent-1"}, &Project{}, false},
			{"can be root", GoalOptions{}, &Project{}, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.options.CanBeRoot(tt.project); got != tt.want {
					t.Errorf("CanBeRoot() = %t; want %t", got, tt.want)
				}
			})
		}
	})

	t.Run("WantsRoot", func(t *testing.T) {
		if !(&GoalOptions{}).WantsRoot() {
			t.Errorf("WantsRoot() expected true when ParentID is empty")
		}
		if (&GoalOptions{ParentID: "parent-1"}).WantsRoot() {
			t.Errorf("WantsRoot() expected false when ParentID is set")
		}
	})

	t.Run("IsOrphan", func(t *testing.T) {
		tests := []struct {
			name    string
			options GoalOptions
			project *Project
			want    bool
		}{
			{"parent exists (not orphan)", GoalOptions{ParentID: "parent-1"}, &Project{}, false},
			{"wants root and project empty (not orphan)", GoalOptions{}, &Project{}, false},
			{"wants root but project has root (is orphan)", GoalOptions{}, &Project{RootGoalID: "root-1"}, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.options.IsOrphan(tt.project); got != tt.want {
					t.Errorf("IsOrphan() = %t; want %t", got, tt.want)
				}
			})
		}
	})
}
