package driving

import (
	"bytes"
	"context"
	"path/filepath"
	"sattchel/internal/tracker/adapters/driven"
	"sattchel/internal/tracker/core"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	configPath := filepath.Join(tempDir, "config.yml")
	v.SetConfigFile(configPath)
	v.Set("tracker", map[string]any{})
	_ = v.WriteConfigAs(configPath)

	cfg, err := LoadConfig(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	writer := &dummyWriter{}
	cmd := projects(service, cfg, writer)

	executeCmd := func(args ...string) (string, error) {
		var resetFlags func(*cobra.Command)
		resetFlags = func(c *cobra.Command) {
			c.Flags().VisitAll(func(f *pflag.Flag) {
				_ = f.Value.Set(f.DefValue)
			})
			for _, sub := range c.Commands() {
				resetFlags(sub)
			}
		}
		resetFlags(cmd)

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

	_, err = executeCmd("split", projA.ID, "--goal", goalBID, "--new", "Split Proj C")
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

	// 4. Move Goal B from Split Proj C back to Proj A under Goal A
	writer.successMsgs = nil
	// Find Split Proj C
	var projC core.Project
	for _, p := range projs {
		if p.Label == "Split Proj C" {
			projC = p
		}
	}
	if projC.ID == "" {
		t.Fatalf("Split Proj C not found")
	}

	_, err = executeCmd("split", projC.ID, "--goal", goalBID, "--to", projA.ID, "--parent", gA.ID)
	if err != nil {
		t.Fatalf("split move-goal failed: %v", err)
	}

	if len(writer.successMsgs) == 0 || !strings.Contains(writer.successMsgs[0], "Goal moved successfully") {
		t.Errorf("expected success message, got: %v", writer.successMsgs)
	}

	// Verify goals in Proj A
	goals, err = service.GetGoals(context.Background(), projA.ID)
	if err != nil {
		t.Fatalf("failed to get goals: %v", err)
	}
	if len(goals) != 2 {
		t.Errorf("expected Proj A to have 2 goals after move, got: %d", len(goals))
	}

	// 5. Verify error when both --new and --to are provided
	_, err = executeCmd("split", projC.ID, "--goal", goalBID, "--to", projA.ID, "--new", "Invalid Proj")
	if err == nil || !strings.Contains(err.Error(), "cannot specify both --new and --to") {
		t.Errorf("expected error containing 'cannot specify both --new and --to', got: %v", err)
	}
}
