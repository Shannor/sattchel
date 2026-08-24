package driving

import (
	"bytes"
	"context"
	"path/filepath"
	"sattchel/internal/printer"
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

	buf := new(bytes.Buffer)
	writer := printer.NewStyleWriterWithWriters(buf, buf)
	cmd := goals(service, cfg, writer)
	executeCmd := func(args ...string) (string, error) {
		buf.Reset()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		return stripANSI(buf.String()), err
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

func TestListGoalsLinkRelationship(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "tracker.json")
	repo := driven.NewFileStorage(dbPath, nil)
	service := core.NewService(repo)
	v := viper.New()
	cfg, err := LoadConfig(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	project, err := service.CreateProject(context.Background(), "Proj B", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	rootGoal, err := service.CreateGoal(context.Background(), project.ID, "Root Goal", core.GoalOptions{})
	if err != nil {
		t.Fatalf("failed to create root goal: %v", err)
	}
	_, err = service.CreateGoal(context.Background(), project.ID, "Req Child Goal", core.GoalOptions{
		ParentID:         rootGoal.ID,
		LinkRelationship: core.LinkRequired,
	})
	if err != nil {
		t.Fatalf("failed to create required child goal: %v", err)
	}
	_, err = service.CreateGoal(context.Background(), project.ID, "Opt Child Goal", core.GoalOptions{
		ParentID:         rootGoal.ID,
		LinkRelationship: core.LinkOptional,
	})
	if err != nil {
		t.Fatalf("failed to create optional child goal: %v", err)
	}

	buf := new(bytes.Buffer)
	writer := printer.NewStyleWriterWithWriters(buf, buf)
	cmd := goals(service, cfg, writer)

	buf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list", "--projectId", project.ID})
	err = cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	out := stripANSI(buf.String())
	if !strings.Contains(out, "root") {
		t.Errorf("expected list output to contain 'root', got:\n%s", out)
	}
	if !strings.Contains(out, "required") {
		t.Errorf("expected list output to contain 'required', got:\n%s", out)
	}
	if !strings.Contains(out, "optional") {
		t.Errorf("expected list output to contain 'optional', got:\n%s", out)
	}

	buf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list", "--projectId", project.ID, "--relationship", "required", "--flat"})
	err = cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("list command with relationship filter failed: %v", err)
	}

	flatOut := stripANSI(buf.String())
	if !strings.Contains(flatOut, "Req Child Goal") {
		t.Errorf("expected flat filtered output to contain 'Req Child Goal', got:\n%s", flatOut)
	}
	if strings.Contains(flatOut, "Opt Child Goal") {
		t.Errorf("expected flat filtered output to NOT contain 'Opt Child Goal', got:\n%s", flatOut)
	}
}
