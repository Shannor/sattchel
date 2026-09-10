package core

import "testing"

func TestNormalizedLabel(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  string
	}{
		{"empty label", "", ""},
		{"all spaces", "   ", ""},
		{"mixed case and spaces", "  My Awesome Project  ", "my awesome project"},
		{"special characters", "Project-A!", "project-a!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Project{Label: tt.label}
			if got := p.NormalizedLabel(); got != tt.want {
				t.Errorf("NormalizedLabel() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizedStatus(t *testing.T) {
	t.Run("defaults empty status to draft", func(t *testing.T) {
		p := Project{}
		if got := p.NormalizedStatus(); got != ProjectDraft {
			t.Errorf("NormalizedStatus() = %q; want %q", got, ProjectDraft)
		}
	})

	t.Run("preserves explicit status", func(t *testing.T) {
		p := Project{Status: ProjectInProgress}
		if got := p.NormalizedStatus(); got != ProjectInProgress {
			t.Errorf("NormalizedStatus() = %q; want %q", got, ProjectInProgress)
		}
	})
}

func TestSetStatus(t *testing.T) {
	t.Run("empty status becomes draft", func(t *testing.T) {
		p := Project{}
		p.SetStatus("")
		if p.Status != ProjectDraft {
			t.Errorf("expected Status %q, got %q", ProjectDraft, p.Status)
		}
	})

	t.Run("explicit status is stored", func(t *testing.T) {
		p := Project{}
		p.SetStatus(ProjectComplete)
		if p.Status != ProjectComplete {
			t.Errorf("expected Status %q, got %q", ProjectComplete, p.Status)
		}
	})
}

func TestSetRoot(t *testing.T) {
	p := Project{}
	g := Goal{ID: "root-goal-1"}
	p.SetRoot(g)

	if p.RootGoalID != "root-goal-1" {
		t.Errorf("expected RootGoalID 'root-goal-1', got %q", p.RootGoalID)
	}
}
