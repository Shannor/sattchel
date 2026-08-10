package driving

import (
	"bytes"
	"context"
	"path/filepath"
	"sattchel/internal/tracker/adapters/driven"
	"sattchel/internal/tracker/core"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestGoalsCLI(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "tracker.json")
	repo := driven.NewFileStorage(dbPath, nil)
	service := core.NewService(repo)
	v := viper.New()
	cfg, err := LoadConfig(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	project, err := service.CreateProject(context.Background(), "Proj A", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	rootGoal, err := service.CreateGoal(context.Background(), project.ID, "Root Goal", core.GoalOptions{})
	if err != nil {
		t.Fatalf("failed to create root goal: %v", err)
	}
	childGoal, err := service.CreateGoal(context.Background(), project.ID, "Child Goal", core.GoalOptions{ParentID: rootGoal.ID})
	if err != nil {
		t.Fatalf("failed to create child goal: %v", err)
	}

	cmd := goals(service, cfg)
	executeCmd := func(args ...string) (string, error) {
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		return buf.String(), err
	}

	out, err := executeCmd("delete", childGoal.ID, "--projectId", project.ID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if !strings.Contains(out, "deleted successfully") {
		t.Fatalf("unexpected output from delete: %q", out)
	}

	remainingGoals, err := service.GetGoals(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("failed to get goals: %v", err)
	}
	if len(remainingGoals) != 1 || remainingGoals[0].ID != rootGoal.ID {
		t.Fatalf("expected only the root goal to remain, got %+v", remainingGoals)
	}

	_, err = executeCmd("delete", rootGoal.ID, "--projectId", project.ID)
	if err == nil {
		t.Fatal("expected deleting the root goal to fail")
	}
}
