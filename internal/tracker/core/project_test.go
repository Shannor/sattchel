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

func TestSetRoot(t *testing.T) {
	p := Project{}
	g := Goal{ID: "root-goal-1"}
	p.SetRoot(g)

	if p.RootGoalID != "root-goal-1" {
		t.Errorf("expected RootGoalID 'root-goal-1', got %q", p.RootGoalID)
	}
}
