package driving

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"sattchel/internal/printer"
	"sattchel/internal/tracker/adapters/driven"
	"sattchel/internal/tracker/core"
	"strings"
	"testing"

	"charm.land/huh/v2"
	"github.com/spf13/viper"
)

func TestGoalsCLI(t *testing.T) {
	t.Run("delete leaf goal", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "tracker.json")
		repo := driven.NewFileStorage(dbPath, nil)
		service := core.NewService(repo)
		v := viper.New()
		configPath := filepath.Join(tempDir, "config.yml")
		v.SetConfigFile(configPath)
		v.Set("tracker", map[string]any{})
		_ = v.WriteConfigAs(configPath)
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
	})

	t.Run("recursive delete removes descendants and clears current goal", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "tracker.json")
		repo := driven.NewFileStorage(dbPath, nil)
		service := core.NewService(repo)
		v := viper.New()
		configPath := filepath.Join(tempDir, "config.yml")
		v.SetConfigFile(configPath)
		v.Set("tracker", map[string]any{})
		_ = v.WriteConfigAs(configPath)
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
		parentGoal, err := service.CreateGoal(context.Background(), project.ID, "Parent Goal", core.GoalOptions{ParentID: rootGoal.ID})
		if err != nil {
			t.Fatalf("failed to create parent goal: %v", err)
		}
		childGoal, err := service.CreateGoal(context.Background(), project.ID, "Child Goal", core.GoalOptions{ParentID: parentGoal.ID})
		if err != nil {
			t.Fatalf("failed to create child goal: %v", err)
		}
		if err := cfg.SetCurrentGoalID(childGoal.ID); err != nil {
			t.Fatalf("failed to set current goal: %v", err)
		}

		originalConfirmRecursiveGoalDelete := confirmRecursiveGoalDelete
		confirmRecursiveGoalDelete = func(goal *core.Goal) (bool, error) {
			if goal == nil {
				t.Fatal("expected goal to confirm")
			}
			if goal.Name != parentGoal.Name {
				t.Fatalf("expected goal name %q, got %q", parentGoal.Name, goal.Name)
			}
			if !goal.HasChildren() {
				t.Fatal("expected goal to report children")
			}
			return true, nil
		}
		defer func() { confirmRecursiveGoalDelete = originalConfirmRecursiveGoalDelete }()

		buf := new(bytes.Buffer)
		writer := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, writer)
		buf.Reset()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"delete", parentGoal.ID, "--projectId", project.ID, "-r"})
		err = cmd.ExecuteContext(context.Background())
		if err != nil {
			t.Fatalf("recursive delete failed: %v", err)
		}
		out := stripANSI(buf.String())
		if !strings.Contains(out, "and its descendants deleted successfully") {
			t.Fatalf("unexpected output from recursive delete: %q", out)
		}

		remainingGoals, err := service.GetGoals(context.Background(), project.ID)
		if err != nil {
			t.Fatalf("failed to get goals: %v", err)
		}
		if len(remainingGoals) != 1 || remainingGoals[0].ID != rootGoal.ID {
			t.Fatalf("expected only the root goal to remain, got %+v", remainingGoals)
		}
		if cfg.CurrentGoalID() != "" {
			t.Fatalf("expected current goal to be cleared, got %q", cfg.CurrentGoalID())
		}
	})

	t.Run("recursive delete confirmation no does not delete", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "tracker.json")
		repo := driven.NewFileStorage(dbPath, nil)
		service := core.NewService(repo)
		v := viper.New()
		configPath := filepath.Join(tempDir, "config.yml")
		v.SetConfigFile(configPath)
		v.Set("tracker", map[string]any{})
		_ = v.WriteConfigAs(configPath)
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
		parentGoal, err := service.CreateGoal(context.Background(), project.ID, "Parent Goal", core.GoalOptions{ParentID: rootGoal.ID})
		if err != nil {
			t.Fatalf("failed to create parent goal: %v", err)
		}
		_, err = service.CreateGoal(context.Background(), project.ID, "Child Goal", core.GoalOptions{ParentID: parentGoal.ID})
		if err != nil {
			t.Fatalf("failed to create child goal: %v", err)
		}

		originalConfirmRecursiveGoalDelete := confirmRecursiveGoalDelete
		confirmRecursiveGoalDelete = func(goal *core.Goal) (bool, error) {
			return false, nil
		}
		defer func() { confirmRecursiveGoalDelete = originalConfirmRecursiveGoalDelete }()

		buf := new(bytes.Buffer)
		writer := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, writer)
		buf.Reset()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"delete", parentGoal.ID, "--projectId", project.ID, "-r"})
		err = cmd.ExecuteContext(context.Background())
		if err != nil {
			t.Fatalf("expected recursive delete cancellation without error, got %v", err)
		}
		out := stripANSI(buf.String())
		if !strings.Contains(out, "Recursive delete cancelled") {
			t.Fatalf("unexpected output from recursive delete cancellation: %q", out)
		}

		remainingGoals, err := service.GetGoals(context.Background(), project.ID)
		if err != nil {
			t.Fatalf("failed to get goals: %v", err)
		}
		if len(remainingGoals) != 3 {
			t.Fatalf("expected all goals to remain, got %+v", remainingGoals)
		}
	})

	t.Run("recursive delete confirmation abort does not delete", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "tracker.json")
		repo := driven.NewFileStorage(dbPath, nil)
		service := core.NewService(repo)
		v := viper.New()
		configPath := filepath.Join(tempDir, "config.yml")
		v.SetConfigFile(configPath)
		v.Set("tracker", map[string]any{})
		_ = v.WriteConfigAs(configPath)
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
		parentGoal, err := service.CreateGoal(context.Background(), project.ID, "Parent Goal", core.GoalOptions{ParentID: rootGoal.ID})
		if err != nil {
			t.Fatalf("failed to create parent goal: %v", err)
		}
		_, err = service.CreateGoal(context.Background(), project.ID, "Child Goal", core.GoalOptions{ParentID: parentGoal.ID})
		if err != nil {
			t.Fatalf("failed to create child goal: %v", err)
		}

		originalConfirmRecursiveGoalDelete := confirmRecursiveGoalDelete
		confirmRecursiveGoalDelete = func(goal *core.Goal) (bool, error) {
			return false, huh.ErrUserAborted
		}
		defer func() { confirmRecursiveGoalDelete = originalConfirmRecursiveGoalDelete }()

		buf := new(bytes.Buffer)
		writer := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, writer)
		buf.Reset()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"delete", parentGoal.ID, "--projectId", project.ID, "-r"})
		err = cmd.ExecuteContext(context.Background())
		if !errors.Is(err, huh.ErrUserAborted) {
			t.Fatalf("expected huh.ErrUserAborted, got %v", err)
		}

		remainingGoals, err := service.GetGoals(context.Background(), project.ID)
		if err != nil {
			t.Fatalf("failed to get goals: %v", err)
		}
		if len(remainingGoals) != 3 {
			t.Fatalf("expected all goals to remain, got %+v", remainingGoals)
		}
	})

	t.Run("delete without id chooses goal interactively", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "tracker.json")
		repo := driven.NewFileStorage(dbPath, nil)
		service := core.NewService(repo)
		v := viper.New()
		configPath := filepath.Join(tempDir, "config.yml")
		v.SetConfigFile(configPath)
		v.Set("tracker", map[string]any{})
		_ = v.WriteConfigAs(configPath)
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

		originalChooseGoalForDelete := chooseGoalForDelete
		chooseGoalForDelete = func(goals []core.Goal, title string, currentGoalID string, filterFn func(*core.Goal) bool, validateFn func(string) error) (string, error) {
			if len(goals) != 2 {
				t.Fatalf("expected 2 goals to choose from, got %d", len(goals))
			}
			if title != "Select Goal to Delete" {
				t.Fatalf("unexpected chooser title: %q", title)
			}
			return childGoal.ID, nil
		}
		defer func() { chooseGoalForDelete = originalChooseGoalForDelete }()

		buf := new(bytes.Buffer)
		writer := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, writer)
		buf.Reset()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"delete", "--projectId", project.ID})
		err = cmd.ExecuteContext(context.Background())
		if err != nil {
			t.Fatalf("interactive delete failed: %v", err)
		}
		out := stripANSI(buf.String())
		if !strings.Contains(out, "deleted successfully") {
			t.Fatalf("unexpected output from interactive delete: %q", out)
		}

		remainingGoals, err := service.GetGoals(context.Background(), project.ID)
		if err != nil {
			t.Fatalf("failed to get goals: %v", err)
		}
		if len(remainingGoals) != 1 || remainingGoals[0].ID != rootGoal.ID {
			t.Fatalf("expected only the root goal to remain, got %+v", remainingGoals)
		}
	})
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
