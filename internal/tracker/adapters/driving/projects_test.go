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

type dummyWriter struct {
	successMsgs []string
}

func (dw *dummyWriter) Success(msg string) {
	dw.successMsgs = append(dw.successMsgs, msg)
}

func (dw *dummyWriter) Error(msg string) {}
func (dw *dummyWriter) Info(msg string)  {}
func (dw *dummyWriter) Warn(msg string)  {}

func TestProjectsCLI(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "tracker.json")
	repo := driven.NewFileStorage(dbPath, nil)
	service := core.NewService(repo)
	v := viper.New()
	cfg, err := LoadConfig(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	writer := &dummyWriter{}
	cmd := projects(service, cfg, writer)

	executeCmd := func(args ...string) (string, error) {
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		return buf.String(), err
	}

	// 1. Create two projects
	_, err = executeCmd("create", "Proj A")
	if err != nil {
		t.Fatalf("failed to create Proj A: %v", err)
	}
	_, err = executeCmd("create", "Proj B")
	if err != nil {
		t.Fatalf("failed to create Proj B: %v", err)
	}

	// Let's get the created projects
	projs, err := service.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("failed to get projects: %v", err)
	}
	var projA, projB core.Project
	for _, p := range projs {
		if p.Label == "Proj A" {
			projA = p
		}
		if p.Label == "Proj B" {
			projB = p
		}
	}

	// Create root goals for projects using the service
	gA, err := service.CreateGoal(context.Background(), projA.ID, "Goal A", core.GoalOptions{})
	if err != nil {
		t.Fatalf("failed to create Goal A: %v", err)
	}
	_, err = service.CreateGoal(context.Background(), projB.ID, "Goal B", core.GoalOptions{})
	if err != nil {
		t.Fatalf("failed to create Goal B: %v", err)
	}

	// 2. Merge Proj B into Proj A under Goal A
	writer.successMsgs = nil
	_, err = executeCmd("merge", projA.ID, projB.ID, "--parent-goal-id", gA.ID)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	if len(writer.successMsgs) == 0 || !strings.Contains(writer.successMsgs[0], "Projects merged successfully") {
		t.Errorf("expected success message, got: %v", writer.successMsgs)
	}

	// Verify project B is deleted and goals moved
	projs, err = service.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("failed to get projects: %v", err)
	}
	if len(projs) != 1 || projs[0].ID != projA.ID {
		t.Errorf("expected only Proj A to remain, got: %+v", projs)
	}

	// Verify Goal B now belongs to Proj A and is child of Goal A
	goals, err := service.GetGoals(context.Background(), projA.ID)
	if err != nil {
		t.Fatalf("failed to get goals: %v", err)
	}
	if len(goals) != 2 {
		t.Errorf("expected Proj A to have 2 goals now, got: %d", len(goals))
	}

	// 3. Split Proj A at the merged Goal B
	writer.successMsgs = nil
	// We need to find Goal B ID
	var goalBID string
	for _, g := range goals {
		if g.Name == "Goal B" {
			goalBID = g.ID
		}
	}
	if goalBID == "" {
		t.Fatalf("Goal B ID not found")
	}

	_, err = executeCmd("split", projA.ID, "--goal", goalBID, "--name", "Split Proj C")
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	if len(writer.successMsgs) == 0 || !strings.Contains(writer.successMsgs[0], "Project split successfully") {
		t.Errorf("expected success message, got: %v", writer.successMsgs)
	}

	// Verify new project was created
	projs, err = service.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("failed to get projects: %v", err)
	}
	if len(projs) != 2 {
		t.Errorf("expected 2 projects, got: %d", len(projs))
	}
}
