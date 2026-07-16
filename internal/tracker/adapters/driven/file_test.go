package driven_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"sattchel/internal/tracker/adapters/driven"
	"sattchel/internal/tracker/core"
)

func TestFileStorageTransaction(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "tracker.json")

	storage := driven.NewFileStorage(dbPath, nil)
	ctx := context.Background()

	// 1. Test Successful Transaction (Commit)
	err := storage.Transaction(ctx, func(txCtx context.Context) error {
		p, err := storage.CreateProject(txCtx, &core.Project{Label: "Project A"})
		if err != nil {
			return err
		}
		_, err = storage.CreateGoal(txCtx, p.ID, &core.Goal{Name: "Goal A", ProjectID: p.ID})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected transaction to succeed: %v", err)
	}

	// Verify the database has the project and goal
	projects, err := storage.GetProjects(ctx)
	if err != nil {
		t.Fatalf("failed to get projects: %v", err)
	}
	if len(projects) != 1 || projects[0].Label != "Project A" {
		t.Errorf("expected 1 project labeled 'Project A', got %v", projects)
	}

	// 2. Test Transaction Rollback
	projID := projects[0].ID
	err = storage.Transaction(ctx, func(txCtx context.Context) error {
		_, err := storage.CreateGoal(txCtx, projID, &core.Goal{Name: "Goal B", ProjectID: projID})
		if err != nil {
			return err
		}
		// Intentionally fail the transaction
		return errors.New("simulated database write failure")
	})
	if err == nil {
		t.Fatal("expected transaction to return error")
	}

	// Verify "Goal B" was NOT created (rolled back)
	goals, err := storage.GetGoals(ctx, projID)
	if err != nil {
		t.Fatalf("failed to get goals: %v", err)
	}
	// "Goal A" should be the only goal
	if len(goals) != 1 || goals[0].Name != "Goal A" {
		t.Errorf("expected only 1 goal 'Goal A' after rollback, got %v", goals)
	}

	// Read the file from disk directly to verify "Goal B" is not on disk either
	diskStorage := driven.NewFileStorage(dbPath, nil)
	diskGoals, err := diskStorage.GetGoals(ctx, projID)
	if err != nil {
		t.Fatalf("failed to get goals from disk: %v", err)
	}
	if len(diskGoals) != 1 || diskGoals[0].Name != "Goal A" {
		t.Errorf("expected disk to only have 'Goal A', got %v", diskGoals)
	}
}
